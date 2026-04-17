package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	adminroleOutput string
	adminroleJQ     string
)

var adminroleCmd = &cobra.Command{
	Use:   "adminrole",
	Short: "Show the authenticated user's admin roles",
	Long: `Fetch the authenticated user's admin role list via
GET /api/v1/adminrole. Returns an empty list for general users.`,
	RunE: runAdminrole,
}

func init() {
	rootCmd.AddCommand(adminroleCmd)
	adminroleCmd.Flags().StringVarP(&adminroleOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	adminroleCmd.Flags().StringVar(&adminroleJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runAdminrole(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.GetAdminRole(cmd.Context())
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(adminroleOutput), adminroleJQ, func() error {
		if len(res.Role) == 0 {
			fmt.Fprintln(os.Stdout, "(no admin roles)")
			return nil
		}
		fmt.Fprintln(os.Stdout, strings.Join(res.Role, "\n"))
		return nil
	})
}
