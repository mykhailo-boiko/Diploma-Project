
from __future__ import annotations

import json
import logging
from contextlib import asynccontextmanager
from typing import Any

import redis.asyncio as aioredis
import uvicorn
from fastapi import Depends, FastAPI, HTTPException, Request, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, Response
from prometheus_client import CONTENT_TYPE_LATEST, generate_latest

from auth import AuthError, UserClaims, validate_token
from budget import SessionBudget
from config import HOST, PORT, REDIS_URL
from context_store import ContextStore
from llm import GeminiUnavailableError, chat_completion
from loop_guard import LoopGuard
from mcp_client import MCPClient
from observability import instrument_fastapi, instrument_httpx, logfire_span, setup_logfire
from plan_store import PlanStore

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)

setup_logfire("mcp-host")
instrument_httpx()

mcp = MCPClient()

_plan_store: PlanStore | None = None
_context_store: ContextStore | None = None
_budget = SessionBudget()
_loop_guard = LoopGuard()

@asynccontextmanager
async def lifespan(app: FastAPI):

    global _plan_store, _context_store  # noqa: PLW0603

    redis_client = aioredis.from_url(REDIS_URL, decode_responses=False)
    _plan_store = PlanStore(redis_client)
    _context_store = ContextStore(redis_client)
    cache_redis = aioredis.from_url(REDIS_URL, decode_responses=True)
    mcp.attach_cache(cache_redis)
    _budget.attach(cache_redis)
    _loop_guard.attach(cache_redis)
    logger.info("Redis connected: %s", REDIS_URL)

    await mcp.connect()
    logger.info("MCP Host started — tools: %d", len(mcp.tools))
    yield
    await mcp.close()
    await redis_client.aclose()
    logger.info("MCP Host stopped")

app = FastAPI(title="MCP Host", version="0.1.0", lifespan=lifespan)
instrument_fastapi(app)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000", "http://localhost:5173"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/metrics")
async def metrics() -> Response:

    return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)

@app.get("/health")
async def health() -> JSONResponse:

    connected = mcp._session is not None
    tools_count = len(mcp.tools)
    redis_ok = await _plan_store.health() if _plan_store else False
    return JSONResponse({
        "status": "ok" if connected else "disconnected",
        "tools": tools_count,
        "redis": "ok" if redis_ok else "disconnected",
    })

async def _require_jwt(request: Request) -> UserClaims:

    header = request.headers.get("authorization") or ""
    if not header.lower().startswith("bearer "):
        raise HTTPException(status_code=401, detail="missing or malformed Authorization header")
    token = header.split(" ", 1)[1].strip()
    try:
        return validate_token(token)
    except AuthError as exc:
        raise HTTPException(status_code=401, detail=str(exc))

@app.get("/api/v1/mcp/budget/{session_id}")
async def get_budget(
    session_id: str,
    user_id: str | None = None,
    claims: UserClaims = Depends(_require_jwt),
) -> JSONResponse:

    effective_user_id = user_id or claims.user_id
    status = await _budget.get_status(session_id, effective_user_id)
    return JSONResponse({
        "session_used": status.session_used,
        "session_cap": status.session_cap,
        "user_hour_used": status.user_hour_used,
        "user_hour_cap": status.user_hour_cap,
        "exceeded": status.exceeded,
    })

@app.get("/api/v1/mcp/plans/{session_id}")
async def list_plans(
    session_id: str,
    claims: UserClaims = Depends(_require_jwt),
) -> JSONResponse:

    if _plan_store is None:
        return JSONResponse({"error": "plan store not available"}, status_code=503)
    plans = await _plan_store.list_by_session(session_id)
    return JSONResponse({"plans": plans})

@app.get("/api/v1/mcp/plans/{session_id}/{plan_id}")
async def get_plan(session_id: str, plan_id: str) -> JSONResponse:

    if _plan_store is None:
        return JSONResponse({"error": "plan store not available"}, status_code=503)
    plan = await _plan_store.get(session_id, plan_id)
    if plan is None:
        return JSONResponse({"error": "plan not found"}, status_code=404)
    return JSONResponse(plan.model_dump(mode="json"))

@app.websocket("/ws/chat")
async def websocket_chat(ws: WebSocket) -> None:

    token = ws.query_params.get("token", "")
    if not token:
        await ws.close(code=4001, reason="missing token")
        return

    try:
        claims = validate_token(token)
    except AuthError as exc:
        await ws.close(code=4001, reason=str(exc))
        return

    await ws.accept()
    logger.info("WebSocket connected: user=%s role=%s", claims.user_id, claims.role)

    session_id = claims.user_id

    history: list[Any] = []
    if _context_store:
        history = await _context_store.load(session_id)
        if history:
            logger.info("Restored %d history turns for session %s", len(history), session_id)

    try:
        await _send(ws, "system", f"Connected as {claims.email} (role: {claims.role}). How can I help?")

        while True:
            raw = await ws.receive_text()
            msg = _parse_message(raw)
            if msg is None:
                await _send(ws, "error", "Invalid message format. Send JSON: {\"message\": \"your text\"}")
                continue

            user_text = msg.get("message", "").strip()
            if not user_text:
                await _send(ws, "error", "Empty message")
                continue

            if user_text.lower() in ("/clear", "/reset"):
                history = []
                if _context_store:
                    await _context_store.delete(session_id)
                if _budget:
                    await _budget.reset(session_id)
                if _loop_guard:
                    await _loop_guard.reset(session_id)
                await _send(ws, "system", "Conversation history, token budget, and loop counters cleared.")
                continue

            logger.info("User %s: %s", claims.user_id, user_text[:100])

            await _send(ws, "thinking", "Processing your request...")

            async def on_stream(msg_type: str, content: str) -> None:
                await _send(ws, msg_type, content)

            trace_id = msg.get("trace_id") if isinstance(msg, dict) else None

            try:
                with logfire_span(
                    "chat.turn",
                    session_id=session_id,
                    user_id=claims.user_id,
                    user_email=claims.email,
                    user_role=claims.role,
                    trace_id=trace_id,
                    message_preview=user_text[:200],
                ) as span:
                    response_text, updated_history = await chat_completion(
                        mcp=mcp,
                        history=history,
                        user_message=user_text,
                        user_role=claims.role,
                        session_id=session_id,
                        user_id=claims.user_id,
                        trace_id=trace_id,
                        plan_store=_plan_store,
                        on_stream=on_stream,
                        budget=_budget,
                        loop_guard=_loop_guard,
                    )
                    span.set_attribute("response_length", len(response_text))
                history = updated_history
                if _context_store:
                    await _context_store.save(session_id, history)
                await _send(ws, "message", response_text)
            except GeminiUnavailableError as exc:
                logger.warning("Gemini upstream unavailable: %s", exc)
                await _send(ws, "error", str(exc))
            except Exception as exc:
                logger.error("Chat completion error: %s", exc, exc_info=True)
                await _send(ws, "error", f"Failed to process request: {exc}")

    except WebSocketDisconnect:
        logger.info("WebSocket disconnected: user=%s", claims.user_id)
    except Exception as exc:
        logger.error("WebSocket error: %s", exc, exc_info=True)
        try:
            await _send(ws, "error", f"Internal error: {exc}")
            await ws.close(code=1011)
        except Exception:
            pass

async def _send(ws: WebSocket, msg_type: str, content: str) -> None:

    await ws.send_json({"type": msg_type, "content": content})

def _parse_message(raw: str) -> dict[str, Any] | None:

    try:
        data = json.loads(raw)
        if isinstance(data, dict):
            return data
    except (json.JSONDecodeError, ValueError):
        pass
    return None

def main() -> None:

    uvicorn.run("main:app", host=HOST, port=PORT, log_level="info")

if __name__ == "__main__":
    main()
