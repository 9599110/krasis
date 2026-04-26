package auth

import "testing"

// M1.3 验收：OAuth 授权 URL 构造正确（GitHub / Google）（见 docs/验收标准.md 1.3）
func TestOAuthManager_GetAuthURL_GitHub(t *testing.T) {
	m := NewOAuthManager(map[Provider]OAuthConfig{
		ProviderGitHub: {
			ClientID:     "cid",
			ClientSecret: "sec",
			RedirectURI:  "https://app/cb",
		},
	})
	u, err := m.GetAuthURL(ProviderGitHub, "state-xyz")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://github.com/login/oauth/authorize?client_id=cid&redirect_uri=https%3A%2F%2Fapp%2Fcb&scope=user%3Aemail&state=state-xyz" {
		t.Fatalf("unexpected url: %q", u)
	}
}

func TestOAuthManager_GetAuthURL_Google(t *testing.T) {
	m := NewOAuthManager(map[Provider]OAuthConfig{
		ProviderGoogle: {
			ClientID:     "gcid",
			ClientSecret: "gsec",
			RedirectURI:  "https://app/gcb",
		},
	})
	u, err := m.GetAuthURL(ProviderGoogle, "st")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://accounts.google.com/o/oauth2/v2/auth?client_id=gcid&redirect_uri=https%3A%2F%2Fapp%2Fgcb&response_type=code&scope=email+profile&state=st" {
		t.Fatalf("unexpected url: %q", u)
	}
}

func TestOAuthManager_GetAuthURL_UnknownProvider(t *testing.T) {
	m := NewOAuthManager(map[Provider]OAuthConfig{})
	_, err := m.GetAuthURL(ProviderGitHub, "x")
	if err != ErrUnknownProvider {
		t.Fatalf("got %v want %v", err, ErrUnknownProvider)
	}
}
