"""Service configuration for MCP Host."""

import os

JWT_SECRET = os.getenv("JWT_SECRET", "dev-secret-change-me")
JWT_ALGORITHM = "HS256"

GEMINI_API_KEY = os.getenv("GEMINI_API_KEY", "")
GEMINI_MODEL = os.getenv("GEMINI_MODEL", "gemini-2.5-flash")

MCP_SERVER_CMD = os.getenv("MCP_SERVER_CMD", "python")
MCP_SERVER_ARGS = os.getenv("MCP_SERVER_ARGS", "../mcp-orchestrator/main.py").split(",")
MCP_SERVER_ENV = {
    "GATEWAY_URL": os.getenv("GATEWAY_URL", "http://localhost:8080"),
    "MCP_REQUEST_TIMEOUT": os.getenv("MCP_REQUEST_TIMEOUT", "30"),
}

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")

SESSION_TTL = int(os.getenv("SESSION_TTL", "1800"))

HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", "8090"))

MAX_TOOL_ROUNDS = int(os.getenv("MAX_TOOL_ROUNDS", "10"))
TOOL_TIMEOUT = int(os.getenv("TOOL_TIMEOUT", "30"))

RETRY_MAX_ATTEMPTS = int(os.getenv("RETRY_MAX_ATTEMPTS", "3"))
RETRY_BASE_DELAY = float(os.getenv("RETRY_BASE_DELAY", "1.0"))
RETRY_MAX_DELAY = float(os.getenv("RETRY_MAX_DELAY", "10.0"))
