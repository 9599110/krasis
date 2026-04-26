"""Data types and models for the Krasis SDK."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Generic, TypeVar

T = TypeVar("T")


@dataclass
class User:
    id: str
    username: str
    email: str
    avatar_url: str = ""
    role: str = ""
    created_at: str = ""
    updated_at: str = ""


@dataclass
class Note:
    id: str
    title: str
    content: str = ""
    folder_id: str = ""
    owner_id: str = ""
    is_public: bool = False
    version: int = 0
    created_at: str = ""
    updated_at: str = ""

    @property
    def preview(self) -> str:
        if len(self.content) <= 150:
            return self.content
        return self.content[:147] + "..."


@dataclass
class NoteVersion:
    id: str
    note_id: str
    version: int
    content: str = ""
    change_summary: str = ""
    created_by: str = ""
    created_at: str = ""


@dataclass
class Folder:
    id: str
    name: str
    parent_id: str = ""
    color: str = ""
    sort_order: int = 0
    created_at: str = ""
    updated_at: str = ""


@dataclass
class ShareStatus:
    id: str
    note_id: str
    token: str = ""
    permission: str = "read"
    is_active: bool = True
    expires_at: str = ""
    created_at: str = ""


@dataclass
class SearchResult:
    type: str = ""
    id: str = ""
    title: str = ""
    highlights: str = ""
    score: float = 0.0
    updated_at: str = ""


@dataclass
class Session:
    id: str
    user_id: str
    ip_address: str = ""
    user_agent: str = ""
    is_current: bool = False
    created_at: str = ""
    last_active: str = ""


@dataclass
class AskRequest:
    question: str
    conversation_id: str = ""
    model: str = ""
    stream: bool = False


@dataclass
class AskResponse:
    answer: str = ""
    conversation_id: str = ""
    references: list["Reference"] = field(default_factory=list)


@dataclass
class Reference:
    note_id: str = ""
    title: str = ""
    score: float = 0.0


@dataclass
class Conversation:
    id: str
    title: str = ""
    model: str = ""
    created_at: str = ""
    updated_at: str = ""


@dataclass
class Message:
    id: str
    conversation_id: str
    role: str = ""
    content: str = ""
    created_at: str = ""


@dataclass
class PresignResult:
    upload_url: str = ""
    file_id: str = ""
    key: str = ""


@dataclass
class FileItem:
    id: str
    file_name: str = ""
    file_type: str = ""
    size_bytes: int = 0
    status: str = ""
    created_at: str = ""


@dataclass
class Paginated(Generic[T]):
    items: list[T] = field(default_factory=list)
    total: int = 0
    page: int = 0
    size: int = 0


@dataclass
class AwarenessPayload:
    cursor: dict[str, Any] | None = None
    selection: dict[str, Any] | None = None


@dataclass
class SyncPayload:
    update: str = ""
    version: int = 0
