"""Notes, folders, and sharing modules."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from krasis.client import Client
from krasis.types import Folder, Note, NoteVersion, Paginated, ShareStatus


@dataclass
class ListNotesOptions:
    folder_id: str = ""
    page: int = 1
    size: int = 20


@dataclass
class UpdateNoteOptions:
    title: str | None = None
    content: str | None = None
    folder_id: str | None = None
    is_public: bool | None = None
    version: int = 0
    change_summary: str | None = None


class NotesModule:
    """Note CRUD operations."""

    def __init__(self, client: Client) -> None:
        self._client = client

    async def list(self, opts: ListNotesOptions | None = None) -> Paginated[Note]:
        if opts is None:
            opts = ListNotesOptions()
        params: dict[str, str] = {
            "page": str(opts.page),
            "size": str(opts.size),
        }
        if opts.folder_id:
            params["folder_id"] = opts.folder_id
        qs = "&".join(f"{k}={v}" for k, v in params.items())
        return await self._client.get_paginated(f"/notes?{qs}", Note)

    async def create(
        self, title: str, content: str = "", folder_id: str | None = None
    ) -> Note:
        body: dict[str, Any] = {"title": title, "content": content}
        if folder_id:
            body["folder_id"] = folder_id
        data = await self._client.post("/notes", body=body)
        return Note(**data)

    async def get(self, note_id: str) -> Note:
        data = await self._client.get(f"/notes/{note_id}")
        return Note(**data)

    async def update(self, note_id: str, opts: UpdateNoteOptions) -> Note:
        body: dict[str, Any] = {}
        if opts.title is not None:
            body["title"] = opts.title
        if opts.content is not None:
            body["content"] = opts.content
        if opts.folder_id is not None:
            body["folder_id"] = opts.folder_id
        if opts.is_public is not None:
            body["is_public"] = opts.is_public
        if opts.change_summary is not None:
            body["change_summary"] = opts.change_summary

        headers: dict[str, str] = {}
        if opts.version > 0:
            headers["If-Match"] = str(opts.version)

        data = await self._client.put(f"/notes/{note_id}", body=body, headers=headers)
        return Note(**data)

    async def delete(self, note_id: str) -> None:
        await self._client.delete(f"/notes/{note_id}")

    async def versions(self, note_id: str) -> list[NoteVersion]:
        data = await self._client.get(f"/notes/{note_id}/versions")
        return [NoteVersion(**v) for v in data]

    async def restore_version(self, note_id: str, version: int) -> Note:
        data = await self._client.post(f"/notes/{note_id}/versions/{version}/restore")
        return Note(**data)


class FoldersModule:
    """Folder CRUD operations."""

    def __init__(self, client: Client) -> None:
        self._client = client

    async def list(self) -> list[Folder]:
        data = await self._client.get("/folders")
        return [Folder(**f) for f in data]

    async def create(
        self, name: str, parent_id: str | None = None, color: str | None = None, sort_order: int = 0
    ) -> Folder:
        body: dict[str, Any] = {"name": name, "sort_order": sort_order}
        if parent_id:
            body["parent_id"] = parent_id
        if color:
            body["color"] = color
        data = await self._client.post("/folders", body=body)
        return Folder(**data)

    async def update(
        self,
        folder_id: str,
        name: str = "",
        parent_id: str | None = None,
        color: str | None = None,
        sort_order: int | None = None,
    ) -> Folder:
        body: dict[str, Any] = {}
        if name:
            body["name"] = name
        if parent_id:
            body["parent_id"] = parent_id
        if color:
            body["color"] = color
        if sort_order is not None:
            body["sort_order"] = sort_order
        data = await self._client.put(f"/folders/{folder_id}", body=body)
        return Folder(**data)

    async def delete(self, folder_id: str) -> None:
        await self._client.delete(f"/folders/{folder_id}")


class ShareModule:
    """Note sharing operations."""

    def __init__(self, client: Client) -> None:
        self._client = client

    async def create(
        self,
        note_id: str,
        permission: str = "read",
        password: str = "",
        expires_at: str = "",
    ) -> ShareStatus:
        body: dict[str, Any] = {"permission": permission}
        if password:
            body["password"] = password
        if expires_at:
            body["expires_at"] = expires_at
        data = await self._client.post(f"/notes/{note_id}/share", body=body)
        return ShareStatus(**data)

    async def get(self, note_id: str) -> ShareStatus:
        data = await self._client.get(f"/notes/{note_id}/share")
        return ShareStatus(**data)

    async def revoke(self, note_id: str) -> None:
        await self._client.delete(f"/notes/{note_id}/share")

    async def access_by_token(self, token: str, password: str = "") -> Note:
        path = f"/share/{token}"
        if password:
            from urllib.parse import quote
            path += f"?password={quote(password)}"
        data = await self._client.get(path)
        return Note(**data)
