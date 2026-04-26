"""Authentication and user modules."""

from __future__ import annotations

from urllib.parse import urlencode

from krasis.client import Client
from krasis.types import Session, User


class AuthModule:
    """Authentication operations."""

    def __init__(self, client: Client) -> None:
        self._client = client

    def oauth_url(self, provider: str, redirect_uri: str = "", state: str = "") -> str:
        """Build the OAuth authorization URL."""
        params: dict[str, str] = {"provider": provider}
        if redirect_uri:
            params["redirect_uri"] = redirect_uri
        if state:
            params["state"] = state
        return f"{self._client.base_url}/auth/oauth?{urlencode(params)}"

    async def callback(self, provider: str, code: str, state: str = "") -> dict:
        """Exchange the OAuth code for a token."""
        body: dict[str, str] = {"provider": provider, "code": code}
        if state:
            body["state"] = state
        return await self._client.post("/auth/oauth/callback", body=body)

    async def login(self, email: str, password: str) -> None:
        """Login and store the token."""
        result = await self._client.post("/auth/login", body={
            "email": email,
            "password": password,
        })
        self._client.set_token(result["token"])

    async def register(self, email: str, password: str, username: str) -> None:
        """Create a new user account."""
        await self._client.post("/auth/register", body={
            "email": email,
            "password": password,
            "username": username,
        })

    async def logout(self) -> None:
        """Logout and clear the token."""
        try:
            await self._client.post("/auth/logout")
        finally:
            self._client.clear_token()

    async def me(self) -> User:
        """Fetch the current user profile."""
        data = await self._client.get("/auth/me")
        return User(**data)


class UserModule:
    """User session and profile operations."""

    def __init__(self, client: Client) -> None:
        self._client = client

    async def sessions(self) -> list[Session]:
        """List active sessions."""
        data = await self._client.get("/user/sessions")
        return [Session(**s) for s in data]

    async def revoke_session(self, session_id: str) -> None:
        """Terminate a specific session."""
        await self._client.delete(f"/user/sessions/{session_id}")

    async def update_profile(self, username: str = "", avatar_url: str = "") -> None:
        """Update the current user's profile."""
        body: dict[str, str] = {}
        if username:
            body["username"] = username
        if avatar_url:
            body["avatar_url"] = avatar_url
        await self._client.put("/user/profile", body=body)
