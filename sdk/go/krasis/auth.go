package krasis

import (
	"net/url"
)

// AuthModule handles authentication operations.
type AuthModule struct {
	client *Client
}

// NewAuthModule creates a new auth module.
func NewAuthModule(c *Client) *AuthModule {
	return &AuthModule{client: c}
}

// OAuthURL returns the OAuth authorization URL for the given provider.
func (a *AuthModule) OAuthURL(provider string, redirectURI, state string) string {
	params := url.Values{}
	params.Set("provider", provider)
	if redirectURI != "" {
		params.Set("redirect_uri", redirectURI)
	}
	if state != "" {
		params.Set("state", state)
	}
	return a.client.baseURL + "/auth/oauth?" + params.Encode()
}

// Callback exchanges the OAuth code for a token.
func (a *AuthModule) Callback(provider, code, state string) (map[string]any, error) {
	body := map[string]string{"provider": provider, "code": code}
	if state != "" {
		body["state"] = state
	}
	var result map[string]any
	err := a.client.Post("/auth/oauth/callback", body, &result)
	return result, err
}

// Login authenticates with email and password, storing the token.
func (a *AuthModule) Login(email, password string) error {
	var result struct {
		Token string `json:"token"`
	}
	err := a.client.Post("/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, &result)
	if err != nil {
		return err
	}
	a.client.SetToken(result.Token)
	return nil
}

// Register creates a new user account.
func (a *AuthModule) Register(email, password, username string) error {
	return a.client.Post("/auth/register", map[string]string{
		"email":    email,
		"password": password,
		"username": username,
	}, nil)
}

// Logout invalidates the current session and clears the token.
func (a *AuthModule) Logout() error {
	defer a.client.ClearToken()
	return a.client.Post("/auth/logout", nil, nil)
}

// Me fetches the current user profile.
func (a *AuthModule) Me() (*User, error) {
	var user User
	err := a.client.Get("/auth/me", &user)
	return &user, err
}

// UserModule handles user session and profile operations.
type UserModule struct {
	client *Client
}

// NewUserModule creates a new user module.
func NewUserModule(c *Client) *UserModule {
	return &UserModule{client: c}
}

// Sessions returns the list of active sessions for the current user.
func (u *UserModule) Sessions() ([]Session, error) {
	var sessions []Session
	err := u.client.Get("/user/sessions", &sessions)
	return sessions, err
}

// RevokeSession terminates a specific session by ID.
func (u *UserModule) RevokeSession(sessionID string) error {
	return u.client.Delete("/user/sessions/"+sessionID, nil)
}

// UpdateProfile updates the current user's profile.
func (u *UserModule) UpdateProfile(username, avatarURL string) error {
	body := map[string]any{}
	if username != "" {
		body["username"] = username
	}
	if avatarURL != "" {
		body["avatar_url"] = avatarURL
	}
	return u.client.Put("/user/profile", body, nil, nil)
}
