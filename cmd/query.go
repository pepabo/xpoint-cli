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
	queryListOutput string
	queryListJQ     string

	queryExecNoRun  bool
	queryExecRows   int
	queryExecOffset int
	queryExecJQ     string

	queryGraphFormat string
	queryGraphOutput string
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Manage X-point queries",
}

var queryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available queries",
	Long:  "List available queries via GET /api/v1/query/.",
	RunE:  runQueryList,
}

var queryExecCmd = &cobra.Command{
	Use:   "exec <query_code>",
	Short: "Execute a query and show its result",
	Long: `Fetch a query and its execution result via GET /api/v1/query/{query_code}.

By default the query is executed (exec_flg=true) and the response contains
both the definition and exec_result. Pass --no-run to fetch the definition
only. --rows (default 500) and --offset control pagination for list queries.`,
	Args: cobra.ExactArgs(1),
	RunE: runQueryExec,
}

var queryGraphCmd = &cobra.Command{
	Use:   "graph <query_code>",
	Short: "Download the graph image of a query",
	Long: `Download the graph image configured on a query via
GET /api/v1/query/graph/{query_code}.

The format defaults to PNG; pass --format jpeg for JPEG. Output destination:
  --output FILE    save to FILE
  --output DIR/    save into DIR/ using the server-provided filename
  --output -       write the image bytes to stdout
  (default)        use the server-provided filename in the current directory`,
	Args: cobra.ExactArgs(1),
	RunE: runQueryGraph,
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.AddCommand(queryListCmd)
	queryCmd.AddCommand(queryExecCmd)
	queryCmd.AddCommand(queryGraphCmd)

	lf := queryListCmd.Flags()
	lf.StringVarP(&queryListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	lf.StringVar(&queryListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	ef := queryExecCmd.Flags()
	ef.BoolVar(&queryExecNoRun, "no-run", false, "do not execute the query; return the definition only (exec_flg=false)")
	ef.IntVar(&queryExecRows, "rows", 0, "max rows returned by list queries (0 = omit; server default 500; range 1-10000)")
	ef.IntVar(&queryExecOffset, "offset", 0, "offset for list queries (0 = omit)")
	ef.StringVar(&queryExecJQ, "jq", "", "apply a gojq filter to the JSON response")

	gf := queryGraphCmd.Flags()
	gf.StringVar(&queryGraphFormat, "format", "", "image format: png | jpeg (default: png)")
	gf.StringVarP(&queryGraphOutput, "output", "o", "", "output path: FILE, DIR/, or - for stdout (default: server-provided filename in current directory)")
}

func runQueryList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.ListAvailableQueries(cmd.Context())
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(queryListOutput), queryListJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "GROUP_ID\tGROUP_NAME\tQUERY_ID\tQUERY_CODE\tQUERY_NAME\tQUERY_TYPE\tFORM")
		for _, g := range res.QueryGroups {
			if len(g.Queries) == 0 {
				fmt.Fprintf(w, "%d\t%s\t-\t-\t-\t-\t-\n", g.QueryGroupID, g.QueryGroupName)
				continue
			}
			for _, q := range g.Queries {
				fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\t%s\n",
					g.QueryGroupID, g.QueryGroupName, q.QueryID, q.QueryCode, q.QueryName, q.QueryType, formatQueryForm(q),
				)
			}
		}
		return nil
	})
}

// formatQueryForm renders the form column for `query list`: single form shows
// "<name> (fid)"; multi-form shows form count and a "+" suffix.
func formatQueryForm(q xpoint.Query) string {
	if q.FormCount > 1 {
		return fmt.Sprintf("%d forms", q.FormCount)
	}
	if q.FormName == "" {
		return "-"
	}
	return fmt.Sprintf("%s (%d)", q.FormName, q.FID)
}

func runQueryExec(cmd *cobra.Command, args []string) error {
	queryCode := strings.TrimSpace(args[0])
	if queryCode == "" {
		return fmt.Errorf("query_code is required")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	p := xpoint.GetQueryParams{ExecFlag: !queryExecNoRun}
	if cmd.Flags().Changed("rows") {
		v := queryExecRows
		p.Rows = &v
	}
	if cmd.Flags().Changed("offset") {
		v := queryExecOffset
		p.Offset = &v
	}

	raw, err := client.GetQuery(cmd.Context(), queryCode, p)
	if err != nil {
		return err
	}
	if queryExecJQ != "" {
		return runJQ(raw, queryExecJQ)
	}
	return writeJSON(os.Stdout, raw)
}

func runQueryGraph(cmd *cobra.Command, args []string) error {
	queryCode := strings.TrimSpace(args[0])
	if queryCode == "" {
		return fmt.Errorf("query_code is required")
	}
	format := strings.ToLower(strings.TrimSpace(queryGraphFormat))
	switch format {
	case "", "png", "jpeg":
	default:
		return fmt.Errorf("unknown --format %q (must be png or jpeg)", queryGraphFormat)
	}

	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	filename, data, err := client.GetQueryGraph(cmd.Context(), queryCode, format)
	if err != nil {
		return err
	}

	if queryGraphOutput == "-" {
		_, werr := os.Stdout.Write(data)
		return werr
	}
	if filename == "" {
		ext := "png"
		if format == "jpeg" {
			ext = "jpg"
		}
		filename = fmt.Sprintf("%s.%s", queryCode, ext)
	}
	dst := resolveDownloadPath(queryGraphOutput, filename, 0)
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("write image: %w", err)
	}
	fmt.Fprintf(os.Stderr, "saved: %s (%d bytes)\n", dst, len(data))
	return nil
}
