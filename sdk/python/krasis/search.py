"""Search and file modules."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from krasis.client import Client
from krasis.types import FileItem, PresignResult, SearchResult


@dataclass
class SearchOptions:
    page: int = 1
    size: int = 20
    type: str = ""


class SearchModule:
    """Full-text search operations."""

    def __init__(self, client: Client) -> None:
        self._client = client

    async def query(self, q: str, opts: SearchOptions | None = None) -> list[SearchResult]:
        if opts is None:
            opts = SearchOptions()
        params: dict[str, str] = {
            "q": q,
            "page": str(opts.page),
            "size": str(opts.size),
        }
        if opts.type:
            params["type"] = opts.type
        qs = "&".join(f"{k}={v}" for k, v in params.items())
        wrapper = await self._client.get(f"/search?{qs}")
        return [SearchResult(**item) for item in wrapper.get("items", [])]


class FileModule:
    """File upload operations."""

    def __init__(self, client: Client) -> None:
        self._client = client

    async def presign_upload(
        self,
        file_name: str,
        file_type: str,
        note_id: str | None = None,
        size_bytes: int | None = None,
    ) -> PresignResult:
        body: dict[str, Any] = {
            "file_name": file_name,
            "file_type": file_type,
        }
        if note_id:
            body["note_id"] = note_id
        if size_bytes is not None:
            body["size_bytes"] = size_bytes
        data = await self._client.post("/files/presign", body=body)
        return PresignResult(**data)

    async def confirm_upload(self, file_id: str) -> None:
        await self._client.post(f"/files/{file_id}/confirm")

    async def delete(self, file_id: str) -> None:
        await self._client.delete(f"/files/{file_id}")
