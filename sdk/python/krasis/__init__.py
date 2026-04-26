"""Krasis Python SDK — async client for the Krasis intelligent notes API."""

from krasis.client import Client
from krasis.sdk import KrasisSDK
from krasis.errors import (
    KrasisError,
    AuthenticationError,
    NotFoundError,
    RateLimitError,
    VersionConflictError,
    APIError,
)
from krasis.auth import AuthModule, UserModule
from krasis.notes import NotesModule, FoldersModule, ShareModule, ListNotesOptions, UpdateNoteOptions
from krasis.search import SearchModule, FileModule, SearchOptions
from krasis.ai import AIModule
from krasis.collab import CollabModule, CollabEvent, CollabHandler

__all__ = [
    "Client",
    "KrasisSDK",
    "KrasisError",
    "AuthenticationError",
    "NotFoundError",
    "RateLimitError",
    "VersionConflictError",
    "APIError",
    "AuthModule",
    "UserModule",
    "NotesModule",
    "FoldersModule",
    "ShareModule",
    "ListNotesOptions",
    "UpdateNoteOptions",
    "SearchModule",
    "FileModule",
    "SearchOptions",
    "AIModule",
    "CollabModule",
    "CollabEvent",
    "CollabHandler",
]
