package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/zalando/go-keyring"
)

func resetAuthFlags() {
	flagSubdomain = ""
	flagDomainCode = ""
	flagUser = ""
	flagGenericAPIToken = ""
	flagAPIAccessToken = ""
	flagClientID = ""
}

func clearAuthEnv(t *testing.T) {
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "")
	t.Setenv("XPOINT_DOMAIN_CODE", "")
	t.Setenv("XPOINT_USER", "")
	t.Setenv("XPOINT_CLIENT_ID", "")
}

func TestAuthFromFlags_AccessToken(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	flagAPIAccessToken = "tok"

	auth, ok, err := authFromFlags()
	if err != nil {
		t.Fatalf("authFromFlags: %v", err)
	}
	if !ok || auth.AccessToken != "tok" {
		t.Errorf("got ok=%v auth=%+v, want ok=true AccessToken=tok", ok, auth)
	}
}

func TestAuthFromFlags_Generic(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	flagGenericAPIToken = "gen"
	flagDomainCode = "d"
	flagUser = "u"

	auth, ok, err := authFromFlags()
	if err != nil {
		t.Fatalf("authFromFlags: %v", err)
	}
	if !ok || auth.GenericAPIToken != "gen" || auth.DomainCode != "d" || auth.User != "u" {
		t.Errorf("unexpected: ok=%v auth=%+v", ok, auth)
	}
}

func TestAuthFromFlags_BothSetIsError(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	flagAPIAccessToken = "tok"
	flagGenericAPIToken = "gen"

	_, _, err := authFromFlags()
	if err == nil || !strings.Contains(err.Error(), "cannot specify both") {
		t.Errorf("err = %v, want 'cannot specify both'", err)
	}
}

func TestAuthFromFlags_GenericMissingDomainOrUser(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	flagGenericAPIToken = "gen"
	flagUser = "u"

	_, ok, err := authFromFlags()
	if err == nil || !strings.Contains(err.Error(), "DOMAIN_CODE") {
		t.Errorf("err = %v", err)
	}
	if ok {
		t.Errorf("ok should be false on error")
	}
}

func TestAuthFromFlags_None(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	auth, ok, err := authFromFlags()
	if err != nil {
		t.Fatalf("authFromFlags: %v", err)
	}
	if ok {
		t.Errorf("ok should be false when no flags set")
	}
	if auth.AccessToken != "" || auth.GenericAPIToken != "" {
		t.Errorf("auth should be zero, got %+v", auth)
	}
}

func TestAuthFromEnv_AccessToken(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "tok")

	auth, ok, err := authFromEnv()
	if err != nil {
		t.Fatalf("authFromEnv: %v", err)
	}
	if !ok || auth.AccessToken != "tok" {
		t.Errorf("got ok=%v auth=%+v, want ok=true AccessToken=tok", ok, auth)
	}
}

func TestAuthFromEnv_Generic(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "gen")
	t.Setenv("XPOINT_DOMAIN_CODE", "d")
	t.Setenv("XPOINT_USER", "u")

	auth, ok, err := authFromEnv()
	if err != nil {
		t.Fatalf("authFromEnv: %v", err)
	}
	if !ok || auth.DomainCode != "d" || auth.User != "u" || auth.GenericAPIToken != "gen" {
		t.Errorf("unexpected: ok=%v auth=%+v", ok, auth)
	}
}

func TestAuthFromEnv_BothSetIsError(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "tok")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "gen")

	_, _, err := authFromEnv()
	if err == nil || !strings.Contains(err.Error(), "cannot specify both") {
		t.Errorf("err = %v, want 'cannot specify both'", err)
	}
}

func TestAuthFromEnv_None(t *testing.T) {
	resetAuthFlags()
	clearAuthEnv(t)
	_, ok, err := authFromEnv()
	if err != nil {
		t.Fatalf("authFromEnv: %v", err)
	}
	if ok {
		t.Errorf("ok should be false when no env set")
	}
}

// TestNewClientFromFlags_Priority verifies the documented precedence order:
// flags > env > keyring. The actual transport behavior of the resulting
// client is not asserted here (covered by xpoint package tests); this only
// exercises which credential source is selected.
func TestNewClientFromFlags_Priority(t *testing.T) {
	t.Run("flag beats env and keyring", func(t *testing.T) {
		resetAuthFlags()
		clearAuthEnv(t)
		keyring.MockInit()
		t.Setenv("XPOINT_SUBDOMAIN", "acme")

		flagAPIAccessToken = "from-flag"
		t.Setenv("XPOINT_API_ACCESS_TOKEN", "from-env")
		_ = xpoint.SaveToken(&xpoint.StoredToken{
			Subdomain: "acme", ClientID: "c", Token: xpoint.Token{AccessToken: "from-keyring"},
		})

		auth, ok, err := authFromFlags()
		if err != nil || !ok || auth.AccessToken != "from-flag" {
			t.Errorf("flag should win: ok=%v auth=%+v err=%v", ok, auth, err)
		}
	})

	t.Run("env beats keyring when no flag", func(t *testing.T) {
		resetAuthFlags()
		clearAuthEnv(t)
		keyring.MockInit()
		t.Setenv("XPOINT_SUBDOMAIN", "acme")

		t.Setenv("XPOINT_API_ACCESS_TOKEN", "from-env")
		_ = xpoint.SaveToken(&xpoint.StoredToken{
			Subdomain: "acme", ClientID: "c", Token: xpoint.Token{AccessToken: "from-keyring"},
		})

		_, ok, err := authFromFlags()
		if err != nil || ok {
			t.Fatalf("authFromFlags should return ok=false when no flag: ok=%v err=%v", ok, err)
		}
		auth, ok, err := authFromEnv()
		if err != nil || !ok || auth.AccessToken != "from-env" {
			t.Errorf("env should be picked: ok=%v auth=%+v err=%v", ok, auth, err)
		}
	})

	t.Run("keyring used when no flag and no env", func(t *testing.T) {
		resetAuthFlags()
		clearAuthEnv(t)
		keyring.MockInit()
		t.Setenv("XPOINT_SUBDOMAIN", "acme")
		_ = xpoint.SaveToken(&xpoint.StoredToken{
			Subdomain: "acme", ClientID: "c", Token: xpoint.Token{AccessToken: "from-keyring"},
		})

		_, ok, _ := authFromFlags()
		if ok {
			t.Fatal("authFromFlags ok should be false")
		}
		_, ok, _ = authFromEnv()
		if ok {
			t.Fatal("authFromEnv ok should be false")
		}
		auth, err := loadStoredTokenAuth(context.Background(), "acme")
		if err != nil || auth.AccessToken != "from-keyring" {
			t.Errorf("keyring fallback failed: auth=%+v err=%v", auth, err)
		}
	})
}

func TestLoadStoredTokenAuth_NotFound(t *testing.T) {
	keyring.MockInit()
	_, err := loadStoredTokenAuth(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "authentication is required") {
		t.Errorf("err = %v, want authentication-required message", err)
	}
}

func TestResolveSubdomain(t *testing.T) {
	resetAuthFlags()
	t.Setenv("XPOINT_SUBDOMAIN", "sub1")
	sub, err := resolveSubdomain()
	if err != nil {
		t.Fatalf("resolveSubdomain: %v", err)
	}
	if sub != "sub1" {
		t.Errorf("sub = %q", sub)
	}

	resetAuthFlags()
	t.Setenv("XPOINT_SUBDOMAIN", "")
	if _, err := resolveSubdomain(); err == nil {
		t.Error("expected error when subdomain missing")
	}
}
