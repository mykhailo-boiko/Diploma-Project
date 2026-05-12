
import os

GATEWAY_URL = os.getenv("GATEWAY_URL", "http://localhost:8080")
REQUEST_TIMEOUT = float(os.getenv("MCP_REQUEST_TIMEOUT", "30"))

SERVICE_USER_EMAIL = os.getenv("SERVICE_USER_EMAIL", "admin@chainorchestra.local")
SERVICE_USER_PASSWORD = os.getenv("SERVICE_USER_PASSWORD", "")
