package xpoint

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthConfig holds the parameters required to drive the X-point OAuth2
// Authorization Code + PKCE flow.
type OAuthConfig struct {
	Subdomain  string
	DomainCode string // optional: appended to the auth endpoint path when set
	ClientID   string
	HTTPClient *http.Client

	// tokenURL overrides TokenEndpoint when non-empty. Used by tests.
	tokenURL string
}

// overrideTokenURL is a test hook to point the token endpoint at an httptest
// server without having to construct a full https URL through Subdomain.
func (c *OAuthConfig) overrideTokenURL(u string) { c.tokenURL = u }

// Token represents a token response from the X-point token endpoint.
// ExpiresAt is computed locally from ExpiresIn at receive time.
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// Expired reports whether the access token is past its ExpiresAt with a small
// safety margin so a request that fires "right now" still sees a valid token.
func (t *Token) Expired() bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(t.ExpiresAt.Add(-30 * time.Second))
}

func (c *OAuthConfig) baseURL() string {
	return fmt.Sprintf("https://%s.atledcloud.jp/xpoint", c.Subdomain)
}

// AuthEndpoint returns the authorization endpoint URL.
func (c *OAuthConfig) AuthEndpoint() string {
	if c.DomainCode != "" {
		return fmt.Sprintf("%s/a/v1/oauth2/auth/%s", c.baseURL(), c.DomainCode)
	}
	return fmt.Sprintf("%s/a/v1/oauth2/auth", c.baseURL())
}

// TokenEndpoint returns the token endpoint URL.
func (c *OAuthConfig) TokenEndpoint() string {
	if c.tokenURL != "" {
		return c.tokenURL
	}
	return fmt.Sprintf("%s/a/v1/oauth2/token", c.baseURL())
}

func (c *OAuthConfig) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// GenerateCodeVerifier returns a cryptographically random PKCE code verifier
// matching the X-point spec (43..128 chars from [A-Za-z0-9-._~]).
// 32 random bytes Base64URL-encoded yields 43 chars.
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CodeChallengeS256 derives the S256 code challenge from a verifier.
func CodeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// BuildAuthURL constructs the authorization request URL with PKCE parameters.
func (c *OAuthConfig) BuildAuthURL(redirectURI, codeChallenge, state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("code_challenge_method", "S256")
	q.Set("code_challenge", codeChallenge)
	if state != "" {
		q.Set("state", state)
	}
	return c.AuthEndpoint() + "?" + q.Encode()
}

// ExchangeCode trades an authorization code for a token.
func (c *OAuthConfig) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", codeVerifier)
	form.Set("client_id", c.ClientID)
	return c.postToken(ctx, form)
}

// RefreshToken exchanges a refresh token for a new access (and refresh) token.
// Per the X-point spec, the previous refresh token is invalidated by the server
// once a successful refresh occurs.
func (c *OAuthConfig) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", c.ClientID)
	return c.postToken(ctx, form)
}

func (c *OAuthConfig) postToken(ctx context.Context, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("token endpoint error: %s: %s", resp.Status, string(body))
	}

	var tok Token
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if tok.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	return &tok, nil
}

// AuthorizationCodeFlow runs the full Authorization Code + PKCE flow against
// the user's browser. It binds a localhost listener on a random free port,
// returns the authorization URL via openBrowser, then blocks until either the
// callback fires or ctx is cancelled. On success, the resulting tokens are
// returned. The redirect URI registered with X-point must equal
// http://127.0.0.1:<port>/callback (any port).
func (c *OAuthConfig) AuthorizationCodeFlow(ctx context.Context, openBrowser func(url string) error) (*Token, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on localhost: %w", err)
	}
	defer func() { _ = listener.Close() }()

	redirectURI := fmt.Sprintf("http://%s/callback", listener.Addr().String())

	verifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate code verifier: %w", err)
	}
	challenge := CodeChallengeS256(verifier)
	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	type result struct {
		code string
		err  error
	}
	resultCh := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errCode := q.Get("error"); errCode != "" {
			msg := errCode
			if desc := q.Get("error_description"); desc != "" {
				msg = fmt.Sprintf("%s: %s", errCode, desc)
			}
			writeCallbackPage(w, false, msg)
			resultCh <- result{err: fmt.Errorf("authorization error: %s", msg)}
			return
		}
		if got := q.Get("state"); got != state {
			writeCallbackPage(w, false, "state mismatch")
			resultCh <- result{err: errors.New("state mismatch in authorization callback")}
			return
		}
		code := q.Get("code")
		if code == "" {
			writeCallbackPage(w, false, "missing code")
			resultCh <- result{err: errors.New("authorization callback did not include code")}
			return
		}
		writeCallbackPage(w, true, "")
		resultCh <- result{code: code}
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := c.BuildAuthURL(redirectURI, challenge, state)
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		return c.ExchangeCode(ctx, res.code, verifier, redirectURI)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func writeCallbackPage(w http.ResponseWriter, ok bool, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ok {
		_, _ = io.WriteString(w, `<!doctype html><html><head><meta charset="utf-8"><title>xpoint-cli</title></head><body style="font-family:sans-serif"><h1>認証が完了しました</h1><p>このウィンドウを閉じてターミナルに戻ってください。</p></body></html>`)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>xpoint-cli</title></head><body style="font-family:sans-serif"><h1>認証に失敗しました</h1><pre>%s</pre></body></html>`, msg)
}
