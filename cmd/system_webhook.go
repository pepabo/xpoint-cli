package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	systemWebhookListFQDN   string
	systemWebhookListOutput string
	systemWebhookListJQ     string

	systemWebhookAddURL     string
	systemWebhookAddRemarks string
	systemWebhookAddJQ      string

	systemWebhookUpdateFQDN    string
	systemWebhookUpdateURL     string
	systemWebhookUpdateRemarks string
	systemWebhookUpdateJQ      string

	systemWebhookDeleteFQDN string
)

var systemWebhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage X-point form webhooks (admin)",
}

var systemWebhookListCmd = &cobra.Command{
	Use:   "list <form_code|form_id>",
	Short: "List webhook configs for a form (admin)",
	Long: `List webhook configs via GET /api/v1/system/{fid}/webhooks.

The argument may be a form_code or a numeric form_id. --fqdn is
required and only configs whose destination URL FQDN matches are
returned. Requires an administrator account.`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemWebhookList,
}

var systemWebhookAddCmd = &cobra.Command{
	Use:   "add <form_code|form_id>",
	Short: "Register a webhook config for a form (admin)",
	Long: `Register a webhook config via POST /api/v1/system/{fid}/webhooks.

The argument may be a form_code or a numeric form_id.`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemWebhookAdd,
}

var systemWebhookUpdateCmd = &cobra.Command{
	Use:   "update <form_code|form_id> <webhook_id>",
	Short: "Update a webhook config (admin)",
	Long: `Update an existing webhook config via
PATCH /api/v1/system/{fid}/webhooks/{webhookId}.

--fqdn is required and must match the current destination URL FQDN.
Omit --url / --remarks to leave them unchanged.`,
	Args: cobra.ExactArgs(2),
	RunE: runSystemWebhookUpdate,
}

var systemWebhookDeleteCmd = &cobra.Command{
	Use:   "delete <form_code|form_id> <webhook_id>",
	Short: "Delete a webhook config (admin)",
	Long: `Delete a webhook config via
DELETE /api/v1/system/{fid}/webhooks/{webhookId}.

--fqdn is required and must match the current destination URL FQDN.`,
	Args: cobra.ExactArgs(2),
	RunE: runSystemWebhookDelete,
}

func init() {
	systemCmd.AddCommand(systemWebhookCmd)
	systemWebhookCmd.AddCommand(systemWebhookListCmd)
	systemWebhookCmd.AddCommand(systemWebhookAddCmd)
	systemWebhookCmd.AddCommand(systemWebhookUpdateCmd)
	systemWebhookCmd.AddCommand(systemWebhookDeleteCmd)

	lf := systemWebhookListCmd.Flags()
	lf.StringVar(&systemWebhookListFQDN, "fqdn", "", "destination URL FQDN to filter by (required, e.g. example.com)")
	lf.StringVarP(&systemWebhookListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	lf.StringVar(&systemWebhookListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
	_ = systemWebhookListCmd.MarkFlagRequired("fqdn")

	af := systemWebhookAddCmd.Flags()
	af.StringVar(&systemWebhookAddURL, "url", "", "destination URL (required)")
	af.StringVar(&systemWebhookAddRemarks, "remarks", "", "remarks")
	af.StringVar(&systemWebhookAddJQ, "jq", "", "apply a gojq filter to the JSON response")
	_ = systemWebhookAddCmd.MarkFlagRequired("url")

	uf := systemWebhookUpdateCmd.Flags()
	uf.StringVar(&systemWebhookUpdateFQDN, "fqdn", "", "current destination URL FQDN (required, e.g. example.com)")
	uf.StringVar(&systemWebhookUpdateURL, "url", "", "new destination URL (omit to leave unchanged)")
	uf.StringVar(&systemWebhookUpdateRemarks, "remarks", "", "new remarks (omit to leave unchanged)")
	uf.StringVar(&systemWebhookUpdateJQ, "jq", "", "apply a gojq filter to the JSON response")
	_ = systemWebhookUpdateCmd.MarkFlagRequired("fqdn")

	df := systemWebhookDeleteCmd.Flags()
	df.StringVar(&systemWebhookDeleteFQDN, "fqdn", "", "destination URL FQDN of the config to delete (required)")
	_ = systemWebhookDeleteCmd.MarkFlagRequired("fqdn")
}

func runSystemWebhookList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	formID, err := resolveSystemFormID(cmd.Context(), client, args[0])
	if err != nil {
		return err
	}
	res, err := client.ListWebhookConfigs(cmd.Context(), formID, systemWebhookListFQDN)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(systemWebhookListOutput), systemWebhookListJQ, func() error {
		fmt.Fprintf(os.Stdout, "FORM: %s  TYPE: %s\n", res.FormName, res.FormType)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "ID\tURL\tREMARKS")
		for _, h := range res.Webhooks {
			fmt.Fprintf(w, "%d\t%s\t%s\n", h.ID, h.URL, h.Remarks)
		}
		return nil
	})
}

func runSystemWebhookAdd(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	formID, err := resolveSystemFormID(cmd.Context(), client, args[0])
	if err != nil {
		return err
	}
	req := xpoint.CreateWebhookRequest{
		URL:     systemWebhookAddURL,
		Remarks: systemWebhookAddRemarks,
	}
	res, err := client.CreateWebhookConfig(cmd.Context(), formID, req)
	if err != nil {
		return err
	}
	if systemWebhookAddJQ != "" {
		return runJQ(res, systemWebhookAddJQ)
	}
	return writeJSON(os.Stdout, res)
}

func runSystemWebhookUpdate(cmd *cobra.Command, args []string) error {
	webhookID := strings.TrimSpace(args[1])
	if webhookID == "" {
		return fmt.Errorf("webhook_id is required")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	formID, err := resolveSystemFormID(cmd.Context(), client, args[0])
	if err != nil {
		return err
	}

	req := xpoint.UpdateWebhookRequest{FQDN: systemWebhookUpdateFQDN}
	if cmd.Flags().Changed("url") {
		v := systemWebhookUpdateURL
		req.URL = &v
	}
	if cmd.Flags().Changed("remarks") {
		v := systemWebhookUpdateRemarks
		req.Remarks = &v
	}

	res, err := client.UpdateWebhookConfig(cmd.Context(), formID, webhookID, req)
	if err != nil {
		return err
	}
	if systemWebhookUpdateJQ != "" {
		return runJQ(res, systemWebhookUpdateJQ)
	}
	return writeJSON(os.Stdout, res)
}

func runSystemWebhookDelete(cmd *cobra.Command, args []string) error {
	webhookID := strings.TrimSpace(args[1])
	if webhookID == "" {
		return fmt.Errorf("webhook_id is required")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	formID, err := resolveSystemFormID(cmd.Context(), client, args[0])
	if err != nil {
		return err
	}
	out, err := client.DeleteWebhookConfig(cmd.Context(), formID, webhookID, systemWebhookDeleteFQDN)
	if err != nil {
		return err
	}
	if len(out) == 0 {
		fmt.Fprintln(os.Stdout, "deleted")
		return nil
	}
	return writeJSON(os.Stdout, out)
}
