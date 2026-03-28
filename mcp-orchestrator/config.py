"""Service configuration for MCP orchestrator."""

import os

GATEWAY_URL = os.getenv("GATEWAY_URL", "http://localhost:8080")
REQUEST_TIMEOUT = float(os.getenv("MCP_REQUEST_TIMEOUT", "30"))
