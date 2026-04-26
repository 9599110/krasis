"""AI module with streaming support."""

from __future__ import annotations

import json
from collections.abc import AsyncIterator
from typing import Any

import aiohttp

from krasis.client import Client
from krasis.types import AskRequest, AskResponse, Conversation, Message


class AIModule:
    """AI question and conversation operations."""

    def __init__(self, client: Client) -> None:
        self._client = client

    async def ask(self, req: AskRequest) -> AskResponse:
        """Send a question and return the answer."""
        body = self._request_to_dict(req)
        data = await self._client.post("/ai/ask", body=body)
        return AskResponse(**data)

    async def ask_stream(self, req: AskRequest) -> AsyncIterator[str]:
        """Send a question and yield tokens as they arrive via SSE."""
        req.stream = True
        body = self._request_to_dict(req)

        import json as _json

        session = await self._client._get_session()
        url = f"{self._client.base_url}/ai/ask/stream"
        headers: dict[str, str] = {"Content-Type": "application/json"}
        if self._client.token:
            headers["Authorization"] = f"Bearer {self._client.token}"

        async with session.post(url, headers=headers, data=_json.dumps(body)) as resp:
            if resp.status != 200:
                body_text = await resp.text()
                raise Exception(f"HTTP {resp.status}: {body_text}")

            last_event = ""
            async for line in resp.content:
                line_str = line.decode("utf-8").strip()
                if line_str.startswith("event: "):
                    last_event = line_str[7:]
                    continue
                if line_str.startswith("data: "):
                    payload = line_str[6:]
                    if last_event == "token":
                        try:
                            d = json.loads(payload)
                            token = d.get("token", "")
                            if token:
                                yield token
                        except json.JSONDecodeError:
                            continue
                    elif last_event == "done":
                        return

    async def list_conversations(self) -> list[Conversation]:
        """Return all conversations."""
        data = await self._client.get("/ai/conversations")
        return [Conversation(**c) for c in data]

    async def get_messages(self, conversation_id: str) -> list[Message]:
        """Return all messages in a conversation."""
        data = await self._client.get(f"/ai/conversations/{conversation_id}/messages")
        return [Message(**m) for m in data]

    async def create_conversation(self, title: str = "", model: str = "") -> Conversation:
        """Create a new conversation."""
        body: dict[str, str] = {}
        if title:
            body["title"] = title
        if model:
            body["model"] = model
        data = await self._client.post("/ai/conversations", body=body)
        return Conversation(**data)

    @staticmethod
    def _request_to_dict(req: AskRequest) -> dict[str, Any]:
        result: dict[str, Any] = {"question": req.question}
        if req.conversation_id:
            result["conversation_id"] = req.conversation_id
        if req.model:
            result["model"] = req.model
        result["stream"] = req.stream
        return result
