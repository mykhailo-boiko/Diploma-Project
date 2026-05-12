
from __future__ import annotations

import contextlib
import logging
import os
from typing import Any, Iterator

logger = logging.getLogger(__name__)

_initialized = False
_logfire_mod: Any | None = None

def setup_logfire(service_name: str = "mcp-host") -> None:

    global _initialized, _logfire_mod  # noqa: PLW0603

    if _initialized:
        return

    try:
        import logfire  # type: ignore
    except ImportError:
        logger.info("logfire not installed; observability spans will be no-ops")
        _initialized = True
        return

    token = os.getenv("LOGFIRE_TOKEN", "").strip() or None
    environment = os.getenv("LOGFIRE_ENVIRONMENT", "development")

    try:
        logfire.configure(
            token=token,
            service_name=service_name,
            environment=environment,
            send_to_logfire="if-token-present",
            console=False,
        )
        _logfire_mod = logfire
        if token:
            logger.info("Logfire configured (service=%s env=%s)", service_name, environment)
        else:
            logger.info("Logfire configured in local mode (no LOGFIRE_TOKEN)")
    except Exception as exc:
        logger.warning("Logfire setup failed: %s", exc)
    _initialized = True

def instrument_fastapi(app: Any) -> None:
    if _logfire_mod is None:
        return
    try:
        _logfire_mod.instrument_fastapi(app, excluded_urls=[r"/health", r"/metrics"])
    except Exception as exc:
        logger.debug("logfire.instrument_fastapi failed: %s", exc)

def instrument_httpx() -> None:
    if _logfire_mod is None:
        return
    try:
        _logfire_mod.instrument_httpx()
    except Exception as exc:
        logger.debug("logfire.instrument_httpx failed: %s", exc)

class _NoOpSpan:
    def __enter__(self) -> "_NoOpSpan":
        return self

    def __exit__(self, *_: Any) -> None:
        return None

    def set_attribute(self, *_: Any, **__: Any) -> None:
        return None

    def set_attributes(self, *_: Any, **__: Any) -> None:
        return None

    def record_exception(self, *_: Any, **__: Any) -> None:
        return None

@contextlib.contextmanager
def logfire_span(name: str, **attributes: Any) -> Iterator[Any]:

    if _logfire_mod is None:
        yield _NoOpSpan()
        return
    safe_attrs = {k: v for k, v in attributes.items() if v is not None}
    try:
        with _logfire_mod.span(name, **safe_attrs) as span:
            yield span
    except Exception as exc:
        logger.debug("logfire span %r failed: %s", name, exc)
        yield _NoOpSpan()

def extract_entity_ids(args: dict[str, Any]) -> dict[str, str]:

    out: dict[str, str] = {}
    if not isinstance(args, dict):
        return out
    for key in (
        "order_id", "shipment_id", "product_id", "warehouse_id", "carrier_id",
        "user_id", "notification_id", "tracking_number", "customer_name", "entity_id",
    ):
        v = args.get(key)
        if isinstance(v, str) and v:
            out[key] = v
    return out
