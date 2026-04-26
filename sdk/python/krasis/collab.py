"""WebSocket collaboration module."""

from __future__ import annotations

import asyncio
import json
import math
from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Any
from urllib.parse import urlencode

import websockets

from krasis.types import AwarenessPayload, SyncPayload


@dataclass
class CollabEvent:
    type: str = ""
    payload: dict[str, Any] = field(default_factory=dict)
    user_id: str = ""


CollabHandler = Callable[[CollabEvent], None]


class CollabModule:
    """WebSocket collaboration session manager."""

    def __init__(self, ws_base_url: str, token: Callable[[], str]) -> None:
        self._ws_base_url = ws_base_url
        self._token = token
        self._note_id = ""
        self._handlers: list[CollabHandler] = []
        self._ws: websockets.WebSocketClientProtocol | None = None
        self._reconnect_attempts = 0
        self._max_reconnect_attempts = 5
        self._read_task: asyncio.Task[None] | None = None

    def on(self, handler: CollabHandler) -> None:
        """Register a collaboration event handler."""
        self._handlers.append(handler)

    async def connect(self, note_id: str) -> None:
        """Establish a WebSocket connection to a note."""
        self._note_id = note_id
        self._reconnect_attempts = 0
        await self._do_connect()

    async def _do_connect(self) -> None:
        if self._ws:
            await self._ws.close()

        params = urlencode({"note_id": self._note_id, "token": self._token()})
        url = f"{self._ws_base_url}/ws/collab?{params}"

        try:
            self._ws = await websockets.connect(url)
            self._emit(CollabEvent(type="open"))
            self._read_task = asyncio.create_task(self._read_loop())
        except Exception as exc:
            self._emit(CollabEvent(type="error", payload={"error": str(exc)}))
            await self._schedule_reconnect()

    async def _read_loop(self) -> None:
        try:
            assert self._ws is not None
            async for message in self._ws:
                try:
                    msg = json.loads(message)
                    self._emit(CollabEvent(
                        type=msg.get("type", ""),
                        payload=msg.get("payload", {}),
                        user_id=msg.get("user_id", ""),
                    ))
                except json.JSONDecodeError:
                    continue
        except websockets.ConnectionClosed:
            self._emit(CollabEvent(type="close"))
            await self._schedule_reconnect()

    async def send_sync(self, update: str, version: int) -> None:
        """Broadcast a document update."""
        await self._send({
            "type": "sync",
            "payload": {"update": update, "version": version},
        })

    async def send_awareness(self, payload: dict[str, Any]) -> None:
        """Broadcast awareness state."""
        await self._send({"type": "awareness", "payload": payload})

    async def send_presence_query(self) -> None:
        """Request the list of online users."""
        await self._send({"type": "awareness_query", "payload": {}})

    async def _send(self, msg: dict[str, Any]) -> None:
        if self._ws is None:
            return
        await self._ws.send(json.dumps(msg))

    def _emit(self, event: CollabEvent) -> None:
        for handler in self._handlers:
            handler(event)

    async def _schedule_reconnect(self) -> None:
        if self._reconnect_attempts >= self._max_reconnect_attempts:
            return
        self._reconnect_attempts += 1
        delay = min(1.0 * (2 ** self._reconnect_attempts), 30.0)
        await asyncio.sleep(delay)
        await self._do_connect()

    async def disconnect(self) -> None:
        """Close the connection and stop reconnecting."""
        if self._read_task:
            self._read_task.cancel()
            try:
                await self._read_task
            except asyncio.CancelledError:
                pass
        self._reconnect_attempts = self._max_reconnect_attempts
        if self._ws:
            await self._ws.close()
            self._ws = None
