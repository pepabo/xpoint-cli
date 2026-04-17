package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	meOutput string
	meJQ     string
)

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated X-point user's profile",
	Long: `Fetch the authenticated user's SCIM profile via
GET /scim/v2/{domain_code}/Me.

Requires OAuth2 access (the generic API token does not grant access to
SCIM). domain_code is resolved from --xpoint-domain-code, XPOINT_DOMAIN_CODE,
or the value saved alongside the OAuth token by 'xp auth login'.`,
	RunE: runMe,
}

func init() {
	rootCmd.AddCommand(meCmd)
	meCmd.Flags().StringVarP(&meOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	meCmd.Flags().StringVar(&meJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runMe(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	domain := resolveDomainCode()
	if domain == "" {
		return fmt.Errorf("domain code is required: set --xpoint-domain-code / XPOINT_DOMAIN_CODE, or run 'xp auth login' to store one")
	}

	info, err := client.GetSelfInfo(cmd.Context(), domain)
	if err != nil {
		return err
	}

	return render(info, resolveOutputFormat(meOutput), meJQ, func() error {
		w := newList(os.Stdout)
		w.AddRow("user_code:", info.AtledExt.UserCode)
		w.AddRow("user_name:", info.UserName)
		w.AddRow("display_name:", info.DisplayName)
		w.AddRow("scim_id:", info.ID)
		w.Print()
		return nil
	})
}
