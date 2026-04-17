package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	docSearchBody     string
	docSearchSize     int
	docSearchOffset   int
	docSearchPage     int
	docSearchOutput   string
	docSearchJQ       string
	docSearchTitle    string
	docSearchFormName string
	docSearchFormID   int
	docSearchFGID     int
	docSearchWriters  []string
	docSearchGroups   []string
	docSearchMe       bool
	docSearchSince    string
	docSearchUntil    string

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

	docDocviewFormCode  string
	docDocviewFormName  string
	docDocviewRouteCode string
	docDocviewFromDocID int
	docDocviewProxyUser string
	docDocviewOutput    string

	docDocviewUploadFormCode     string
	docDocviewUploadFormName     string
	docDocviewUploadRouteCode    string
	docDocviewUploadFromDocID    int
	docDocviewUploadProxyUser    string
	docDocviewUploadDatas        string
	docDocviewUploadFile         string
	docDocviewUploadFileName     string
	docDocviewUploadRemarks      string
	docDocviewUploadDetailNo     int
	docDocviewUploadEvidenceType int
	docDocviewUploadOutput       string

	docOpenviewProxyUser string
	docOpenviewOutput    string

	docStatusviewOutput string

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

The search condition can be specified by either a raw JSON body (--body) or
convenience filter flags. Mixing --body with filter flags is rejected.

Raw body (--body) accepts one of:
  - inline JSON string                    (e.g. --body '{"title":"経費"}')
  - a path to a JSON file                 (e.g. --body ./search.json)
  - "-" to read the body from stdin       (e.g. --body -)

Filter flags build a search body automatically:
  --title <s>              partial match on the document title (件名)
  --form-name <s>          partial match on the form name
  --form-id <n>            form ID (fid)
  --form-group-id <n>      form group ID (fgid)
  --writer <code>          writer user code (repeatable)
  --writer-group <code>    writer user-group code (repeatable)
  --me                     shorthand for --writer <current user code>;
                           looked up via XPOINT_USER or /scim/v2/{domain_code}/Me
  --since <YYYY-MM-DD>     lower bound of 新規更新日 (cr_dt)
  --until <YYYY-MM-DD>     upper bound of 新規更新日 (cr_dt)

If neither --body nor any filter flag is given, an empty object is sent
(matches all documents).`,
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

var documentDocviewCmd = &cobra.Command{
	Use:   "docview",
	Short: "Fetch the HTML view for creating a new document or a related document",
	Long: `Fetch the HTML form used to create a new document or a related document via
GET /api/v1/documents/docview.

Specify the target form with either --form-code or --form-name (form name takes
priority on the server side if both are given). --route-code is required; pass
an empty string ("") for standard forms or "--condroute" for workflow forms
with auto-select routes.

By default the HTML is saved to a file named like "docview-<fcd|formname>.html"
in the current directory. Use --output to override:
  --output FILE    save to FILE
  --output DIR/    save into DIR/ using the default filename
  --output -       write the HTML to stdout`,
	Args: cobra.NoArgs,
	RunE: runDocumentDocview,
}

var documentDocviewUploadCmd = &cobra.Command{
	Use:   "docview-upload",
	Short: "Fetch the HTML form for a new document with pre-filled data or attachment (multipart)",
	Long: `Fetch the HTML form for creating a new document or a related document with
pre-filled field values and/or a pre-attached file via POST
/multiapi/v1/documents/docview.

Specify the target form with either --form-code or --form-name. --route-code
is required. Pass --datas to supply pre-fill JSON, and --file to attach a
single file (--file-name defaults to basename of --file).`,
	Args: cobra.NoArgs,
	RunE: runDocumentDocviewUpload,
}

var documentOpenviewCmd = &cobra.Command{
	Use:   "openview <docid>",
	Short: "Fetch the HTML view of a document",
	Long: `Fetch the HTML view of a document via GET /api/v1/documents/{docid}/openview.

The HTML is the same rendering as the X-point viewer but cannot be closed by
the in-page close button (use the browser tab/window close instead).`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentOpenview,
}

var documentStatusviewCmd = &cobra.Command{
	Use:   "statusview <docid>",
	Short: "Fetch the HTML of the approval status view",
	Long:  "Fetch the HTML of the approval status view via GET /api/v1/documents/{docid}/statusview.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentStatusview,
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
	documentCmd.AddCommand(documentDocviewCmd)
	documentCmd.AddCommand(documentDocviewUploadCmd)
	documentCmd.AddCommand(documentOpenviewCmd)
	documentCmd.AddCommand(documentStatusviewCmd)
	documentCmd.AddCommand(documentOpenCmd)
	documentCmd.AddCommand(documentCommentCmd)
	documentCommentCmd.AddCommand(documentCommentAddCmd)
	documentCommentCmd.AddCommand(documentCommentGetCmd)
	documentCommentCmd.AddCommand(documentCommentEditCmd)
	documentCommentCmd.AddCommand(documentCommentDeleteCmd)

	f := documentSearchCmd.Flags()
	f.StringVar(&docSearchBody, "body", "", "search condition JSON: inline, file path, or - for stdin (cannot be combined with filter flags)")
	f.IntVar(&docSearchSize, "size", 0, "number of items per page (0 = omit, server default 50; max 1000)")
	f.IntVar(&docSearchOffset, "offset", 0, "result offset (0 = omit)")
	f.IntVar(&docSearchPage, "page", 0, "result page (0 = omit)")
	f.StringVarP(&docSearchOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	f.StringVar(&docSearchJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
	f.StringVar(&docSearchTitle, "title", "", "partial match on document title")
	f.StringVar(&docSearchFormName, "form-name", "", "partial match on form name")
	f.IntVar(&docSearchFormID, "form-id", 0, "form ID (fid); 0 = omit")
	f.IntVar(&docSearchFGID, "form-group-id", 0, "form group ID (fgid); 0 = omit")
	f.StringSliceVar(&docSearchWriters, "writer", nil, "writer user code (repeatable)")
	f.StringSliceVar(&docSearchGroups, "writer-group", nil, "writer user-group code (repeatable)")
	f.BoolVar(&docSearchMe, "me", false, "restrict to documents written by the current user (XPOINT_USER, or /scim/v2/{domain_code}/Me)")
	f.StringVar(&docSearchSince, "since", "", "lower bound of 新規更新日 (YYYY-MM-DD)")
	f.StringVar(&docSearchUntil, "until", "", "upper bound of 新規更新日 (YYYY-MM-DD)")

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

	dvf := documentDocviewCmd.Flags()
	dvf.StringVar(&docDocviewFormCode, "form-code", "", "form code (fcd); use --form-name instead or in addition (form-name wins)")
	dvf.StringVar(&docDocviewFormName, "form-name", "", "form name")
	dvf.StringVar(&docDocviewRouteCode, "route-code", "", "approval route code (required; \"\" for standard forms, \"--condroute\" for auto-select workflow routes)")
	dvf.IntVar(&docDocviewFromDocID, "from-docid", 0, "source document ID for a related document (0 = omit)")
	dvf.StringVar(&docDocviewProxyUser, "proxy-user", "", "proxy applicant user code (代理申請)")
	dvf.StringVarP(&docDocviewOutput, "output", "o", "", "output path: FILE, DIR/, or - for stdout (default: docview-<form>.html in current dir)")

	duf := documentDocviewUploadCmd.Flags()
	duf.StringVar(&docDocviewUploadFormCode, "form-code", "", "form code (fcd)")
	duf.StringVar(&docDocviewUploadFormName, "form-name", "", "form name (wins over --form-code when both are given)")
	duf.StringVar(&docDocviewUploadRouteCode, "route-code", "", "approval route code (required)")
	duf.IntVar(&docDocviewUploadFromDocID, "from-docid", 0, "source document ID for a related document (0 = omit)")
	duf.StringVar(&docDocviewUploadProxyUser, "proxy-user", "", "proxy applicant user code")
	duf.StringVar(&docDocviewUploadDatas, "datas", "", "pre-fill data JSON: inline, file path, or - for stdin")
	duf.StringVar(&docDocviewUploadFile, "file", "", "path to a file to pre-attach (at most one file)")
	duf.StringVar(&docDocviewUploadFileName, "file-name", "", "override the attachment filename (default: basename of --file)")
	duf.StringVar(&docDocviewUploadRemarks, "remarks", "", "attachment remarks (備考)")
	duf.IntVar(&docDocviewUploadDetailNo, "detail-no", 0, "detail row number for an attachment-type form (0 = omit)")
	duf.IntVar(&docDocviewUploadEvidenceType, "evidence-type", -1, "電帳法書類区分 0:その他 / 1:電子取引 (-1 = omit; default 1 on server)")
	duf.StringVarP(&docDocviewUploadOutput, "output", "o", "", "output path: FILE, DIR/, or - for stdout (default: docview-<form>.html in current dir)")

	ovf := documentOpenviewCmd.Flags()
	ovf.StringVar(&docOpenviewProxyUser, "proxy-user", "", "proxy user code (代理モードで書類を表示)")
	ovf.StringVarP(&docOpenviewOutput, "output", "o", "", "output path: FILE, DIR/, or - for stdout (default: openview-<docid>.html in current dir)")

	svf := documentStatusviewCmd.Flags()
	svf.StringVarP(&docStatusviewOutput, "output", "o", "", "output path: FILE, DIR/, or - for stdout (default: statusview-<docid>.html in current dir)")

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

	hasFilters := docSearchTitle != "" || docSearchFormName != "" || docSearchFormID != 0 ||
		docSearchFGID != 0 || len(docSearchWriters) > 0 || len(docSearchGroups) > 0 ||
		docSearchMe || docSearchSince != "" || docSearchUntil != ""
	if hasFilters {
		if len(bodyBytes) > 0 {
			return fmt.Errorf("--body cannot be combined with filter flags (--title, --form-*, --writer*, --me, --since, --until)")
		}
		meCode := ""
		if docSearchMe {
			meCode, err = resolveCurrentUserCode(cmd.Context(), client)
			if err != nil {
				return err
			}
		}
		built, err := buildSearchBodyFromFlags(meCode)
		if err != nil {
			return err
		}
		bodyBytes = built
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

func runDocumentDocview(cmd *cobra.Command, args []string) error {
	if docDocviewFormCode == "" && docDocviewFormName == "" {
		return fmt.Errorf("either --form-code or --form-name is required")
	}
	if !cmd.Flags().Changed("route-code") {
		return fmt.Errorf("--route-code is required (use an empty string \"\" for standard forms)")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	params := xpoint.DocviewParams{
		FormCode:  docDocviewFormCode,
		FormName:  docDocviewFormName,
		RouteCode: docDocviewRouteCode,
		ProxyUser: docDocviewProxyUser,
	}
	if docDocviewFromDocID != 0 {
		v := docDocviewFromDocID
		params.FromDocID = &v
	}

	data, err := client.GetDocview(cmd.Context(), params)
	if err != nil {
		return err
	}
	defaultName := defaultDocviewFilename("docview", params.FormCode, params.FormName, 0)
	return writeHTMLOutput(docDocviewOutput, defaultName, data)
}

func runDocumentDocviewUpload(cmd *cobra.Command, args []string) error {
	if docDocviewUploadFormCode == "" && docDocviewUploadFormName == "" {
		return fmt.Errorf("either --form-code or --form-name is required")
	}
	if !cmd.Flags().Changed("route-code") {
		return fmt.Errorf("--route-code is required (use an empty string \"\" for standard forms)")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	params := xpoint.DocviewMultipartParams{
		FormCode:  docDocviewUploadFormCode,
		FormName:  docDocviewUploadFormName,
		RouteCode: docDocviewUploadRouteCode,
		ProxyUser: docDocviewUploadProxyUser,
	}
	if docDocviewUploadFromDocID != 0 {
		v := docDocviewUploadFromDocID
		params.FromDocID = &v
	}
	if docDocviewUploadDatas != "" {
		datas, err := loadStringInput(docDocviewUploadDatas)
		if err != nil {
			return fmt.Errorf("load --datas: %w", err)
		}
		params.Datas = datas
	}
	if docDocviewUploadFile != "" {
		content, err := os.ReadFile(docDocviewUploadFile)
		if err != nil {
			return fmt.Errorf("read --file: %w", err)
		}
		name := docDocviewUploadFileName
		if name == "" {
			name = filepath.Base(docDocviewUploadFile)
		}
		f := &xpoint.DocviewMultipartFile{
			Name:    name,
			Remarks: docDocviewUploadRemarks,
			Content: content,
		}
		if cmd.Flags().Changed("detail-no") {
			v := docDocviewUploadDetailNo
			f.DetailNo = &v
		}
		if docDocviewUploadEvidenceType >= 0 {
			v := docDocviewUploadEvidenceType
			f.EvidenceType = &v
		}
		params.File = f
	}

	data, err := client.PostDocviewMultipart(cmd.Context(), params)
	if err != nil {
		return err
	}
	defaultName := defaultDocviewFilename("docview", params.FormCode, params.FormName, 0)
	return writeHTMLOutput(docDocviewUploadOutput, defaultName, data)
}

func runDocumentOpenview(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	data, err := client.GetDocumentOpenview(cmd.Context(), docID, docOpenviewProxyUser)
	if err != nil {
		return err
	}
	defaultName := defaultDocviewFilename("openview", "", "", docID)
	return writeHTMLOutput(docOpenviewOutput, defaultName, data)
}

func runDocumentStatusview(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	data, err := client.GetDocumentStatusview(cmd.Context(), docID)
	if err != nil {
		return err
	}
	defaultName := defaultDocviewFilename("statusview", "", "", docID)
	return writeHTMLOutput(docStatusviewOutput, defaultName, data)
}

// defaultDocviewFilename builds a reasonable default output filename for
// docview/openview/statusview HTML responses.
func defaultDocviewFilename(prefix, formCode, formName string, docID int) string {
	switch {
	case docID > 0:
		return fmt.Sprintf("%s-%d.html", prefix, docID)
	case formCode != "":
		return fmt.Sprintf("%s-%s.html", prefix, sanitizeFilename(formCode))
	case formName != "":
		return fmt.Sprintf("%s-%s.html", prefix, sanitizeFilename(formName))
	default:
		return prefix + ".html"
	}
}

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\x00':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// writeHTMLOutput writes HTML bytes to stdout (out == "-"), to a path given
// by out (FILE or DIR/), or to defaultName in the current directory when out
// is empty.
func writeHTMLOutput(out, defaultName string, data []byte) error {
	if out == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	var dst string
	switch {
	case out == "":
		dst = defaultName
	case strings.HasSuffix(out, string(os.PathSeparator)) || strings.HasSuffix(out, "/"):
		dst = filepath.Join(out, defaultName)
	default:
		if info, err := os.Stat(out); err == nil && info.IsDir() {
			dst = filepath.Join(out, defaultName)
		} else {
			dst = out
		}
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("write html: %w", err)
	}
	fmt.Fprintf(os.Stderr, "saved: %s (%d bytes)\n", dst, len(data))
	return nil
}

// loadStringInput accepts an inline string, a file path, or "-" for stdin
// and returns the resulting string content.
func loadStringInput(s string) (string, error) {
	if s == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(b), nil
	}
	if info, err := os.Stat(s); err == nil && !info.IsDir() {
		b, err := os.ReadFile(s)
		if err != nil {
			return "", fmt.Errorf("read file: %w", err)
		}
		return string(b), nil
	}
	return s, nil
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

type writerListEntry struct {
	Type string `json:"type"`
	Code string `json:"code"`
}

// resolveCurrentUserCode returns the authenticated user's X-point user code
// for --me. It prefers XPOINT_USER / --xpoint-user; if neither is set, it
// falls back to GET /scim/v2/{domain_code}/Me and reads the atled SCIM
// extension's userCode (not userName — that's the login name, while the
// writer_list API expects the numeric user code).
func resolveCurrentUserCode(ctx context.Context, client *xpoint.Client) (string, error) {
	if u := pick(flagUser, "XPOINT_USER"); u != "" {
		return u, nil
	}
	domain := resolveDomainCode()
	if domain == "" {
		return "", fmt.Errorf("--me requires the current user code: set --xpoint-user / XPOINT_USER, or provide a domain code (--xpoint-domain-code / XPOINT_DOMAIN_CODE / stored OAuth login) to look it up via /scim/v2/{domain_code}/Me")
	}
	info, err := client.GetSelfInfo(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("resolve --me via /scim/v2/%s/Me: %w", domain, err)
	}
	if info.AtledExt.UserCode == "" {
		return "", fmt.Errorf("resolve --me: userCode is empty in /scim/v2/%s/Me response (atled SCIM extension missing)", domain)
	}
	return info.AtledExt.UserCode, nil
}

// buildSearchBodyFromFlags converts --title / --form-* / --writer* / --me /
// --since / --until into a JSON request body for POST /api/v1/search/documents.
//
// meCode is the resolved user code to use for --me (empty if --me was not set
// or resolution is not needed).
func buildSearchBodyFromFlags(meCode string) (json.RawMessage, error) {
	body := map[string]any{}

	if docSearchTitle != "" {
		body["title"] = docSearchTitle
	}
	if docSearchFormName != "" {
		body["form_name"] = docSearchFormName
	}
	if docSearchFormID != 0 {
		body["fid"] = docSearchFormID
	}
	if docSearchFGID != 0 {
		body["fgid"] = docSearchFGID
	}

	var writers []writerListEntry
	for _, code := range docSearchWriters {
		if code = strings.TrimSpace(code); code != "" {
			writers = append(writers, writerListEntry{Type: "user", Code: code})
		}
	}
	for _, code := range docSearchGroups {
		if code = strings.TrimSpace(code); code != "" {
			writers = append(writers, writerListEntry{Type: "group", Code: code})
		}
	}
	if meCode != "" {
		writers = append(writers, writerListEntry{Type: "user", Code: meCode})
	}
	if len(writers) > 0 {
		body["writer_list"] = writers
	}

	if docSearchSince != "" || docSearchUntil != "" {
		body["date_type"] = "cr_dt"
		body["dt_cond_type"] = "1"
		if docSearchSince != "" {
			t, err := parseSearchDate(docSearchSince)
			if err != nil {
				return nil, fmt.Errorf("--since: %w", err)
			}
			body["lower_year"] = t.Year()
			body["lower_month"] = int(t.Month())
			body["lower_day"] = t.Day()
		}
		if docSearchUntil != "" {
			t, err := parseSearchDate(docSearchUntil)
			if err != nil {
				return nil, fmt.Errorf("--until: %w", err)
			}
			body["upper_year"] = t.Year()
			body["upper_month"] = int(t.Month())
			body["upper_day"] = t.Day()
		}
	}

	return json.Marshal(body)
}

func parseSearchDate(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", s)
	}
	return t, nil
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
