"""Top-level SDK class that composes all modules."""

from __future__ import annotations

from dataclasses import dataclass, field, InitVar

from krasis.client import Client
from krasis.auth import AuthModule, UserModule
from krasis.notes import NotesModule, FoldersModule, ShareModule
from krasis.search import SearchModule, FileModule
from krasis.ai import AIModule
from krasis.collab import CollabModule


@dataclass
class KrasisSDK:
    """High-level SDK entry point that composes all modules."""

    base_url: str
    token: InitVar[str | None] = None

    client: Client = field(init=False)
    auth: AuthModule = field(init=False)
    users: UserModule = field(init=False)
    notes: NotesModule = field(init=False)
    folders: FoldersModule = field(init=False)
    share: ShareModule = field(init=False)
    search: SearchModule = field(init=False)
    files: FileModule = field(init=False)
    ai: AIModule = field(init=False)
    _collab: CollabModule | None = field(default=None, repr=False)

    def __post_init__(self, token: str | None) -> None:
        self.client = Client(base_url=self.base_url)
        if token:
            self.client.set_token(token)
        self.auth = AuthModule(self.client)
        self.users = UserModule(self.client)
        self.notes = NotesModule(self.client)
        self.folders = FoldersModule(self.client)
        self.share = ShareModule(self.client)
        self.search = SearchModule(self.client)
        self.files = FileModule(self.client)
        self.ai = AIModule(self.client)

    @property
    def collab(self) -> CollabModule:
        if self._collab is None:
            ws_url = self.base_url.replace("http", "ws")
            token = self.client.token or ""
            self._collab = CollabModule(ws_base_url=ws_url, token=token)
        return self._collab

    def set_token(self, token: str) -> None:
        self.client.set_token(token)

    def clear_token(self) -> None:
        self.client.clear_token()

    @property
    def is_authenticated(self) -> bool:
        return self.client.token is not None

    async def close(self) -> None:
        if self._collab:
            await self._collab.disconnect()
        await self.client.close()

    async def __aenter__(self) -> "KrasisSDK":
        return self

    async def __aexit__(self, *args: object) -> None:
        await self.close()
