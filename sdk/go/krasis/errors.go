package krasis

import (
	"errors"
	"fmt"
)

// Standard SDK errors.
var (
	ErrAuthentication = errors.New("authentication required")
	ErrNotFound       = errors.New("resource not found")
	ErrRateLimit      = errors.New("rate limit exceeded")
	ErrBadRequest     = errors.New("bad request")
)

// VersionConflictError is returned when a note update conflicts.
type VersionConflictError struct {
	ServerVersion int
	ServerNote    *Note
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("version conflict: server version is %d", e.ServerVersion)
}

// APIError represents an error response from the server.
type APIError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error [%s] (status %d): %s", e.Code, e.StatusCode, e.Message)
}
