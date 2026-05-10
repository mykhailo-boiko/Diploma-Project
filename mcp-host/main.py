"""MCP Host — FastAPI WebSocket server with Gemini LLM and MCP tool execution."""

from __future__ import annotations

import json
import logging
from contextlib import asynccontextmanager
from typing import Any

import redis.asyncio as aioredis
import uvicorn
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from auth import AuthError, validate_token
from config import HOST, PORT, REDIS_URL
from context_store import ContextStore
from llm import chat_completion
from mcp_client import MCPClient
from plan_store import PlanStore

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)

mcp = MCPClient()

_plan_store: PlanStore | None = None
_context_store: ContextStore | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Start/stop the MCP Client and Redis with the application."""
    global _plan_store, _context_store  # noqa: PLW0603

    redis_client = aioredis.from_url(REDIS_URL, decode_responses=False)
    _plan_store = PlanStore(redis_client)
    _context_store = ContextStore(redis_client)
    cache_redis = aioredis.from_url(REDIS_URL, decode_responses=True)
    mcp.attach_cache(cache_redis)
    logger.info("Redis connected: %s", REDIS_URL)

    await mcp.connect()
    logger.info("MCP Host started — tools: %d", len(mcp.tools))
    yield
    await mcp.close()
    await redis_client.aclose()
    logger.info("MCP Host stopped")


app = FastAPI(title="MCP Host", version="0.1.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000", "http://localhost:5173"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
async def health() -> JSONResponse:
    """Health check endpoint."""
    connected = mcp._session is not None
    tools_count = len(mcp.tools)
    redis_ok = await _plan_store.health() if _plan_store else False
    return JSONResponse({
        "status": "ok" if connected else "disconnected",
        "tools": tools_count,
        "redis": "ok" if redis_ok else "disconnected",
    })


@app.get("/api/v1/mcp/plans/{session_id}")
async def list_plans(session_id: str) -> JSONResponse:
    """List execution plans for a session."""
    if _plan_store is None:
        return JSONResponse({"error": "plan store not available"}, status_code=503)
    plans = await _plan_store.list_by_session(session_id)
    return JSONResponse({"plans": plans})


@app.get("/api/v1/mcp/plans/{session_id}/{plan_id}")
async def get_plan(session_id: str, plan_id: str) -> JSONResponse:
    """Get details of a specific execution plan."""
    if _plan_store is None:
        return JSONResponse({"error": "plan store not available"}, status_code=503)
    plan = await _plan_store.get(session_id, plan_id)
    if plan is None:
        return JSONResponse({"error": "plan not found"}, status_code=404)
    return JSONResponse(plan.model_dump(mode="json"))


@app.websocket("/ws/chat")
async def websocket_chat(ws: WebSocket) -> None:
    """WebSocket chat endpoint.

    JWT token is passed as a query parameter: /ws/chat?token=<jwt>
    Each message is a JSON object: {"message": "user text"}
    Responses are JSON objects: {"type": "message", "content": "..."} or {"type": "error", "content": "..."}
    """
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
                await _send(ws, "system", "Conversation history cleared.")
                continue

            logger.info("User %s: %s", claims.user_id, user_text[:100])

            await _send(ws, "thinking", "Processing your request...")

            async def on_stream(msg_type: str, content: str) -> None:
                await _send(ws, msg_type, content)

            try:
                response_text, updated_history = await chat_completion(
                    mcp=mcp,
                    history=history,
                    user_message=user_text,
                    user_role=claims.role,
                    session_id=session_id,
                    plan_store=_plan_store,
                    on_stream=on_stream,
                )
                history = updated_history
                if _context_store:
                    await _context_store.save(session_id, history)
                await _send(ws, "message", response_text)
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
    """Send a typed JSON message over WebSocket."""
    await ws.send_json({"type": msg_type, "content": content})


def _parse_message(raw: str) -> dict[str, Any] | None:
    """Parse incoming WebSocket message as JSON."""
    try:
        data = json.loads(raw)
        if isinstance(data, dict):
            return data
    except (json.JSONDecodeError, ValueError):
        pass
    return None


def main() -> None:
    """Run the MCP Host server."""
    uvicorn.run("main:app", host=HOST, port=PORT, log_level="info")


if __name__ == "__main__":
    main()
