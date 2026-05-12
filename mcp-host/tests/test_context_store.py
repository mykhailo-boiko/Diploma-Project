
import json
from unittest.mock import AsyncMock

import pytest
from google.genai import types

from context_store import ContextStore, _deserialize_history, _serialize_history

@pytest.fixture
def mock_redis():
    return AsyncMock()

@pytest.fixture
def store(mock_redis):
    return ContextStore(mock_redis)

class TestContextStoreSave:
    async def test_save_stores_json_with_ttl(self, store, mock_redis):
        history = [types.Content(role="user", parts=[types.Part(text="hello")])]
        await store.save("s1", history)

        mock_redis.set.assert_awaited_once()
        call_args = mock_redis.set.call_args
        assert call_args[0][0] == "history:s1"
        data = json.loads(call_args[0][1])
        assert len(data) == 1
        assert data[0]["role"] == "user"
        assert data[0]["parts"][0]["text"] == "hello"
        assert call_args[1]["ex"] == 1800

    async def test_save_empty_history(self, store, mock_redis):
        await store.save("s1", [])
        mock_redis.set.assert_awaited_once()
        data = json.loads(mock_redis.set.call_args[0][1])
        assert data == []

class TestContextStoreLoad:
    async def test_load_existing_session(self, store, mock_redis):
        raw = json.dumps([{"role": "user", "parts": [{"text": "hi"}]}])
        mock_redis.get.return_value = raw.encode()

        result = await store.load("s1")
        assert len(result) == 1
        assert result[0].role == "user"
        assert result[0].parts[0].text == "hi"

    async def test_load_missing_session(self, store, mock_redis):
        mock_redis.get.return_value = None
        result = await store.load("missing")
        assert result == []

    async def test_load_corrupt_data(self, store, mock_redis):
        mock_redis.get.return_value = b"not-json"
        result = await store.load("corrupt")
        assert result == []

class TestContextStoreDelete:
    async def test_delete_session(self, store, mock_redis):
        await store.delete("s1")
        mock_redis.delete.assert_awaited_once_with("history:s1")

class TestSerializationRoundTrip:
    def test_text_part_roundtrip(self):
        original = [
            types.Content(role="user", parts=[types.Part(text="hello")]),
            types.Content(role="model", parts=[types.Part(text="world")]),
        ]
        serialized = _serialize_history(original)
        restored = _deserialize_history(serialized)

        assert len(restored) == 2
        assert restored[0].role == "user"
        assert restored[0].parts[0].text == "hello"
        assert restored[1].role == "model"
        assert restored[1].parts[0].text == "world"

    def test_function_call_roundtrip(self):
        original = [
            types.Content(role="model", parts=[
                types.Part(function_call=types.FunctionCall(
                    name="orders_list",
                    args={"status": "pending"},
                    id="fc1",
                )),
            ]),
        ]
        serialized = _serialize_history(original)
        restored = _deserialize_history(serialized)

        assert len(restored) == 1
        fc = restored[0].parts[0].function_call
        assert fc.name == "orders_list"
        assert fc.args == {"status": "pending"}
        assert fc.id == "fc1"

    def test_function_response_roundtrip(self):
        original = [
            types.Content(role="user", parts=[
                types.Part(function_response=types.FunctionResponse(
                    name="orders_list",
                    response={"result": [{"id": "1"}]},
                    id="fc1",
                )),
            ]),
        ]
        serialized = _serialize_history(original)
        restored = _deserialize_history(serialized)

        assert len(restored) == 1
        fr = restored[0].parts[0].function_response
        assert fr.name == "orders_list"
        assert fr.response == {"result": [{"id": "1"}]}

    def test_mixed_parts_roundtrip(self):
        original = [
            types.Content(role="user", parts=[types.Part(text="list orders")]),
            types.Content(role="model", parts=[
                types.Part(function_call=types.FunctionCall(
                    name="orders_list", args={},
                )),
            ]),
            types.Content(role="user", parts=[
                types.Part(function_response=types.FunctionResponse(
                    name="orders_list", response={"orders": []},
                )),
            ]),
            types.Content(role="model", parts=[types.Part(text="No orders found.")]),
        ]
        serialized = _serialize_history(original)
        restored = _deserialize_history(serialized)

        assert len(restored) == 4
        assert restored[0].parts[0].text == "list orders"
        assert restored[1].parts[0].function_call.name == "orders_list"
        assert restored[2].parts[0].function_response.name == "orders_list"
        assert restored[3].parts[0].text == "No orders found."

    def test_empty_history(self):
        assert _serialize_history([]) == []
        assert _deserialize_history([]) == []
