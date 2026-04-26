package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Provider string

const (
	ProviderGitHub Provider = "github"
	ProviderGoogle Provider = "google"
)

var ErrUnknownProvider = errors.New("unknown oauth provider")

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type UserInfo struct {
	ProviderUserID string `json:"provider_user_id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	AvatarURL      string `json:"avatar_url"`
}

type OAuthManager struct {
	configs    map[Provider]OAuthConfig
	httpClient *http.Client
}

func NewOAuthManager(configs map[Provider]OAuthConfig) *OAuthManager {
	return &OAuthManager{
		configs:    configs,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *OAuthManager) GetAuthURL(provider Provider, state string) (string, error) {
	config, ok := m.configs[provider]
	if !ok {
		return "", ErrUnknownProvider
	}

	switch provider {
	case ProviderGitHub:
		params := url.Values{}
		params.Set("client_id", config.ClientID)
		params.Set("redirect_uri", config.RedirectURI)
		params.Set("scope", "user:email")
		params.Set("state", state)
		return fmt.Sprintf("https://github.com/login/oauth/authorize?%s", params.Encode()), nil
	case ProviderGoogle:
		params := url.Values{}
		params.Set("client_id", config.ClientID)
		params.Set("redirect_uri", config.RedirectURI)
		params.Set("response_type", "code")
		params.Set("scope", "email profile")
		params.Set("state", state)
		return fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?%s", params.Encode()), nil
	default:
		return "", ErrUnknownProvider
	}
}

func (m *OAuthManager) ExchangeCode(ctx context.Context, provider Provider, code string) (*TokenResponse, error) {
	config, ok := m.configs[provider]
	if !ok {
		return nil, ErrUnknownProvider
	}

	switch provider {
	case ProviderGitHub:
		return m.exchangeGitHubCode(ctx, config, code)
	case ProviderGoogle:
		return m.exchangeGoogleCode(ctx, config, code)
	default:
		return nil, ErrUnknownProvider
	}
}

func (m *OAuthManager) GetUserInfo(ctx context.Context, provider Provider, accessToken string) (*UserInfo, error) {
	switch provider {
	case ProviderGitHub:
		return m.getGitHubUserInfo(ctx, accessToken)
	case ProviderGoogle:
		return m.getGoogleUserInfo(ctx, accessToken)
	default:
		return nil, ErrUnknownProvider
	}
}

func (m *OAuthManager) exchangeGitHubCode(ctx context.Context, cfg OAuthConfig, code string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Body = io.NopCloser(nil)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode github token response: %w", err)
	}

	return &tokenResp, nil
}

func (m *OAuthManager) exchangeGoogleCode(ctx context.Context, cfg OAuthConfig, code string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("redirect_uri", cfg.RedirectURI)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = io.NopCloser(nil)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode google token response: %w", err)
	}

	return &tokenResp, nil
}

func (m *OAuthManager) getGitHubUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// If email is null, fetch from emails endpoint
	email := result.Email
	if email == "" {
		email = m.getGitHubPrimaryEmail(ctx, accessToken)
	}

	return &UserInfo{
		ProviderUserID: fmt.Sprintf("github:%d", result.ID),
		Email:          email,
		Username:       result.Login,
		AvatarURL:      result.AvatarURL,
	}, nil
}

func (m *OAuthManager) getGitHubPrimaryEmail(ctx context.Context, accessToken string) string {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return ""
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	return ""
}

func (m *OAuthManager) getGoogleUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		Picture   string `json:"picture"`
		Verified  bool   `json:"verified_email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &UserInfo{
		ProviderUserID: "google:" + result.ID,
		Email:          result.Email,
		Username:       result.Name,
		AvatarURL:      result.Picture,
	}, nil
}
