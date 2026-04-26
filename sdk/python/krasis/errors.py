"""Custom exception types for the Krasis SDK."""


class KrasisError(Exception):
    """Base exception for all Krasis SDK errors."""


class AuthenticationError(KrasisError):
    """Raised when the server returns 401 Unauthorized."""


class NotFoundError(KrasisError):
    """Raised when the server returns 404 Not Found."""


class RateLimitError(KrasisError):
    """Raised when the server returns 429 Too Many Requests."""


class VersionConflictError(KrasisError):
    """Raised when the server returns 409 Conflict (optimistic locking)."""

    def __init__(self, message: str = "", current_version: int = 0):
        super().__init__(message)
        self.current_version = current_version


class APIError(KrasisError):
    """Raised for unexpected server errors (5xx or other)."""

    def __init__(self, message: str = "", status_code: int = 0):
        super().__init__(message)
        self.status_code = status_code
