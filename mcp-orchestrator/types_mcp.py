
from __future__ import annotations

import re
from datetime import date, datetime, timezone
from typing import Annotated, Any, Literal

from pydantic import BeforeValidator, Field

_UUID_RE = re.compile(r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
_TRACKING_RE = re.compile(r"^CO-\d{4}-[A-Z0-9]{6}$")
_E164_RE = re.compile(r"^\+[1-9]\d{6,14}$")
_EMAIL_RE = re.compile(r"^[^@\s]+@[^@\s]+\.[^@\s]+$")

def _parse_iso_date(v: Any) -> date | str:
    if v is None or v == "":
        return v
    if isinstance(v, date) and not isinstance(v, datetime):
        return v
    if isinstance(v, datetime):
        return v.date()
    if isinstance(v, str):
        s = v.strip()
        try:
            return date.fromisoformat(s)
        except ValueError as exc:
            raise ValueError(
                f"must be ISO 8601 date in format 'YYYY-MM-DD' (e.g. '2026-05-12'); "
                f"received: {v!r}. Common mistakes: using '/' or '.' instead of '-', "
                f"or including time (use ISODateTime for timestamps). Original error: {exc}"
            ) from None
    raise ValueError(
        f"must be a string in format 'YYYY-MM-DD'; received type {type(v).__name__} value {v!r}"
    )

def _parse_iso_datetime(v: Any) -> datetime | str:
    if v is None or v == "":
        return v
    if isinstance(v, datetime):
        return v if v.tzinfo else v.replace(tzinfo=timezone.utc)
    if isinstance(v, date):
        return datetime(v.year, v.month, v.day, tzinfo=timezone.utc)
    if isinstance(v, str):
        s = v.strip().replace("Z", "+00:00")
        try:
            parsed = datetime.fromisoformat(s)
        except ValueError as exc:
            raise ValueError(
                f"must be ISO 8601 datetime (RFC3339), e.g. '2026-05-12T15:30:00Z' or "
                f"'2026-05-12T15:30:00+02:00'; received: {v!r}. "
                f"For date-only filters use 'YYYY-MM-DD' with ISODate fields. "
                f"Original error: {exc}"
            ) from None
        return parsed if parsed.tzinfo else parsed.replace(tzinfo=timezone.utc)
    raise ValueError(
        f"must be ISO 8601 datetime string; received type {type(v).__name__} value {v!r}"
    )

def _validate_uuid(v: Any) -> str:
    if not isinstance(v, str):
        raise ValueError(
            f"must be a UUID string (8-4-4-4-12 hex format); received type {type(v).__name__} value {v!r}. "
            f"Example: '550e8400-e29b-41d4-a716-446655440000'"
        )
    s = v.strip()
    if not _UUID_RE.match(s):
        raise ValueError(
            f"must be a valid UUID in 8-4-4-4-12 hex format; received: {v!r}. "
            f"Example: '550e8400-e29b-41d4-a716-446655440000'. "
            f"Common mistakes: missing hyphens, wrong length, using a tracking number (CO-YYYY-XXXXXX) "
            f"or customer name instead of UUID."
        )
    return s.lower()

def _validate_tracking_number(v: Any) -> str:
    if not isinstance(v, str):
        raise ValueError(
            f"must be a tracking number string; received type {type(v).__name__} value {v!r}"
        )
    s = v.strip().upper()
    if not _TRACKING_RE.match(s):
        raise ValueError(
            f"tracking_number must be in format 'CO-YYYY-XXXXXX' (e.g. 'CO-2026-B32B8A'); "
            f"received: {v!r}. "
            f"If you have a shipment UUID, use the shipments_get tool instead — tracking numbers "
            f"are different from internal shipment IDs."
        )
    return s

def _validate_e164(v: Any) -> str:
    if not isinstance(v, str):
        raise ValueError(f"phone must be E.164 string; received type {type(v).__name__}")
    s = v.strip().replace(" ", "").replace("-", "")
    if not _E164_RE.match(s):
        raise ValueError(
            f"phone must be E.164 format (starts with '+' country code, 7-15 digits total); "
            f"received: {v!r}. Examples: '+380501112233', '+15555550100'. "
            f"Common mistakes: missing '+', including spaces/dashes/parentheses, including extension."
        )
    return s

def _validate_email(v: Any) -> str:
    if not isinstance(v, str):
        raise ValueError(f"email must be string; received type {type(v).__name__}")
    s = v.strip().lower()
    if not _EMAIL_RE.match(s):
        raise ValueError(
            f"email must be in 'local@domain.tld' format; received: {v!r}"
        )
    return s

UUIDStr = Annotated[
    str,
    BeforeValidator(_validate_uuid),
    Field(description="UUID v4 string in 8-4-4-4-12 hex format, e.g. '550e8400-e29b-41d4-a716-446655440000'"),
]

TrackingNumber = Annotated[
    str,
    BeforeValidator(_validate_tracking_number),
    Field(description="Public tracking number in format 'CO-YYYY-XXXXXX', e.g. 'CO-2026-B32B8A'"),
]

ISODate = Annotated[
    date,
    BeforeValidator(_parse_iso_date),
    Field(description="ISO 8601 date string in format 'YYYY-MM-DD', e.g. '2026-05-12'"),
]

ISODateTime = Annotated[
    datetime,
    BeforeValidator(_parse_iso_datetime),
    Field(description="ISO 8601 / RFC3339 datetime string, e.g. '2026-05-12T15:30:00Z' or '2026-05-12T15:30:00+02:00'"),
]

PhoneE164 = Annotated[
    str,
    BeforeValidator(_validate_e164),
    Field(description="Phone number in E.164 format, e.g. '+380501112233'"),
]

EmailAddr = Annotated[
    str,
    BeforeValidator(_validate_email),
    Field(description="Email address in 'local@domain.tld' format"),
]

PositiveInt = Annotated[int, Field(gt=0, le=1_000_000, description="positive integer (> 0)")]
NonNegativeInt = Annotated[int, Field(ge=0, le=10_000_000, description="non-negative integer (>= 0)")]
PageLimit = Annotated[int, Field(gt=0, le=1000, description="page size (1..1000); default 100 in most tools")]
PageOffset = Annotated[int, Field(ge=0, le=1_000_000, description="page offset (>= 0)")]
Money = Annotated[
    float,
    Field(ge=0, le=10_000_000.0, description="monetary amount in major units (e.g. 19.99 = $19.99)"),
]

OrderStatus = Literal[
    "pending", "confirmed", "processing", "shipped",
    "delivered", "completed", "cancelled", "returned",
]

ShipmentStatus = Literal[
    "created", "label_created", "awaiting_pickup", "picked_up",
    "in_transit", "at_hub", "out_for_delivery", "delivery_attempted",
    "held_at_office", "delivered", "failed", "returned_to_sender",
    "returned", "cancelled", "redirected",
]

StockMovementType = Literal["inbound", "outbound", "adjustment"]

CarrierType = Literal["air", "sea", "land", "rail", "express"]

NotificationType = Literal[
    "order_created", "order_updated", "order_cancelled",
    "low_stock", "stock_changed",
    "shipment_created", "shipment_updated", "shipment_milestone",
    "system", "system_alert", "info",
    "low_stock_alert", "analytics_anomaly",
]

NotificationPriority = Literal["low", "medium", "high", "urgent"]

NotificationChannel = Literal["in_app", "email", "sms"]

UserRole = Literal["admin", "operator", "warehouse_manager", "logistics_manager", "analyst"]

SortOrder = Literal["asc", "desc"]

SimulatorScenario = Literal["idle", "steady", "holiday_spike", "carrier_failure", "demand_surge"]

AnalyticsMetric = Literal[
    "revenue", "order_count", "aov", "cancellation_rate",
    "on_time_rate", "shipment_count", "low_stock_count",
]

ForecastMethod = Literal["linear", "rolling-avg", "ets-simple"]

AnomalyCategory = Literal["sales", "logistics", "inventory", "business", "all"]

ProductCategory = Literal[
    "Electronics", "Furniture", "Clothing", "Food", "Tools",
    "Industrial", "Office", "Sports", "Automotive", "Health", "Other",
]
