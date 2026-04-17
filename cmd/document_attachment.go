package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	docAttachAddUser         string
	docAttachAddFile         string
	docAttachAddName         string
	docAttachAddRemarks      string
	docAttachAddOverwrite    string
	docAttachAddReason       string
	docAttachAddDetailNo     int
	docAttachAddEvidenceType int
	docAttachAddOutput       string
	docAttachAddJQ           string

	docAttachListOutput string
	docAttachListJQ     string

	docAttachGetOutput string

	docAttachUpdateUser         string
	docAttachUpdateFile         string
	docAttachUpdateName         string
	docAttachUpdateRemarks      string
	docAttachUpdateReason       string
	docAttachUpdateDetailNo     int
	docAttachUpdateEvidenceType int
	docAttachUpdateOutput       string
	docAttachUpdateJQ           string

	docAttachDeleteUser   string
	docAttachDeleteReason string
	docAttachDeleteYes    bool
	docAttachDeleteOutput string
	docAttachDeleteJQ     string
)

var documentAttachmentCmd = &cobra.Command{
	Use:     "attachment",
	Aliases: []string{"attach"},
	Short:   "Manage document attachments",
}

var documentAttachmentAddCmd = &cobra.Command{
	Use:   "add <docid>",
	Short: "Upload a new attachment to a document",
	Long: `Upload a file via POST /multiapi/v1/attachments/{docid}.

--user is required (sent as user_code; when it does not match an existing
user, the server falls back to the authenticated user). --file accepts a
file path or "-" for stdin.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumentAttachmentAdd,
}

var documentAttachmentListCmd = &cobra.Command{
	Use:     "list <docid>",
	Aliases: []string{"ls"},
	Short:   "List attachments on a document",
	Long:    `List attachments via GET /api/v1/attachments/{docid}.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runDocumentAttachmentList,
}

var documentAttachmentGetCmd = &cobra.Command{
	Use:   "get <docid> <seq>",
	Short: "Download an attachment",
	Long: `Download an attachment via GET /api/v1/attachments/{docid}/{attach_seq}.

By default the file is saved to the current directory using the filename
provided by the server (Content-Disposition). Use --output to override:
  --output FILE    save to FILE
  --output DIR/    save into DIR/ using the server-provided filename
  --output -       write the file to stdout`,
	Args: cobra.ExactArgs(2),
	RunE: runDocumentAttachmentGet,
}

var documentAttachmentUpdateCmd = &cobra.Command{
	Use:   "update <docid> <seq>",
	Short: "Update an attachment (metadata and/or file)",
	Long: `Update an attachment via PATCH /multiapi/v1/attachments/{docid}/{attach_seq}.

Provide --file (and --file-name when reading from stdin) to replace the
uploaded file, otherwise only metadata (remarks, reason, etc.) is updated.`,
	Args: cobra.ExactArgs(2),
	RunE: runDocumentAttachmentUpdate,
}

var documentAttachmentDeleteCmd = &cobra.Command{
	Use:   "delete <docid> <seq>",
	Short: "Delete an attachment",
	Long: `Delete an attachment via PATCH /multiapi/v1/attachments/{docid}/{attach_seq}
with the delete form field set to true. --user is required.

By default the command prompts for confirmation. Pass --yes to skip it.`,
	Args: cobra.ExactArgs(2),
	RunE: runDocumentAttachmentDelete,
}

func init() {
	documentCmd.AddCommand(documentAttachmentCmd)
	documentAttachmentCmd.AddCommand(documentAttachmentAddCmd)
	documentAttachmentCmd.AddCommand(documentAttachmentListCmd)
	documentAttachmentCmd.AddCommand(documentAttachmentGetCmd)
	documentAttachmentCmd.AddCommand(documentAttachmentUpdateCmd)
	documentAttachmentCmd.AddCommand(documentAttachmentDeleteCmd)

	af := documentAttachmentAddCmd.Flags()
	af.StringVar(&docAttachAddUser, "user", "", "user code (required; sent as user_code)")
	af.StringVar(&docAttachAddFile, "file", "", "path to the file to upload (required; use - for stdin)")
	af.StringVar(&docAttachAddName, "file-name", "", "uploaded file name including extension (default: basename of --file)")
	af.StringVar(&docAttachAddRemarks, "remarks", "", "remarks / memo")
	af.StringVar(&docAttachAddOverwrite, "overwrite", "", "overwrite same-name attachment: true|false (server default true)")
	af.StringVar(&docAttachAddReason, "reason", "", "reason for overwrite (required for some forms)")
	af.IntVar(&docAttachAddDetailNo, "detail-no", 0, "detail row number (電帳法 attachment-type forms)")
	af.IntVar(&docAttachAddEvidenceType, "evidence-type", 0, "evidence type: 0 (other) | 1 (electronic transaction)")
	af.StringVarP(&docAttachAddOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	af.StringVar(&docAttachAddJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	lf := documentAttachmentListCmd.Flags()
	lf.StringVarP(&docAttachListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	lf.StringVar(&docAttachListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	gf := documentAttachmentGetCmd.Flags()
	gf.StringVarP(&docAttachGetOutput, "output", "o", "", "output path: FILE, DIR/, or - for stdout (default: server-provided filename in current directory)")

	uf := documentAttachmentUpdateCmd.Flags()
	uf.StringVar(&docAttachUpdateUser, "user", "", "user code (required; sent as user_code)")
	uf.StringVar(&docAttachUpdateFile, "file", "", "path to the replacement file (use - for stdin; omit to keep existing file)")
	uf.StringVar(&docAttachUpdateName, "file-name", "", "replacement file name including extension (default: basename of --file)")
	uf.StringVar(&docAttachUpdateRemarks, "remarks", "", "remarks / memo")
	uf.StringVar(&docAttachUpdateReason, "reason", "", "reason (required for some forms)")
	uf.IntVar(&docAttachUpdateDetailNo, "detail-no", 0, "detail row number (電帳法 attachment-type forms)")
	uf.IntVar(&docAttachUpdateEvidenceType, "evidence-type", 0, "evidence type: 0 (other) | 1 (electronic transaction)")
	uf.StringVarP(&docAttachUpdateOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	uf.StringVar(&docAttachUpdateJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	df := documentAttachmentDeleteCmd.Flags()
	df.StringVar(&docAttachDeleteUser, "user", "", "user code (required; sent as user_code)")
	df.StringVar(&docAttachDeleteReason, "reason", "", "reason (required for some forms)")
	df.BoolVarP(&docAttachDeleteYes, "yes", "y", false, "skip the interactive confirmation prompt")
	df.StringVarP(&docAttachDeleteOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	df.StringVar(&docAttachDeleteJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runDocumentAttachmentAdd(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	if docAttachAddUser == "" {
		return fmt.Errorf("--user is required")
	}
	if docAttachAddFile == "" {
		return fmt.Errorf("--file is required")
	}
	content, fileName, err := loadAttachmentFile(docAttachAddFile, docAttachAddName)
	if err != nil {
		return err
	}
	overwrite, err := parseTristateBool("overwrite", docAttachAddOverwrite)
	if err != nil {
		return err
	}
	req := xpoint.AddAttachmentRequest{
		UserCode:    docAttachAddUser,
		FileName:    fileName,
		FileContent: content,
		Remarks:     docAttachAddRemarks,
		Reason:      docAttachAddReason,
		Overwrite:   overwrite,
	}
	if cmd.Flags().Changed("detail-no") {
		v := docAttachAddDetailNo
		req.DetailNo = &v
	}
	if cmd.Flags().Changed("evidence-type") {
		v := docAttachAddEvidenceType
		req.EvidenceType = &v
	}

	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.AddAttachment(cmd.Context(), docID, req)
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(docAttachAddOutput), docAttachAddJQ, func() error {
		return renderAttachmentMutation(res)
	})
}

func runDocumentAttachmentList(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.ListAttachments(cmd.Context(), docID)
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(docAttachListOutput), docAttachListJQ, func() error {
		w := newTable(os.Stdout, "SEQ", "NAME", "SIZE", "CONTENT_TYPE", "REMARKS")
		for _, a := range res.Attachments {
			w.AddRow(a.Seq, a.Name, a.Size, a.ContentType, a.Remarks)
		}
		w.Print()
		return nil
	})
}

func runDocumentAttachmentGet(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	seq, err := parseAttachSeq(args[1])
	if err != nil {
		return err
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	filename, data, err := client.GetAttachment(cmd.Context(), docID, seq)
	if err != nil {
		return err
	}
	if docAttachGetOutput == "-" {
		_, werr := os.Stdout.Write(data)
		return werr
	}
	dst := resolveAttachmentPath(docAttachGetOutput, filename, docID, seq)
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("write attachment: %w", err)
	}
	fmt.Fprintf(os.Stderr, "saved: %s (%d bytes)\n", dst, len(data))
	return nil
}

func runDocumentAttachmentUpdate(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	seq, err := parseAttachSeq(args[1])
	if err != nil {
		return err
	}
	if docAttachUpdateUser == "" {
		return fmt.Errorf("--user is required")
	}
	req := xpoint.UpdateAttachmentRequest{
		UserCode: docAttachUpdateUser,
		Remarks:  docAttachUpdateRemarks,
		Reason:   docAttachUpdateReason,
	}
	if docAttachUpdateFile != "" {
		content, fileName, err := loadAttachmentFile(docAttachUpdateFile, docAttachUpdateName)
		if err != nil {
			return err
		}
		req.FileContent = content
		req.FileName = fileName
	}
	if cmd.Flags().Changed("detail-no") {
		v := docAttachUpdateDetailNo
		req.DetailNo = &v
	}
	if cmd.Flags().Changed("evidence-type") {
		v := docAttachUpdateEvidenceType
		req.EvidenceType = &v
	}

	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.UpdateAttachment(cmd.Context(), docID, seq, req)
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(docAttachUpdateOutput), docAttachUpdateJQ, func() error {
		return renderAttachmentMutation(res)
	})
}

func runDocumentAttachmentDelete(cmd *cobra.Command, args []string) error {
	docID, err := parseDocID(args[0])
	if err != nil {
		return err
	}
	seq, err := parseAttachSeq(args[1])
	if err != nil {
		return err
	}
	if docAttachDeleteUser == "" {
		return fmt.Errorf("--user is required")
	}
	if !docAttachDeleteYes && !confirmDeleteAttachment(docID, seq) {
		return fmt.Errorf("aborted")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.UpdateAttachment(cmd.Context(), docID, seq, xpoint.UpdateAttachmentRequest{
		UserCode: docAttachDeleteUser,
		Delete:   true,
		Reason:   docAttachDeleteReason,
	})
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(docAttachDeleteOutput), docAttachDeleteJQ, func() error {
		return renderAttachmentMutation(res)
	})
}

func renderAttachmentMutation(res *xpoint.AttachmentMutationResponse) error {
	w := newTable(os.Stdout, "DOCID", "SEQ", "MESSAGE_TYPE", "MESSAGE", "DETAIL")
	w.AddRow(res.DocID, res.Seq, res.MessageType, res.Message, res.Detail)
	w.Print()
	return nil
}

// loadAttachmentFile reads the given path (or stdin when "-"), and resolves
// the uploaded file name (from --file-name or the basename of the path).
func loadAttachmentFile(path, nameFlag string) ([]byte, string, error) {
	if path == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, "", fmt.Errorf("read --file from stdin: %w", err)
		}
		if nameFlag == "" {
			return nil, "", fmt.Errorf("--file-name is required when reading from stdin")
		}
		return data, nameFlag, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read --file: %w", err)
	}
	name := nameFlag
	if name == "" {
		name = filepath.Base(path)
	}
	return data, name, nil
}

// parseTristateBool returns nil for empty, a bool pointer for "true"/"false".
func parseTristateBool(flag, s string) (*bool, error) {
	if s == "" {
		return nil, nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return nil, fmt.Errorf("--%s must be true or false, got %q", flag, s)
	}
	return &b, nil
}

func parseAttachSeq(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid attach seq %q: must be a positive integer", s)
	}
	return n, nil
}

func confirmDeleteAttachment(docID, seq int) bool {
	fmt.Fprintf(os.Stderr, "Really delete attachment seq=%d on document %d? [y/N]: ", seq, docID)
	var ans string
	_, _ = fmt.Fscanln(os.Stdin, &ans)
	switch strings.ToLower(strings.TrimSpace(ans)) {
	case "y", "yes":
		return true
	}
	return false
}

func resolveAttachmentPath(out, serverName string, docID, seq int) string {
	name := filepath.Base(filepath.Clean(serverName))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = fmt.Sprintf("%d_%d", docID, seq)
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
