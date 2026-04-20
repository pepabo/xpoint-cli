package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

const queryExecPageSize = 1000

var (
	queryListOutput string
	queryListJQ     string

	queryExecNoRun bool
	queryExecJQ    string

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
both the definition and exec_result. List-type queries are paged through
automatically (rows=1000) until exec_result.data is exhausted. Pass
--no-run to fetch the definition only.`,
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
		w := newTable(os.Stdout,
			"GROUP_ID", "GROUP_NAME", "QUERY_ID", "QUERY_CODE", "QUERY_NAME", "QUERY_TYPE", "FORM")
		for _, g := range res.QueryGroups {
			if len(g.Queries) == 0 {
				w.AddRow(g.QueryGroupID, g.QueryGroupName, "-", "-", "-", "-", "-")
				continue
			}
			for _, q := range g.Queries {
				w.AddRow(g.QueryGroupID, g.QueryGroupName, q.QueryID, q.QueryCode, q.QueryName, q.QueryType, formatQueryForm(q))
			}
		}
		w.Print()
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

	var raw json.RawMessage
	if queryExecNoRun {
		raw, err = client.GetQuery(cmd.Context(), queryCode, xpoint.GetQueryParams{ExecFlag: false})
	} else {
		raw, err = fetchAllQueryExec(cmd.Context(), client, queryCode)
	}
	if err != nil {
		return err
	}
	if queryExecJQ != "" {
		return runJQ(raw, queryExecJQ)
	}
	return writeJSON(os.Stdout, raw)
}

// fetchAllQueryExec runs a query with exec_flg=true and pages through
// exec_result.data (rows=1000) until the server reports an empty or missing
// data array. For non-list queries whose exec_result does not contain a data
// array, the first response is returned as-is.
func fetchAllQueryExec(ctx context.Context, client *xpoint.Client, queryCode string) (json.RawMessage, error) {
	rows := queryExecPageSize
	offset := 0
	p := xpoint.GetQueryParams{ExecFlag: true, Rows: &rows, Offset: &offset}

	var (
		firstEnv map[string]json.RawMessage
		allData  []json.RawMessage
	)

	for {
		p.Offset = &offset
		raw, err := client.GetQuery(ctx, queryCode, p)
		if err != nil {
			return nil, err
		}
		var env map[string]json.RawMessage
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, fmt.Errorf("parse query response: %w", err)
		}
		if offset == 0 {
			firstEnv = env
		}

		execRaw, ok := env["exec_result"]
		if !ok {
			break
		}
		var execResult map[string]json.RawMessage
		if err := json.Unmarshal(execRaw, &execResult); err != nil {
			break
		}
		dataRaw, ok := execResult["data"]
		if !ok {
			break
		}
		var data []json.RawMessage
		if err := json.Unmarshal(dataRaw, &data); err != nil {
			break
		}
		if len(data) == 0 {
			break
		}
		allData = append(allData, data...)
		if len(data) < queryExecPageSize {
			break
		}
		offset += queryExecPageSize
	}

	if firstEnv == nil {
		return nil, fmt.Errorf("empty response")
	}
	if len(allData) == 0 {
		return json.Marshal(firstEnv)
	}

	var execResult map[string]json.RawMessage
	if raw, ok := firstEnv["exec_result"]; ok {
		_ = json.Unmarshal(raw, &execResult)
	}
	if execResult == nil {
		execResult = map[string]json.RawMessage{}
	}
	encoded, err := json.Marshal(allData)
	if err != nil {
		return nil, err
	}
	execResult["data"] = encoded
	mergedExec, err := json.Marshal(execResult)
	if err != nil {
		return nil, err
	}
	firstEnv["exec_result"] = mergedExec
	return json.Marshal(firstEnv)
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
