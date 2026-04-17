FROM python:3.12-slim

WORKDIR /app

# Install mcp-orchestrator first (it's the MCP Server subprocess).
COPY mcp-orchestrator/pyproject.toml /app/mcp-orchestrator/
RUN pip install --no-cache-dir /app/mcp-orchestrator/

COPY mcp-orchestrator/ /app/mcp-orchestrator/

# Install mcp-host dependencies.
COPY mcp-host/pyproject.toml /app/mcp-host/
RUN pip install --no-cache-dir /app/mcp-host/

COPY mcp-host/ /app/mcp-host/

WORKDIR /app/mcp-host

ENV PYTHONUNBUFFERED=1
ENV MCP_SERVER_CMD=python
ENV MCP_SERVER_ARGS=/app/mcp-orchestrator/main.py

EXPOSE 8090

CMD ["python", "main.py"]
