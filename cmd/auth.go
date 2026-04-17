package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	flagClientID  string
	flagNoBrowser bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage X-point OAuth2 authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via OAuth2 Authorization Code + PKCE flow",
	Long: `Run the X-point OAuth2 Authorization Code + PKCE flow.

The CLI binds an HTTP listener on a random free localhost port, opens the
browser to the X-point authorization endpoint, and waits for the redirect
back. The redirect URI registered for the X-point client must allow
http://127.0.0.1 with any port and the path /callback.

On success, the resulting access and refresh tokens are saved to the system
keyring (Secret Service / Keychain / Credential Manager) keyed by subdomain
and re-used by subsequent commands.`,
	RunE: runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the OAuth token saved for the current subdomain",
	RunE:  runAuthStatus,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)

	f := authLoginCmd.Flags()
	f.StringVar(&flagClientID, "xpoint-client-id", "", "X-point OAuth2 client ID (env: XPOINT_CLIENT_ID)")
	f.BoolVar(&flagNoBrowser, "no-browser", false, "do not launch the browser; print the URL and wait")
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	sub, err := resolveSubdomain()
	if err != nil {
		return err
	}
	clientID := pick(flagClientID, "XPOINT_CLIENT_ID")
	if clientID == "" {
		return errors.New("client ID is required: set --xpoint-client-id or XPOINT_CLIENT_ID")
	}
	domainCode := pick(flagDomainCode, "XPOINT_DOMAIN_CODE")
	if domainCode == "" {
		return errors.New("domain code is required for OAuth2 login: set --xpoint-domain-code or XPOINT_DOMAIN_CODE")
	}

	cfg := &xpoint.OAuthConfig{
		Subdomain:  sub,
		DomainCode: domainCode,
		ClientID:   clientID,
	}

	open := func(url string) error {
		if flagNoBrowser {
			fmt.Fprintf(os.Stdout, "Open the following URL in your browser:\n\n  %s\n\n", url)
			return nil
		}
		fmt.Fprintf(os.Stdout, "Opening browser for X-point authorization...\n  %s\n", url)
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stdout, "could not launch browser (%v); open the URL manually.\n", err)
		}
		return nil
	}

	tok, err := cfg.AuthorizationCodeFlow(cmd.Context(), open)
	if err != nil {
		return err
	}

	stored := &xpoint.StoredToken{
		Subdomain:  sub,
		DomainCode: domainCode,
		ClientID:   clientID,
		Token:      *tok,
	}
	if err := xpoint.SaveToken(stored); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Authentication successful. Token saved to system keyring (subdomain=%s).\n", sub)
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	sub, err := resolveSubdomain()
	if err != nil {
		return err
	}
	stored, err := xpoint.LoadToken(sub)
	if err != nil {
		if errors.Is(err, xpoint.ErrTokenNotFound) {
			fmt.Fprintf(os.Stdout, "subdomain: %s\nstatus:    not logged in (run 'xp auth login')\n", sub)
			return nil
		}
		return err
	}

	fmt.Fprintf(os.Stdout, "subdomain:     %s\n", stored.Subdomain)
	fmt.Fprintf(os.Stdout, "domain_code:   %s\n", stored.DomainCode)
	fmt.Fprintf(os.Stdout, "client_id:     %s\n", stored.ClientID)
	fmt.Fprintf(os.Stdout, "access_token:  %s\n", maskedTokenIfPresent(stored.AccessToken))
	fmt.Fprintf(os.Stdout, "refresh_token: %s\n", maskedTokenIfPresent(stored.RefreshToken))
	fmt.Fprintf(os.Stdout, "token_type:    %s\n", stored.TokenType)
	if !stored.ExpiresAt.IsZero() {
		fmt.Fprintf(os.Stdout, "expires_at:    %s\n", stored.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
		fmt.Fprintf(os.Stdout, "expired:       %v\n", stored.Expired())
	}
	return nil
}

// maskedTokenIfPresent returns "***" when the token is set, "" otherwise.
// We never reveal any portion of the token, even masked, to keep secrets out
// of terminal scrollback and screen-shares.
func maskedTokenIfPresent(s string) string {
	if s == "" {
		return ""
	}
	return "***"
}

// openBrowser launches the default browser for url.
func openBrowser(url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
	case "windows":
		name = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		name = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(name, args...).Start()
}
