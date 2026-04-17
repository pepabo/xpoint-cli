package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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

	docGetOutput string
	docGetJQ     string

	docEditBody   string
	docEditOutput string
	docEditJQ     string

	docDeleteYes    bool
	docDeleteOutput string
	docDeleteJQ     string

	docDownloadOutput string

	docStatusHistory bool
	docStatusJQ      string

	docOpenNoBrowser bool

	docCommentAddContent    string
	docCommentAddAttention  bool
	docCommentAddOutput     string
	docCommentAddJQ         string
	docCommentGetOutput     string
	docCommentGetJQ         string
	docCommentEditContent   string
	docCommentEditAttention string
	docCommentEditOutput    string
	docCommentEditJQ        string
	docCommentDeleteYes     bool
	docCommentDeleteOutput  string
	docCommentDeleteJQ      string
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

var documentGetCmd = &cobra.Command{
	Use:   "get <docid>",
	Short: "Get a document",
	Long: `Retrieve a single document via GET /api/v1/documents/{docid}.

The response varies by form and is returned as JSON; use --jq to extract
specific fields.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentGet,
}

var documentEditCmd = &cobra.Command{
	Use:   "edit <docid>",
	Short: "Update a document",
	Long: `Update a document via PATCH /api/v1/documents/{docid}.

The request body is provided with --body, which accepts one of:
  - inline JSON string                    (e.g. --body '{"route_code":"r1","datas":[...]}')
  - a path to a JSON file                 (e.g. --body ./patch.json)
  - "-" to read the body from stdin       (e.g. --body -)

The body may contain wf_type, datas, route_code, wf_comment, reason, etc.
When performing a workflow-only operation, omit datas.
See "xp schema document.update" for shape.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentEdit,
}

var documentDeleteCmd = &cobra.Command{
	Use:   "delete <docid>",
	Short: "Delete a document",
	Long: `Delete a document via DELETE /api/v1/documents/{docid}.

By default the command prompts for confirmation. Pass --yes to skip it.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentDelete,
}

var documentStatusCmd = &cobra.Command{
	Use:   "status <docid>",
	Short: "Get document approval status",
	Long: `Retrieve the approval status of a document via GET /api/v1/documents/{docid}/status.

The response is returned as JSON and contains the current status, step,
writer, last approver, and the approval flow. Pass --history to include
approval histories for all past versions.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentStatus,
}

var documentOpenCmd = &cobra.Command{
	Use:   "open <docid>",
	Short: "Open a document in the default web browser",
	Long: `Open the document view page in the default web browser.

The URL is built from the configured subdomain (no API request is made).
Pass --no-browser (or -n) to print the URL without launching the browser.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentOpen,
}

var documentCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage document comments",
}

var documentCommentAddCmd = &cobra.Command{
	Use:   "add <docid>",
	Short: "Add a comment to a document",
	Long: `Add a comment to a document via POST /api/v1/documents/{docid}/comments.

The comment body is provided with --content (required). Pass --attention
to mark the comment as important.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentCommentAdd,
}

var documentCommentGetCmd = &cobra.Command{
	Use:   "get <docid>",
	Short: "Get comments on a document",
	Long:  `List comments on a document via GET /api/v1/documents/{docid}/comments.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentCommentGet,
}

var documentCommentEditCmd = &cobra.Command{
	Use:   "edit <docid> <seq>",
	Short: "Update a comment",
	Long: `Update a comment via PATCH /api/v1/documents/{docid}/comments/{seq}.

At least one of --content / --attention must be provided. --attention accepts
"0" (normal) or "1" (important); omit to leave unchanged.`,
	Args: cobra.ExactArgs(2),
	RunE: runDocumentCommentEdit,
}

var documentCommentDeleteCmd = &cobra.Command{
	Use:   "delete <docid> <seq>",
	Short: "Delete a comment",
	Long: `Delete a comment via DELETE /api/v1/documents/{docid}/comments/{seq}.

By default the command prompts for confirmation. Pass --yes to skip it.`,
	Args: cobra.ExactArgs(2),
	RunE: runDocumentCommentDelete,
}

var documentDownloadCmd = &cobra.Command{
	Use:   "download <docid>",
	Short: "Download a document as PDF",
	Long: `Download a document as PDF via GET /api/v1/documents/{docid}/pdf.

By default the PDF is saved to the current directory using the filename
provided by the server (Content-Disposition). Use --output to override:
  --output FILE    save to FILE
  --output DIR/    save into DIR/ using the server-provided filename
  --output -       write the PDF to stdout`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentDownload,
}

func init() {
	rootCmd.AddCommand(documentCmd)
	documentCmd.AddCommand(documentSearchCmd)
	documentCmd.AddCommand(documentCreateCmd)
	documentCmd.AddCommand(documentGetCmd)
	documentCmd.AddCommand(documentEditCmd)
	documentCmd.AddCommand(documentDeleteCmd)
	documentCmd.AddCommand(documentStatusCmd)
	documentCmd.AddCommand(documentDownloadCmd)
	documentCmd.AddCommand(documentOpenCmd)
	documentCmd.AddCommand(documentCommentCmd)
	documentCommentCmd.AddCommand(documentCommentAddCmd)
	documentCommentCmd.AddCommand(documentCommentGetCmd)
	documentCommentCmd.AddCommand(documentCommentEditCmd)
	documentCommentCmd.AddCommand(documentCommentDeleteCmd)

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

	gf := documentGetCmd.Flags()
	gf.StringVarP(&docGetOutput, "output", "o", "", "output format: json (default)")
	gf.StringVar(&docGetJQ, "jq", "", "apply a gojq filter to the JSON response")

	ef := documentEditCmd.Flags()
	ef.StringVar(&docEditBody, "body", "", "request body JSON: inline, file path, or - for stdin (required)")
	ef.StringVarP(&docEditOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	ef.StringVar(&docEditJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	df := documentDeleteCmd.Flags()
	df.BoolVarP(&docDeleteYes, "yes", "y", false, "skip the interactive confirmation prompt")
	df.StringVarP(&docDeleteOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	df.StringVar(&docDeleteJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	sf := documentStatusCmd.Flags()
	sf.BoolVar(&docStatusHistory, "history", false, "include approval histories for all versions")
	sf.StringVar(&docStatusJQ, "jq", "", "apply a gojq filter to the JSON response")

	dlf := documentDownloadCmd.Flags()
	dlf.StringVarP(&docDownloadOutput, "output", "o", "", "output path: FILE, DIR/, or - for stdout (default: server-provided filename in current directory)")

	of := documentOpenCmd.Flags()
	of.BoolVarP(&docOpenNoBrowser, "no-browser", "n", false, "print the URL without launching the browser")

	caf := documentCommentAddCmd.Flags()
	caf.StringVar(&docCommentAddContent, "content", "", "comment content (required)")
	caf.BoolVar(&docCommentAddAttention, "attention", false, "mark as important comment (attentionflg=1)")
	caf.StringVarP(&docCommentAddOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	caf.StringVar(&docCommentAddJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	cgf := documentCommentGetCmd.Flags()
	cgf.StringVarP(&docCommentGetOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	cgf.StringVar(&docCommentGetJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	cef := documentCommentEditCmd.Flags()
	cef.StringVar(&docCommentEditContent, "content", "", "new comment content (omit to leave unchanged)")
	cef.StringVar(&docCommentEditAttention, "attention", "", "new attention flag: 0 (normal) | 1 (important); omit to leave unchanged")
	cef.StringVarP(&docCommentEditOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	cef.StringVar(&docCommentEditJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	cdf := documentCommentDeleteCmd.Flags()
	cdf.BoolVarP(&docCommentDeleteYes, "yes", "y", false, "skip the interactive confirmation prompt")
	cdf.StringVarP(&docCommentDeleteOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	cdf.StringVar(&docCommentDeleteJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
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

type documentCreateOutputView struct {
	DocID       int    `json:"docid"`
	MessageType int    `json:"message_type"`
	Message     string `json:"message"`
	URL         string `json:"url,omitempty"`
}

func runDocumentCreate(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
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

	view := documentCreateOutputView{
		DocID:       res.DocID,
		MessageType: res.MessageType,
		Message:     res.Message,
		URL:         client.DocumentURL(res.DocID),
	}

	return render(&view, resolveOutputFormat(docCreateOutput), docCreateJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "DOCID\tMESSAGE_TYPE\tMESSAGE\tURL")
		fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", view.DocID, view.MessageType, view.Message, view.URL)
		return nil
	})
}

func runDocumentGet(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	raw, err := client.GetDocument(cmd.Context(), docID)
	if err != nil {
		return err
	}

	if docGetJQ != "" {
		return runJQ(raw, docGetJQ)
	}
	// The response is a complex document; always emit JSON.
	return writeJSON(os.Stdout, raw)
}

func runDocumentEdit(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	bodyBytes, err := loadSearchBody(docEditBody)
	if err != nil {
		return err
	}
	if len(bodyBytes) == 0 {
		return fmt.Errorf("--body is required for document edit")
	}

	res, err := client.UpdateDocument(cmd.Context(), docID, bodyBytes)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(docEditOutput), docEditJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "DOCID\tMESSAGE_TYPE\tMESSAGE")
		fmt.Fprintf(w, "%d\t%d\t%s\n", res.DocID, res.MessageType, res.Message)
		return nil
	})
}

func runDocumentDelete(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	if !docDeleteYes {
		if !confirmDelete(docID) {
			return fmt.Errorf("aborted")
		}
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	res, err := client.DeleteDocument(cmd.Context(), docID)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(docDeleteOutput), docDeleteJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "MESSAGE_TYPE\tMESSAGE")
		fmt.Fprintf(w, "%d\t%s\n", res.MessageType, res.Message)
		return nil
	})
}

func runDocumentStatus(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	raw, err := client.GetDocumentStatus(cmd.Context(), docID, docStatusHistory)
	if err != nil {
		return err
	}
	if docStatusJQ != "" {
		return runJQ(raw, docStatusJQ)
	}
	return writeJSON(os.Stdout, raw)
}

func runDocumentDownload(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	filename, data, err := client.DownloadPDF(cmd.Context(), docID)
	if err != nil {
		return err
	}

	out := docDownloadOutput
	if out == "-" {
		_, werr := os.Stdout.Write(data)
		return werr
	}

	dst := resolveDownloadPath(out, filename, docID)
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("write pdf: %w", err)
	}
	fmt.Fprintf(os.Stderr, "saved: %s (%d bytes)\n", dst, len(data))
	return nil
}

func runDocumentOpen(_ *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	sub, err := resolveSubdomain()
	if err != nil {
		return err
	}
	url := xpoint.NewClient(sub, xpoint.Auth{}).DocumentURL(docID)

	if docOpenNoBrowser {
		fmt.Fprintln(os.Stdout, url)
		return nil
	}
	fmt.Fprintf(os.Stdout, "Opening %s\n", url)
	if err := openBrowser(url); err != nil {
		return fmt.Errorf("open browser: %w (run with --no-browser to print the URL)", err)
	}
	return nil
}

func runDocumentCommentAdd(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	if docCommentAddContent == "" {
		return fmt.Errorf("--content is required")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	req := xpoint.AddCommentRequest{Content: docCommentAddContent}
	if docCommentAddAttention {
		req.AttentionFlg = 1
	}
	res, err := client.AddComment(cmd.Context(), docID, req)
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(docCommentAddOutput), docCommentAddJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "DOCID\tSEQ\tMESSAGE_TYPE\tMESSAGE")
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\n", res.DocID, res.Seq, res.MessageType, res.Message)
		return nil
	})
}

func runDocumentCommentGet(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.GetComments(cmd.Context(), docID)
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(docCommentGetOutput), docCommentGetJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "SEQ\tATTENTION\tWRITER\tWRITE_DATE\tCONTENT")
		for _, cm := range res.CommentList {
			attention := "-"
			if cm.AttentionFlg {
				attention = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", cm.SeqNo, attention, cm.WriterName, cm.WriteDate, cm.Content)
		}
		return nil
	})
}

func runDocumentCommentEdit(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	seq, err := parseSeq(args[1])
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("content") && !cmd.Flags().Changed("attention") {
		return fmt.Errorf("at least one of --content / --attention is required")
	}
	req := xpoint.UpdateCommentRequest{}
	if cmd.Flags().Changed("content") {
		v := docCommentEditContent
		req.Content = &v
	}
	if cmd.Flags().Changed("attention") {
		switch docCommentEditAttention {
		case "0":
			v := 0
			req.AttentionFlg = &v
		case "1":
			v := 1
			req.AttentionFlg = &v
		default:
			return fmt.Errorf("--attention must be 0 or 1, got %q", docCommentEditAttention)
		}
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.UpdateComment(cmd.Context(), docID, seq, req)
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(docCommentEditOutput), docCommentEditJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "DOCID\tSEQ\tMESSAGE_TYPE\tMESSAGE")
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\n", res.DocID, res.Seq, res.MessageType, res.Message)
		return nil
	})
}

func runDocumentCommentDelete(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	seq, err := parseSeq(args[1])
	if err != nil {
		return err
	}
	if !docCommentDeleteYes && !confirmDeleteComment(docID, seq) {
		return fmt.Errorf("aborted")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.DeleteComment(cmd.Context(), docID, seq)
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(docCommentDeleteOutput), docCommentDeleteJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "DOCID\tSEQ\tMESSAGE_TYPE\tMESSAGE")
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\n", res.DocID, res.Seq, res.MessageType, res.Message)
		return nil
	})
}

func parseSeq(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid seq %q: must be a positive integer", s)
	}
	return n, nil
}

func confirmDeleteComment(docID, seq int) bool {
	fmt.Fprintf(os.Stderr, "Really delete comment seq=%d on document %d? [y/N]: ", seq, docID)
	var ans string
	_, _ = fmt.Fscanln(os.Stdin, &ans)
	switch strings.ToLower(strings.TrimSpace(ans)) {
	case "y", "yes":
		return true
	}
	return false
}

// resolveDownloadPath decides the on-disk path for a downloaded PDF.
//
// When out is empty, the server-provided filename is used in the current
// directory (falling back to "<docid>.pdf"). When out ends with a path
// separator or names an existing directory, the server-provided filename is
// placed inside it. Otherwise out is used verbatim as the file path. The
// server-provided name is base-name-cleaned to avoid path traversal.
func resolveDownloadPath(out, serverName string, docID int) string {
	name := filepath.Base(filepath.Clean(serverName))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = fmt.Sprintf("%d.pdf", docID)
	}
	if out == "" {
		return name
	}
	if strings.HasSuffix(out, string(os.PathSeparator)) || strings.HasSuffix(out, "/") {
		return filepath.Join(out, name)
	}
	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return filepath.Join(out, name)
	}
	return out
}

func parseDocID(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid docid %q: must be a positive integer", s)
	}
	return n, nil
}

// confirmDelete prompts on stderr and reads a yes/no answer from stdin.
// Anything other than "y" / "yes" (case-insensitive) aborts.
func confirmDelete(docID int) bool {
	fmt.Fprintf(os.Stderr, "Really delete document %d? [y/N]: ", docID)
	var ans string
	_, _ = fmt.Fscanln(os.Stdin, &ans)
	switch strings.ToLower(strings.TrimSpace(ans)) {
	case "y", "yes":
		return true
	}
	return false
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
