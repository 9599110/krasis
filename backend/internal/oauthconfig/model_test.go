package oauthconfig

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOAuthProvider_Model(t *testing.T) {
	now := time.Now()
	provider := &OAuthProvider{
		ID:           uuid.New(),
		Provider:     "github",
		ClientID:     "client-123",
		ClientSecret: "secret-456",
		RedirectURL:  "https://example.com/callback",
		IsEnabled:    true,
		Scopes:       "read:user,user:email",
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if provider.Provider != "github" {
		t.Fatalf("provider mismatch")
	}
	if provider.ClientID != "client-123" {
		t.Fatalf("clientID mismatch")
	}
	if !provider.IsEnabled {
		t.Fatal("IsEnabled should be true")
	}
}

func TestOAuthProvider_JSON(t *testing.T) {
	provider := &OAuthProvider{
		ID:           uuid.New(),
		Provider:     "google",
		ClientID:     "google-client",
		IsEnabled:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	data, err := json.Marshal(provider)
	if err != nil {
		t.Fatal(err)
	}

	var decoded OAuthProvider
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Provider != "google" {
		t.Fatalf("provider mismatch after JSON roundtrip")
	}
}

func TestOAuthConfig_Model(t *testing.T) {
	config := &OAuthConfig{
		ID:         uuid.New(),
		ConfigKey:  "github",
		ConfigValue: `{"enabled":true,"client_id":"test"}`,
		IsSystem:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if config.ConfigKey != "github" {
		t.Fatalf("configKey mismatch")
	}
	if !config.IsSystem {
		t.Fatal("IsSystem should be true")
	}
}

func TestOAuthConfig_JSON(t *testing.T) {
	config := &OAuthConfig{
		ID:         uuid.New(),
		ConfigKey:  "github",
		ConfigValue: `{"enabled":true}`,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}

	var decoded OAuthConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ConfigKey != "github" {
		t.Fatalf("configKey mismatch")
	}
}

func TestOAuthProvider_NullFields(t *testing.T) {
	provider := &OAuthProvider{
		ID:   uuid.New(),
	}

	// Default values should be empty/zero
	if provider.ClientID != "" {
		t.Fatal("ClientID should be empty by default")
	}
	if provider.IsEnabled {
		t.Fatal("IsEnabled should be false by default")
	}
}
