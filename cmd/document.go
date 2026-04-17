package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	docSearchBody   string
	docSearchSize   int
	docSearchOffset int
	docSearchPage   int
	docSearchOutput string
	docSearchJQ     string

	docCreateBody   string
	docCreateOutput string
	docCreateJQ     string
)

var documentCmd = &cobra.Command{
	Use:   "document",
	Short: "Manage X-point documents",
}

var documentSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search documents",
	Long: `Search documents via POST /api/v1/search/documents.

The search condition JSON is provided with --body, which accepts one of:
  - inline JSON string                    (e.g. --body '{"title":"経費"}')
  - a path to a JSON file                 (e.g. --body ./search.json)
  - "-" to read the body from stdin       (e.g. --body -)

If --body is omitted, an empty object is sent (matches all documents).`,
	RunE: runDocumentSearch,
}

var documentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a document",
	Long: `Create a document via POST /api/v1/documents.

The request body is provided with --body, which accepts one of:
  - inline JSON string                    (e.g. --body '{"route_code":"r1",...}')
  - a path to a JSON file                 (e.g. --body ./doc.json)
  - "-" to read the body from stdin       (e.g. --body -)

The body must contain route_code, datas, and a form identifier
(form_code or form_name). See "xp schema document.create" for shape.`,
	RunE: runDocumentCreate,
}

func init() {
	rootCmd.AddCommand(documentCmd)
	documentCmd.AddCommand(documentSearchCmd)
	documentCmd.AddCommand(documentCreateCmd)

	f := documentSearchCmd.Flags()
	f.StringVar(&docSearchBody, "body", "", "search condition JSON: inline, file path, or - for stdin")
	f.IntVar(&docSearchSize, "size", 0, "number of items per page (0 = omit, server default 50; max 1000)")
	f.IntVar(&docSearchOffset, "offset", 0, "result offset (0 = omit)")
	f.IntVar(&docSearchPage, "page", 0, "result page (0 = omit)")
	f.StringVarP(&docSearchOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	f.StringVar(&docSearchJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	cf := documentCreateCmd.Flags()
	cf.StringVar(&docCreateBody, "body", "", "request body JSON: inline, file path, or - for stdin (required)")
	cf.StringVarP(&docCreateOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	cf.StringVar(&docCreateJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runDocumentSearch(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	bodyBytes, err := loadSearchBody(docSearchBody)
	if err != nil {
		return err
	}

	params := xpoint.SearchDocumentsParams{}
	if docSearchSize != 0 {
		v := docSearchSize
		params.Size = &v
	}
	if cmd.Flags().Changed("offset") {
		v := docSearchOffset
		params.Offset = &v
	}
	if docSearchPage != 0 {
		v := docSearchPage
		params.Page = &v
	}

	res, err := client.SearchDocuments(cmd.Context(), params, bodyBytes)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(docSearchOutput), docSearchJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintf(os.Stdout, "total: %d\n", res.TotalCount)
		fmt.Fprintln(w, "DOCID\tFORM_NAME\tWRITER\tWRITE_DATETIME\tSTEP\tSTAT\tTITLE1")
		for _, it := range res.Items {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%s\n",
				it.DocID, it.Form.Name, it.Writer, it.WriteDatetime, it.Step, it.Stat, it.Title1,
			)
		}
		return nil
	})
}

func runDocumentCreate(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags()
	if err != nil {
		return err
	}

	bodyBytes, err := loadSearchBody(docCreateBody)
	if err != nil {
		return err
	}
	if len(bodyBytes) == 0 {
		return fmt.Errorf("--body is required for document create")
	}

	res, err := client.CreateDocument(cmd.Context(), bodyBytes)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(docCreateOutput), docCreateJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "DOCID\tMESSAGE_TYPE\tMESSAGE")
		fmt.Fprintf(w, "%d\t%d\t%s\n", res.DocID, res.MessageType, res.Message)
		return nil
	})
}

// loadSearchBody resolves --body into JSON bytes.
func loadSearchBody(spec string) (json.RawMessage, error) {
	if spec == "" {
		return nil, nil
	}
	var data []byte
	switch {
	case spec == "-":
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read --body from stdin: %w", err)
		}
		data = b
	case strings.HasPrefix(strings.TrimSpace(spec), "{") || strings.HasPrefix(strings.TrimSpace(spec), "["):
		data = []byte(spec)
	default:
		b, err := os.ReadFile(spec)
		if err != nil {
			return nil, fmt.Errorf("read --body file: %w", err)
		}
		data = b
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("--body is not valid JSON")
	}
	return json.RawMessage(data), nil
}
