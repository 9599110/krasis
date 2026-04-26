"""HTTP client with authentication and error handling."""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Any, Type, TypeVar

import aiohttp

from krasis.errors import APIError, AuthenticationError, NotFoundError, RateLimitError, VersionConflictError
from krasis.types import Paginated

T = TypeVar("T")


@dataclass
class Client:
    """Async HTTP client for the Krasis API."""

    base_url: str
    _token: str | None = field(default=None, repr=False)

    def __post_init__(self) -> None:
        self._session: aiohttp.ClientSession | None = None

    async def _get_session(self) -> aiohttp.ClientSession:
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession()
        return self._session

    async def close(self) -> None:
        """Close the underlying HTTP session."""
        if self._session and not self._session.closed:
            await self._session.close()

    async def __aenter__(self) -> "Client":
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.close()

    def set_token(self, token: str) -> None:
        """Set the authentication token."""
        self._token = token

    def clear_token(self) -> None:
        """Remove the authentication token."""
        self._token = None

    @property
    def token(self) -> str | None:
        return self._token

    async def _request(
        self,
        method: str,
        path: str,
        *,
        body: dict[str, Any] | None = None,
        headers: dict[str, str] | None = None,
    ) -> Any:
        session = await self._get_session()
        url = f"{self.base_url}{path}"
        req_headers: dict[str, str] = {"Content-Type": "application/json"}
        if self._token:
            req_headers["Authorization"] = f"Bearer {self._token}"
        if headers:
            req_headers.update(headers)

        json_body = json.dumps(body) if body else None

        async with session.request(method, url, headers=req_headers, data=json_body) as resp:
            if resp.status == 204:
                return None
            raw = await resp.text()
            if resp.status >= 400:
                self._raise_for_status(resp.status, raw)
            envelope = json.loads(raw)
            data = envelope.get("data")
            if data is None:
                data = envelope
            return data

    async def get(self, path: str) -> Any:
        return await self._request("GET", path)

    async def post(self, path: str, body: dict[str, Any] | None = None) -> Any:
        return await self._request("POST", path, body=body)

    async def put(
        self,
        path: str,
        body: dict[str, Any] | None = None,
        headers: dict[str, str] | None = None,
    ) -> Any:
        return await self._request("PUT", path, body=body, headers=headers)

    async def delete(self, path: str) -> Any:
        return await self._request("DELETE", path)

    async def get_paginated(self, path: str, item_type: Type[T]) -> Paginated[T]:
        """Fetch a paginated response and return a Paginated wrapper."""
        data = await self.get(path)
        return Paginated(
            items=[item_type(**item) for item in data.get("items", [])],
            total=data.get("total", 0),
            page=data.get("page", 0),
            size=data.get("size", 0),
        )

    @staticmethod
    def _raise_for_status(status: int, body: str) -> None:
        message = body
        try:
            envelope = json.loads(body)
            message = envelope.get("error", envelope.get("message", body))
        except (json.JSONDecodeError, ValueError):
            pass

        if status == 401:
            raise AuthenticationError(message)
        if status == 404:
            raise NotFoundError(message)
        if status == 409:
            raise VersionConflictError(message)
        if status == 429:
            raise RateLimitError(message)
        raise APIError(message, status_code=status)
