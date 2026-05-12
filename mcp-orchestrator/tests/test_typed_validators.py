
from __future__ import annotations

import sys
from datetime import date, datetime, timezone
from pathlib import Path

import pytest
from pydantic import TypeAdapter, ValidationError

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from types_mcp import (  # noqa: E402
    EmailAddr, ISODate, ISODateTime, PhoneE164, TrackingNumber, UUIDStr,
)

def _adapter(tp):
    return TypeAdapter(tp)

class TestISODate:
    def test_accepts_valid_iso_date(self):
        v = _adapter(ISODate).validate_python("2026-05-12")
        assert v == date(2026, 5, 12)

    def test_accepts_date_object(self):
        v = _adapter(ISODate).validate_python(date(2026, 5, 12))
        assert v == date(2026, 5, 12)

    def test_rejects_slash_format(self):
        with pytest.raises(ValidationError) as exc:
            _adapter(ISODate).validate_python("2026/05/12")
        assert "YYYY-MM-DD" in str(exc.value)

    def test_rejects_textual_month(self):
        with pytest.raises(ValidationError):
            _adapter(ISODate).validate_python("May 12, 2026")

    def test_rejects_datetime_with_time(self):
        with pytest.raises(ValidationError):
            _adapter(ISODate).validate_python("2026-05-12T10:00:00Z")

class TestISODateTime:
    def test_accepts_rfc3339_with_z(self):
        v = _adapter(ISODateTime).validate_python("2026-05-12T15:30:00Z")
        assert v == datetime(2026, 5, 12, 15, 30, 0, tzinfo=timezone.utc)

    def test_accepts_rfc3339_with_offset(self):
        v = _adapter(ISODateTime).validate_python("2026-05-12T15:30:00+02:00")
        assert v.year == 2026 and v.tzinfo is not None

    def test_naive_datetime_assumed_utc(self):
        v = _adapter(ISODateTime).validate_python("2026-05-12T15:30:00")
        assert v.tzinfo is not None

    def test_rejects_invalid_format(self):
        with pytest.raises(ValidationError) as exc:
            _adapter(ISODateTime).validate_python("2026-05-12 not a date")
        assert "RFC3339" in str(exc.value) or "ISO" in str(exc.value)

class TestUUIDStr:
    def test_accepts_lowercase_uuid(self):
        v = _adapter(UUIDStr).validate_python("550e8400-e29b-41d4-a716-446655440000")
        assert v == "550e8400-e29b-41d4-a716-446655440000"

    def test_accepts_uppercase_uuid(self):
        v = _adapter(UUIDStr).validate_python("550E8400-E29B-41D4-A716-446655440000")
        assert v == "550e8400-e29b-41d4-a716-446655440000"

    def test_rejects_missing_hyphens(self):
        with pytest.raises(ValidationError) as exc:
            _adapter(UUIDStr).validate_python("550e8400e29b41d4a716446655440000")
        assert "hyphen" in str(exc.value).lower() or "UUID" in str(exc.value)

    def test_rejects_tracking_number_as_uuid(self):
        with pytest.raises(ValidationError) as exc:
            _adapter(UUIDStr).validate_python("CO-2026-K7H2P9")
        assert "UUID" in str(exc.value)

    def test_rejects_non_string(self):
        with pytest.raises(ValidationError):
            _adapter(UUIDStr).validate_python(12345)

class TestTrackingNumber:
    def test_accepts_valid_tracking(self):
        v = _adapter(TrackingNumber).validate_python("CO-2026-K7H2P9")
        assert v == "CO-2026-K7H2P9"

    def test_uppercases(self):
        v = _adapter(TrackingNumber).validate_python("co-2026-k7h2p9")
        assert v == "CO-2026-K7H2P9"

    def test_rejects_uuid_as_tracking(self):
        with pytest.raises(ValidationError) as exc:
            _adapter(TrackingNumber).validate_python("550e8400-e29b-41d4-a716-446655440000")
        assert "CO-YYYY-XXXXXX" in str(exc.value)

class TestPhoneE164:
    def test_accepts_valid(self):
        assert _adapter(PhoneE164).validate_python("+380501112233") == "+380501112233"

    def test_strips_spaces(self):
        assert _adapter(PhoneE164).validate_python("+380 50 111 22 33") == "+380501112233"

    def test_rejects_missing_plus(self):
        with pytest.raises(ValidationError):
            _adapter(PhoneE164).validate_python("380501112233")

class TestEmailAddr:
    def test_lowercases(self):
        assert _adapter(EmailAddr).validate_python("Foo@Bar.COM") == "foo@bar.com"

    def test_rejects_missing_at(self):
        with pytest.raises(ValidationError):
            _adapter(EmailAddr).validate_python("not an email")
