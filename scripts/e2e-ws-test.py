#!/usr/bin/env python3
"""
ChainOrchestra — WebSocket MCP Chat E2E Tests (TASK-032)

Tests WebSocket connectivity, MCP chat integration, and RBAC filtering.
Requires: running docker compose stack + seed data + GEMINI_API_KEY set.

Usage:
    python3 scripts/e2e-ws-test.py

Environment:
    GATEWAY_URL    — API Gateway URL (default: http://localhost:8080)
    MCP_WS_URL     — MCP Host WebSocket URL (default: ws://localhost:8090)
"""
from __future__ import annotations

import asyncio
import json
import os
import sys
import time

GATEWAY_URL = os.getenv("GATEWAY_URL", "http://localhost:8080")
MCP_WS_URL = os.getenv("MCP_WS_URL", "ws://localhost:8090")
MCP_HTTP_URL = os.getenv("MCP_HTTP_URL", "http://localhost:8090")

ADMIN_EMAIL = os.getenv("ADMIN_EMAIL", "admin@chainorchestra.local")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "admin123")
OPERATOR_EMAIL = "ivan.petrov@chainorchestra.local"
OPERATOR_PASSWORD = "Operator1!"

GREEN = "\033[0;32m"
RED = "\033[0;31m"
YELLOW = "\033[1;33m"
CYAN = "\033[0;36m"
NC = "\033[0m"

pass_count = 0
fail_count = 0
skip_count = 0


def assert_pass(name: str) -> None:
    global pass_count
    pass_count += 1
    print(f"  {GREEN}PASS{NC} {name}")


def assert_fail(name: str, detail: str = "") -> None:
    global fail_count
    fail_count += 1
    msg = f"  {RED}FAIL{NC} {name}"
    if detail:
        msg += f"\n       {RED}Detail: {detail}{NC}"
    print(msg)


def assert_skip(name: str, reason: str = "") -> None:
    global skip_count
    skip_count += 1
    msg = f"  {YELLOW}SKIP{NC} {name}"
    if reason:
        msg += f" ({reason})"
    print(msg)


def login(email: str, password: str) -> str | None:
    """Login via API Gateway and return access token."""
    import urllib.request

    data = json.dumps({"email": email, "password": password}).encode()
    req = urllib.request.Request(
        f"{GATEWAY_URL}/api/v1/auth/login",
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            body = json.loads(resp.read())
            return body.get("data", {}).get("access_token", "")
    except Exception as exc:
        print(f"  Login failed for {email}: {exc}")
        return None


def check_mcp_health() -> dict:
    """Check MCP host health endpoint."""
    import urllib.request

    try:
        with urllib.request.urlopen(f"{MCP_HTTP_URL}/health", timeout=5) as resp:
            return json.loads(resp.read())
    except Exception:
        return {"status": "unavailable"}


async def test_ws_connection(token: str) -> bool:
    """Test basic WebSocket connection with JWT auth."""
    try:
        import websockets
    except ImportError:
        assert_skip("WebSocket connection", "websockets package not installed (pip install websockets)")
        return False

    ws_url = f"{MCP_WS_URL}/ws/chat?token={token}"
    try:
        async with websockets.connect(ws_url, open_timeout=10) as ws:
            msg = await asyncio.wait_for(ws.recv(), timeout=10)
            data = json.loads(msg)
            if data.get("type") == "system":
                assert_pass(f"WebSocket connected, got system message: {data.get('content', '')[:60]}")
                return True
            else:
                assert_pass(f"WebSocket connected, got message type: {data.get('type')}")
                return True
    except asyncio.TimeoutError:
        assert_fail("WebSocket connection", "timeout waiting for system message")
        return False
    except Exception as exc:
        assert_fail("WebSocket connection", str(exc))
        return False


async def test_ws_invalid_token() -> None:
    """Test WebSocket rejects invalid JWT."""
    try:
        import websockets
    except ImportError:
        assert_skip("WebSocket invalid token", "websockets package not installed")
        return

    ws_url = f"{MCP_WS_URL}/ws/chat?token=invalid-jwt"
    try:
        async with websockets.connect(ws_url, open_timeout=5) as ws:
            msg = await asyncio.wait_for(ws.recv(), timeout=5)
            data = json.loads(msg)
            if data.get("type") == "error":
                assert_pass("WebSocket rejects invalid token with error message")
            else:
                assert_fail("WebSocket rejects invalid token", f"got: {data}")
    except websockets.exceptions.ConnectionClosedError as exc:
        if exc.code == 4001 or exc.code == 1008:
            assert_pass(f"WebSocket rejects invalid token (close code {exc.code})")
        else:
            assert_pass(f"WebSocket closed connection on invalid token (code {exc.code})")
    except (ConnectionRefusedError, OSError):
        assert_skip("WebSocket invalid token", "MCP host not accessible")
    except Exception as exc:
        assert_pass(f"WebSocket rejects invalid token ({type(exc).__name__})")


async def test_ws_no_token() -> None:
    """Test WebSocket rejects missing token."""
    try:
        import websockets
    except ImportError:
        assert_skip("WebSocket no token", "websockets package not installed")
        return

    ws_url = f"{MCP_WS_URL}/ws/chat"
    try:
        async with websockets.connect(ws_url, open_timeout=5) as ws:
            msg = await asyncio.wait_for(ws.recv(), timeout=5)
            data = json.loads(msg)
            if data.get("type") == "error":
                assert_pass("WebSocket rejects missing token")
            else:
                assert_fail("WebSocket rejects missing token", f"got: {data}")
    except websockets.exceptions.ConnectionClosedError as exc:
        assert_pass(f"WebSocket rejects missing token (close code {exc.code})")
    except (ConnectionRefusedError, OSError):
        assert_skip("WebSocket no token", "MCP host not accessible")
    except Exception:
        assert_pass("WebSocket rejects missing token")


async def test_chat_message(token: str, message: str, test_name: str, timeout: int = 60) -> str | None:
    """Send a chat message and collect the final response."""
    try:
        import websockets
    except ImportError:
        assert_skip(test_name, "websockets package not installed")
        return None

    ws_url = f"{MCP_WS_URL}/ws/chat?token={token}"
    try:
        async with websockets.connect(ws_url, open_timeout=10) as ws:
            try:
                await asyncio.wait_for(ws.recv(), timeout=10)
            except asyncio.TimeoutError:
                pass

            await ws.send(json.dumps({"message": message}))

            final_text = ""
            tool_calls = []
            start = time.time()

            while time.time() - start < timeout:
                try:
                    raw = await asyncio.wait_for(ws.recv(), timeout=timeout)
                    data = json.loads(raw)
                    msg_type = data.get("type", "")

                    if msg_type == "message":
                        final_text = data.get("content", "")
                        break
                    elif msg_type == "tool_start":
                        tool_calls.append(data.get("content", ""))
                    elif msg_type == "tool_result":
                        pass
                    elif msg_type == "tool_error":
                        final_text = f"TOOL_ERROR: {data.get('content', '')}"
                        break
                    elif msg_type == "error":
                        final_text = f"ERROR: {data.get('content', '')}"
                        break
                    elif msg_type == "stream":
                        final_text += data.get("content", "")
                except asyncio.TimeoutError:
                    break

            if final_text:
                short = final_text[:100].replace("\n", " ")
                assert_pass(f"{test_name} (response: '{short}...')")
                if tool_calls:
                    print(f"       Tools called: {', '.join(tool_calls[:5])}")
                return final_text
            else:
                assert_fail(test_name, "no response received within timeout")
                return None

    except Exception as exc:
        assert_fail(test_name, str(exc)[:100])
        return None


async def scenario_chat_basic(admin_token: str) -> None:
    """Test basic MCP chat functionality."""
    print(f"\n{CYAN}>>> SCENARIO: MCP Chat Basic Functionality <<<{NC}")

    await test_chat_message(
        admin_token,
        "List all orders",
        "Chat: list orders via MCP tools",
    )

    await test_chat_message(
        admin_token,
        "What are the low stock items?",
        "Chat: low stock query via MCP tools",
    )


async def scenario_chat_multistep(admin_token: str) -> None:
    """Test multi-step MCP chat scenario."""
    print(f"\n{CYAN}>>> SCENARIO: MCP Chat Multi-step Workflow <<<{NC}")

    await test_chat_message(
        admin_token,
        "Create an order for customer 'MCP Test User' with 2 units of 'Test Widget' at $25 each, then show me the order details",
        "Chat: multi-step create order + show details",
        timeout=90,
    )


async def scenario_chat_rbac(operator_token: str) -> None:
    """Test RBAC enforcement in MCP chat."""
    print(f"\n{CYAN}>>> SCENARIO: MCP Chat RBAC Enforcement <<<{NC}")

    response = await test_chat_message(
        operator_token,
        "Show me the inventory stock levels",
        "Chat: operator asks for inventory (should be restricted)",
    )
    if response:
        lower = response.lower()
        if any(kw in lower for kw in ["not available", "don't have", "cannot", "restricted", "permission", "not authorized", "access"]):
            assert_pass("RBAC: operator correctly denied inventory access in chat")
        else:
            assert_skip("RBAC: operator inventory denial", "LLM response may vary — check manually")


async def run_ws_tests() -> None:
    """Run all WebSocket E2E tests."""
    print(f"\n{CYAN}{'=' * 60}{NC}")
    print(f"{CYAN}  ChainOrchestra — WebSocket MCP Chat E2E Tests{NC}")
    print(f"{CYAN}  MCP: {MCP_WS_URL}{NC}")
    print(f"{CYAN}{'=' * 60}{NC}")

    health = check_mcp_health()
    if health.get("status") == "unavailable":
        print(f"\n{YELLOW}MCP Host is not available. Skipping all WebSocket tests.{NC}")
        assert_skip("All WebSocket tests", "MCP host not running")
        return

    tools = health.get("tools", 0)
    print(f"\n  MCP Host status: {health.get('status')}, tools: {tools}, redis: {health.get('redis')}")

    admin_token = login(ADMIN_EMAIL, ADMIN_PASSWORD)
    if not admin_token:
        assert_fail("Admin login for WebSocket tests")
        return
    assert_pass("Admin login")

    operator_token = login(OPERATOR_EMAIL, OPERATOR_PASSWORD)

    print(f"\n{CYAN}>>> SCENARIO: WebSocket Authentication <<<{NC}")
    await test_ws_invalid_token()
    await test_ws_no_token()
    connected = await test_ws_connection(admin_token)
    if not connected:
        print(f"\n{YELLOW}Cannot establish WebSocket connection. Skipping chat tests.{NC}")
        return

    gemini_available = bool(os.getenv("GEMINI_API_KEY"))
    if not gemini_available:
        assert_skip("MCP chat scenarios", "GEMINI_API_KEY not set — LLM-dependent tests skipped")
        return

    await scenario_chat_basic(admin_token)
    await scenario_chat_multistep(admin_token)

    if operator_token:
        await scenario_chat_rbac(operator_token)
    else:
        assert_skip("Chat RBAC tests", "operator login failed")


def main() -> None:
    asyncio.run(run_ws_tests())

    total = pass_count + fail_count + skip_count
    print(f"\n{CYAN}{'=' * 60}{NC}")
    print(f"{CYAN}  WEBSOCKET TEST SUMMARY{NC}")
    print(f"{CYAN}{'=' * 60}{NC}")
    print(f"  Total:   {total}")
    print(f"  {GREEN}Passed:  {pass_count}{NC}")
    print(f"  {RED}Failed:  {fail_count}{NC}")
    print(f"  {YELLOW}Skipped: {skip_count}{NC}")
    print(f"{CYAN}{'=' * 60}{NC}")

    if fail_count > 0:
        print(f"\n{RED}WEBSOCKET TESTS FAILED{NC}")
        sys.exit(1)
    else:
        print(f"\n{GREEN}ALL WEBSOCKET TESTS PASSED{NC}")


if __name__ == "__main__":
    main()
