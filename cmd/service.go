package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	serviceOutput string
	serviceJQ     string
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Show X-point version and capability info",
	Long: `Fetch X-point version/feature info via GET /x/v1/service.

The endpoint itself does not require authentication, but this CLI
still resolves a subdomain from the usual flags/env. Useful to check
whether the configured subdomain is single-domain (affects OAuth2
access token requests).`,
	RunE: runService,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.Flags().StringVarP(&serviceOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	serviceCmd.Flags().StringVar(&serviceJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runService(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.GetServiceInfo(cmd.Context())
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(serviceOutput), serviceJQ, func() error {
		w := newList(os.Stdout)
		w.AddRow("version:", res.Version)
		w.AddRow("api_level:", res.APILevel)
		w.AddRow("single_domain:", res.SingleDomain)
		w.AddRow("features:", strings.Join(res.Features, ", "))
		w.Print()
		return nil
	})
}
