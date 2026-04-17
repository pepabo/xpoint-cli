package xpoint

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGenerateCodeVerifier_LengthAndCharset(t *testing.T) {
	v, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier: %v", err)
	}
	if len(v) < 43 || len(v) > 128 {
		t.Errorf("verifier length = %d, want 43..128", len(v))
	}
	for _, r := range v {
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '~'
		if !ok {
			t.Errorf("verifier contains invalid character %q", r)
		}
	}
}

func TestCodeChallengeS256_MatchesSHA256(t *testing.T) {
	verifier := "test-verifier-12345"
	got := CodeChallengeS256(verifier)
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if got != want {
		t.Errorf("challenge = %q, want %q", got, want)
	}
	if strings.ContainsAny(got, "+/=") {
		t.Errorf("challenge must use base64url without padding, got %q", got)
	}
}

func TestAuthEndpoint_WithAndWithoutDomainCode(t *testing.T) {
	c := &OAuthConfig{Subdomain: "acme", ClientID: "cid"}
	if got := c.AuthEndpoint(); got != "https://acme.atledcloud.jp/xpoint/a/v1/oauth2/auth" {
		t.Errorf("auth endpoint = %q", got)
	}
	c.DomainCode = "dom1"
	if got := c.AuthEndpoint(); got != "https://acme.atledcloud.jp/xpoint/a/v1/oauth2/auth/dom1" {
		t.Errorf("auth endpoint with domain = %q", got)
	}
}

func TestBuildAuthURL_Params(t *testing.T) {
	c := &OAuthConfig{Subdomain: "acme", ClientID: "cid"}
	raw := c.BuildAuthURL("http://127.0.0.1:1234/callback", "challenge-x", "state-y")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	checks := map[string]string{
		"response_type":         "code",
		"client_id":             "cid",
		"redirect_uri":          "http://127.0.0.1:1234/callback",
		"code_challenge_method": "S256",
		"code_challenge":        "challenge-x",
		"state":                 "state-y",
	}
	for k, want := range checks {
		if got := q.Get(k); got != want {
			t.Errorf("query[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestExchangeCode_FormAndDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/a/v1/oauth2/token" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse body: %v", err)
		}
		wants := map[string]string{
			"grant_type":    "authorization_code",
			"code":          "the-code",
			"redirect_uri":  "http://127.0.0.1:1/callback",
			"code_verifier": "the-verifier",
			"client_id":     "cid",
		}
		for k, want := range wants {
			if got := form.Get(k); got != want {
				t.Errorf("form[%q] = %q, want %q", k, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"AT","token_type":"bearer","expires_in":3600,"refresh_token":"RT"}`))
	}))
	defer srv.Close()

	c := &OAuthConfig{Subdomain: "acme", ClientID: "cid"}
	c.HTTPClient = srv.Client()
	c.overrideTokenURL(srv.URL + "/a/v1/oauth2/token")

	tok, err := c.ExchangeCode(context.Background(), "the-code", "the-verifier", "http://127.0.0.1:1/callback")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tok.AccessToken != "AT" || tok.RefreshToken != "RT" || tok.TokenType != "bearer" || tok.ExpiresIn != 3600 {
		t.Errorf("unexpected token: %+v", tok)
	}
	if tok.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set when ExpiresIn > 0")
	}
}

func TestRefreshToken_Form(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		if form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q", form.Get("grant_type"))
		}
		if form.Get("refresh_token") != "RT-old" {
			t.Errorf("refresh_token = %q", form.Get("refresh_token"))
		}
		if form.Get("client_id") != "cid" {
			t.Errorf("client_id = %q", form.Get("client_id"))
		}
		_, _ = w.Write([]byte(`{"access_token":"AT2","token_type":"bearer","expires_in":3600,"refresh_token":"RT2"}`))
	}))
	defer srv.Close()

	c := &OAuthConfig{Subdomain: "acme", ClientID: "cid"}
	c.HTTPClient = srv.Client()
	c.overrideTokenURL(srv.URL + "/a/v1/oauth2/token")

	tok, err := c.RefreshToken(context.Background(), "RT-old")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if tok.AccessToken != "AT2" || tok.RefreshToken != "RT2" {
		t.Errorf("unexpected token: %+v", tok)
	}
}

func TestPostToken_ErrorSurfacesStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	c := &OAuthConfig{Subdomain: "acme", ClientID: "cid"}
	c.HTTPClient = srv.Client()
	c.overrideTokenURL(srv.URL + "/a/v1/oauth2/token")

	_, err := c.RefreshToken(context.Background(), "rt")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("error should mention status and body: %v", err)
	}
}

func TestTokenExpired(t *testing.T) {
	tok := &Token{}
	if tok.Expired() {
		t.Error("zero ExpiresAt should not be Expired")
	}
	tok.ExpiresAt = time.Now().Add(time.Hour)
	if tok.Expired() {
		t.Error("future ExpiresAt should not be Expired")
	}
	tok.ExpiresAt = time.Now().Add(-time.Hour)
	if !tok.Expired() {
		t.Error("past ExpiresAt should be Expired")
	}
}

// TestAuthorizationCodeFlow_EndToEnd exercises the full flow with a fake
// "browser" that performs the redirect callback synchronously after the auth
// URL is opened, plus an httptest token endpoint.
func TestAuthorizationCodeFlow_EndToEnd(t *testing.T) {
	var capturedRedirectURI, capturedCodeVerifier string

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		capturedRedirectURI = form.Get("redirect_uri")
		capturedCodeVerifier = form.Get("code_verifier")
		if form.Get("code") != "auth-code-123" {
			t.Errorf("code = %q", form.Get("code"))
		}
		_, _ = w.Write([]byte(`{"access_token":"AT","token_type":"bearer","expires_in":3600,"refresh_token":"RT"}`))
	}))
	defer tokenSrv.Close()

	c := &OAuthConfig{Subdomain: "acme", ClientID: "cid"}
	c.HTTPClient = tokenSrv.Client()
	c.overrideTokenURL(tokenSrv.URL + "/token")

	openBrowser := func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			t.Errorf("parse authURL: %v", err)
			return err
		}
		q := u.Query()
		redirect := q.Get("redirect_uri")
		state := q.Get("state")

		// Simulate the browser hitting the redirect URI after user consent.
		go func() {
			cb, _ := url.Parse(redirect)
			cbQ := cb.Query()
			cbQ.Set("code", "auth-code-123")
			cbQ.Set("state", state)
			cb.RawQuery = cbQ.Encode()
			resp, err := http.Get(cb.String())
			if err != nil {
				t.Errorf("callback GET: %v", err)
				return
			}
			_ = resp.Body.Close()
		}()
		return nil
	}

	tok, err := c.AuthorizationCodeFlow(context.Background(), openBrowser)
	if err != nil {
		t.Fatalf("AuthorizationCodeFlow: %v", err)
	}
	if tok.AccessToken != "AT" || tok.RefreshToken != "RT" {
		t.Errorf("unexpected token: %+v", tok)
	}
	if !strings.HasPrefix(capturedRedirectURI, "http://127.0.0.1:") || !strings.HasSuffix(capturedRedirectURI, "/callback") {
		t.Errorf("redirect URI passed to token endpoint = %q", capturedRedirectURI)
	}
	if capturedCodeVerifier == "" {
		t.Error("code_verifier should be sent to token endpoint")
	}
}

func TestAuthorizationCodeFlow_StateMismatch(t *testing.T) {
	c := &OAuthConfig{Subdomain: "acme", ClientID: "cid"}
	c.overrideTokenURL("http://unused.invalid/token")

	openBrowser := func(authURL string) error {
		u, _ := url.Parse(authURL)
		redirect := u.Query().Get("redirect_uri")
		go func() {
			cb, _ := url.Parse(redirect)
			q := cb.Query()
			q.Set("code", "x")
			q.Set("state", "WRONG")
			cb.RawQuery = q.Encode()
			resp, err := http.Get(cb.String())
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}

	_, err := c.AuthorizationCodeFlow(context.Background(), openBrowser)
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("err = %v, want state mismatch", err)
	}
}
