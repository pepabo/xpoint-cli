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
	systemWebhooklogListFrom      string
	systemWebhooklogListTo        string
	systemWebhooklogListDocID     int
	systemWebhooklogListFormCode  string
	systemWebhooklogListRouteCode string
	systemWebhooklogListStatus    string
	systemWebhooklogListURL       string
	systemWebhooklogListLimit     int
	systemWebhooklogListOffset    int
	systemWebhooklogListOutput    string
	systemWebhooklogListJQ        string

	systemWebhooklogShowJQ string
)

var systemWebhooklogCmd = &cobra.Command{
	Use:   "webhooklog",
	Short: "Inspect X-point webhook delivery logs (admin)",
}

var systemWebhooklogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhook delivery logs",
	Long: `List webhook delivery logs via GET /api/v1/system/webhooklog.

Filters are all optional. --status accepts all | success | failed.
Requires an administrator account.`,
	RunE: runSystemWebhooklogList,
}

var systemWebhooklogShowCmd = &cobra.Command{
	Use:   "show <uuid>",
	Short: "Show webhook delivery log detail (request/response)",
	Long: `Fetch detailed request/response data for a single webhook log entry via
GET /api/v1/system/webhooklog/{uuid}. Requires an administrator account.

The response is emitted as JSON because the request/response body shape
depends on the remote endpoint. Use --jq to filter the output.`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemWebhooklogShow,
}

func init() {
	systemCmd.AddCommand(systemWebhooklogCmd)
	systemWebhooklogCmd.AddCommand(systemWebhooklogListCmd)
	systemWebhooklogCmd.AddCommand(systemWebhooklogShowCmd)

	lf := systemWebhooklogListCmd.Flags()
	lf.StringVar(&systemWebhooklogListFrom, "from", "", "send date from (e.g. 2022/10/01)")
	lf.StringVar(&systemWebhooklogListTo, "to", "", "send date to (e.g. 2022/12/01)")
	lf.IntVar(&systemWebhooklogListDocID, "docid", 0, "document id (0 = omit)")
	lf.StringVar(&systemWebhooklogListFormCode, "form-code", "", "form code")
	lf.StringVar(&systemWebhooklogListRouteCode, "route-code", "", "approval route code")
	lf.StringVar(&systemWebhooklogListStatus, "status", "", "status: all | success | failed")
	lf.StringVar(&systemWebhooklogListURL, "url", "", "destination URL (partial match)")
	lf.IntVar(&systemWebhooklogListLimit, "limit", 0, "fetch count (0 = omit; server default 50)")
	lf.IntVar(&systemWebhooklogListOffset, "offset", 0, "offset (0 = omit; server default 0)")
	lf.StringVarP(&systemWebhooklogListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	lf.StringVar(&systemWebhooklogListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	sf := systemWebhooklogShowCmd.Flags()
	sf.StringVar(&systemWebhooklogShowJQ, "jq", "", "apply a gojq filter to the JSON response")
}

func runSystemWebhooklogList(cmd *cobra.Command, _ []string) error {
	p := xpoint.WebhooklogListParams{
		From:      systemWebhooklogListFrom,
		To:        systemWebhooklogListTo,
		FormCode:  systemWebhooklogListFormCode,
		RouteCode: systemWebhooklogListRouteCode,
		Status:    systemWebhooklogListStatus,
		URL:       systemWebhooklogListURL,
	}
	if cmd.Flags().Changed("docid") {
		v := systemWebhooklogListDocID
		p.DocID = &v
	}
	if cmd.Flags().Changed("limit") {
		v := systemWebhooklogListLimit
		p.Limit = &v
	}
	if cmd.Flags().Changed("offset") {
		v := systemWebhooklogListOffset
		p.Offset = &v
	}
	if p.Status != "" {
		switch strings.ToLower(p.Status) {
		case "all", "success", "failed":
		default:
			return fmt.Errorf("--status must be all|success|failed, got %q", p.Status)
		}
	}

	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.ListWebhooklog(cmd.Context(), p)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(systemWebhooklogListOutput), systemWebhooklogListJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "SEND_DATE\tSTATUS\tDOCID\tFORM_CODE\tROUTE_CODE\tTITLE\tURL\tUUID")
		for _, e := range res.Data {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				e.SendDate, e.StatusCode, e.DocID, e.FormCode, e.RouteCode, e.Title1, e.URL, e.UUID,
			)
		}
		return nil
	})
}

func runSystemWebhooklogShow(cmd *cobra.Command, args []string) error {
	uuid := strings.TrimSpace(args[0])
	if uuid == "" {
		return fmt.Errorf("uuid is required")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	out, err := client.GetWebhooklog(cmd.Context(), uuid)
	if err != nil {
		return err
	}
	if systemWebhooklogShowJQ != "" {
		return runJQ(out, systemWebhooklogShowJQ)
	}
	return writeJSON(os.Stdout, out)
}
