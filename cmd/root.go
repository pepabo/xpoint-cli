package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	flagSubdomain       string
	flagDomainCode      string
	flagUser            string
	flagGenericAPIToken string
	flagAPIAccessToken  string
)

var version = "1.1.2"

var rootCmd = &cobra.Command{
	Use:          "xp",
	Short:        "X-point CLI",
	Long:         "xp is a CLI client for X-point (https://atled-workflow.github.io/X-point-doc/api/).",
	SilenceUsage: true,
	Version:      version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	f := rootCmd.PersistentFlags()
	f.StringVar(&flagSubdomain, "xpoint-subdomain", "", "X-point subdomain (env: XPOINT_SUBDOMAIN)")
	f.StringVar(&flagDomainCode, "xpoint-domain-code", "", "X-point domain code, used with generic API token and OAuth2 login (env: XPOINT_DOMAIN_CODE)")
	f.StringVar(&flagUser, "xpoint-user", "", "X-point user code, used with generic API token (env: XPOINT_USER)")
	f.StringVar(&flagGenericAPIToken, "xpoint-generic-api-token", "", "X-point generic API token, sent as X-ATLED-Generic-API-Token (env: XPOINT_GENERIC_API_TOKEN)")
	f.StringVar(&flagAPIAccessToken, "xpoint-api-access-token", "", "X-point OAuth2 access token, sent as Authorization: Bearer (env: XPOINT_API_ACCESS_TOKEN)")
}

func pick(flagVal, envKey string) string {
	if flagVal != "" {
		return flagVal
	}
	return os.Getenv(envKey)
}

func resolveSubdomain() (string, error) {
	if s := pick(flagSubdomain, "XPOINT_SUBDOMAIN"); s != "" {
		return s, nil
	}
	if t, err := xpoint.LoadToken(); err == nil && t.Subdomain != "" {
		return t.Subdomain, nil
	}
	return "", errors.New("subdomain is required: set --xpoint-subdomain or XPOINT_SUBDOMAIN, or run 'xp auth login' first")
}

// resolveDomainCode returns the X-point domain code in priority order:
// --xpoint-domain-code flag, XPOINT_DOMAIN_CODE env, then the domain code
// saved alongside the OAuth token by `xp auth login`. Returns an empty
// string if none is available.
func resolveDomainCode() string {
	if d := pick(flagDomainCode, "XPOINT_DOMAIN_CODE"); d != "" {
		return d
	}
	if t, err := xpoint.LoadToken(); err == nil && t.DomainCode != "" {
		return t.DomainCode
	}
	return ""
}

// authFromFlags returns the credential supplied via --xpoint-* command-line
// flags. The bool is true only when the user explicitly passed flags. This is
// the highest-priority auth source: a one-shot flag should always win.
func authFromFlags() (xpoint.Auth, bool, error) {
	if flagAPIAccessToken != "" && flagGenericAPIToken != "" {
		return xpoint.Auth{}, false, errors.New("cannot specify both --xpoint-api-access-token and --xpoint-generic-api-token")
	}
	if flagAPIAccessToken != "" {
		return xpoint.Auth{AccessToken: flagAPIAccessToken}, true, nil
	}
	if flagGenericAPIToken != "" {
		auth, err := genericAuth(flagGenericAPIToken)
		return auth, err == nil, err
	}
	return xpoint.Auth{}, false, nil
}

// authFromEnv returns the credential supplied via XPOINT_* environment
// variables. This is the lowest-priority auth source: it acts as a fallback
// when neither a flag nor a keyring-saved OAuth token is available.
func authFromEnv() (xpoint.Auth, bool, error) {
	envAccess := os.Getenv("XPOINT_API_ACCESS_TOKEN")
	envGeneric := os.Getenv("XPOINT_GENERIC_API_TOKEN")
	if envAccess != "" && envGeneric != "" {
		return xpoint.Auth{}, false, errors.New("cannot specify both XPOINT_API_ACCESS_TOKEN and XPOINT_GENERIC_API_TOKEN")
	}
	if envAccess != "" {
		return xpoint.Auth{AccessToken: envAccess}, true, nil
	}
	if envGeneric != "" {
		auth, err := genericAuth(envGeneric)
		return auth, err == nil, err
	}
	return xpoint.Auth{}, false, nil
}

// genericAuth assembles a generic-API-token Auth, sourcing the domain and
// user code from the usual flag-or-env path.
func genericAuth(token string) (xpoint.Auth, error) {
	domain := pick(flagDomainCode, "XPOINT_DOMAIN_CODE")
	user := pick(flagUser, "XPOINT_USER")
	if domain == "" || user == "" {
		return xpoint.Auth{}, fmt.Errorf("XPOINT_DOMAIN_CODE and XPOINT_USER are required when using generic API token")
	}
	return xpoint.Auth{
		DomainCode:      domain,
		User:            user,
		GenericAPIToken: token,
	}, nil
}
