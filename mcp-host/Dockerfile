FROM python:3.12-slim

WORKDIR /app

COPY mcp-orchestrator/ /app/mcp-orchestrator/
RUN pip install --no-cache-dir /app/mcp-orchestrator/

COPY mcp-host/ /app/mcp-host/
RUN pip install --no-cache-dir /app/mcp-host/

WORKDIR /app/mcp-host

ENV PYTHONUNBUFFERED=1
ENV MCP_SERVER_CMD=python
ENV MCP_SERVER_ARGS=/app/mcp-orchestrator/main.py

EXPOSE 8090

CMD ["python", "main.py"]
