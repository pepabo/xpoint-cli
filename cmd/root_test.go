package cmd

import (
	"strings"
	"testing"
)

func resetAuthFlags() {
	flagSubdomain = ""
	flagDomainCode = ""
	flagUser = ""
	flagGenericAPIToken = ""
	flagAPIAccessToken = ""
}

func TestResolveAuth_AccessTokenFromEnv(t *testing.T) {
	resetAuthFlags()
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "tok")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "")
	t.Setenv("XPOINT_DOMAIN_CODE", "")
	t.Setenv("XPOINT_USER", "")

	auth, err := resolveAuth()
	if err != nil {
		t.Fatalf("resolveAuth: %v", err)
	}
	if auth.AccessToken != "tok" {
		t.Errorf("AccessToken = %q", auth.AccessToken)
	}
	if auth.GenericAPIToken != "" {
		t.Errorf("GenericAPIToken should be empty, got %q", auth.GenericAPIToken)
	}
}

func TestResolveAuth_GenericFromEnv(t *testing.T) {
	resetAuthFlags()
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "gen")
	t.Setenv("XPOINT_DOMAIN_CODE", "d")
	t.Setenv("XPOINT_USER", "u")

	auth, err := resolveAuth()
	if err != nil {
		t.Fatalf("resolveAuth: %v", err)
	}
	if auth.DomainCode != "d" || auth.User != "u" || auth.GenericAPIToken != "gen" {
		t.Errorf("unexpected auth: %+v", auth)
	}
}

func TestResolveAuth_BothSetIsError(t *testing.T) {
	resetAuthFlags()
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "tok")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "gen")
	t.Setenv("XPOINT_DOMAIN_CODE", "d")
	t.Setenv("XPOINT_USER", "u")

	_, err := resolveAuth()
	if err == nil || !strings.Contains(err.Error(), "cannot specify both") {
		t.Errorf("err = %v", err)
	}
}

func TestResolveAuth_GenericMissingDomainOrUser(t *testing.T) {
	resetAuthFlags()
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "gen")
	t.Setenv("XPOINT_DOMAIN_CODE", "")
	t.Setenv("XPOINT_USER", "u")

	_, err := resolveAuth()
	if err == nil || !strings.Contains(err.Error(), "DOMAIN_CODE") {
		t.Errorf("err = %v", err)
	}
}

func TestResolveAuth_NoneSet(t *testing.T) {
	resetAuthFlags()
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "")
	t.Setenv("XPOINT_DOMAIN_CODE", "")
	t.Setenv("XPOINT_USER", "")

	_, err := resolveAuth()
	if err == nil || !strings.Contains(err.Error(), "authentication is required") {
		t.Errorf("err = %v", err)
	}
}

func TestResolveAuth_FlagWinsOverEnv(t *testing.T) {
	resetAuthFlags()
	flagAPIAccessToken = "from-flag"
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "from-env")

	auth, err := resolveAuth()
	if err != nil {
		t.Fatalf("resolveAuth: %v", err)
	}
	if auth.AccessToken != "from-flag" {
		t.Errorf("AccessToken = %q, want from-flag", auth.AccessToken)
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
