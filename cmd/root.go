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

var rootCmd = &cobra.Command{
	Use:          "xp",
	Short:        "X-point CLI",
	Long:         "xp is a CLI client for X-point (https://atled-workflow.github.io/X-point-doc/api/).",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	f := rootCmd.PersistentFlags()
	f.StringVar(&flagSubdomain, "xpoint-subdomain", "", "X-point subdomain (env: XPOINT_SUBDOMAIN)")
	f.StringVar(&flagDomainCode, "xpoint-domain-code", "", "X-point domain code, used with generic API token (env: XPOINT_DOMAIN_CODE)")
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
	s := pick(flagSubdomain, "XPOINT_SUBDOMAIN")
	if s == "" {
		return "", errors.New("subdomain is required: set --xpoint-subdomain or XPOINT_SUBDOMAIN")
	}
	return s, nil
}

func resolveAuth() (xpoint.Auth, error) {
	generic := pick(flagGenericAPIToken, "XPOINT_GENERIC_API_TOKEN")
	access := pick(flagAPIAccessToken, "XPOINT_API_ACCESS_TOKEN")

	if generic != "" && access != "" {
		return xpoint.Auth{}, errors.New("cannot specify both generic API token and access token; set only one of XPOINT_GENERIC_API_TOKEN or XPOINT_API_ACCESS_TOKEN")
	}

	if access != "" {
		return xpoint.Auth{AccessToken: access}, nil
	}

	if generic != "" {
		domain := pick(flagDomainCode, "XPOINT_DOMAIN_CODE")
		user := pick(flagUser, "XPOINT_USER")
		if domain == "" || user == "" {
			return xpoint.Auth{}, fmt.Errorf("XPOINT_DOMAIN_CODE and XPOINT_USER are required when using generic API token")
		}
		return xpoint.Auth{
			DomainCode:      domain,
			User:            user,
			GenericAPIToken: generic,
		}, nil
	}

	return xpoint.Auth{}, errors.New("authentication is required: set XPOINT_GENERIC_API_TOKEN (with XPOINT_DOMAIN_CODE and XPOINT_USER) or XPOINT_API_ACCESS_TOKEN")
}
