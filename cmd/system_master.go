package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

const masterDataPageSize = 1000

var (
	systemMasterListOutput string
	systemMasterListJQ     string
	systemMasterShowOutput string
	systemMasterShowJQ     string

	systemMasterDataType      int
	systemMasterDataFormat    string
	systemMasterDataFileName  string
	systemMasterDataDelimiter string
	systemMasterDataTitle     bool
	systemMasterDataNoTitle   bool
	systemMasterDataFields    string
	systemMasterDataOutput    string
	systemMasterDataJQ        string

	systemMasterImportOverwrite bool
	systemMasterImportData      string
	systemMasterImportJQ        string

	systemMasterUploadFile      string
	systemMasterUploadOverwrite bool
)

var systemMasterCmd = &cobra.Command{
	Use:   "master",
	Short: "Manage X-point masters via admin APIs",
}

var systemMasterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List masters (admin)",
	Long:  "List all masters via GET /api/v1/system/master. Requires an administrator account.",
	RunE:  runSystemMasterList,
}

var systemMasterShowCmd = &cobra.Command{
	Use:   "show <master_table_name>",
	Short: "Show a user-specific master's property definition",
	Long: `Fetch user-specific master property info via
GET /api/v1/system/master/{master_table_name}. Requires an administrator
account.`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemMasterShow,
}

var systemMasterDataCmd = &cobra.Command{
	Use:   "data <master_code>",
	Short: "Export master data (JSON or CSV)",
	Long: `Export master rows via GET /api/v1/system/master/{master_code}/data.

--type (required) selects the master kind:
  0  simple master
  1  user-specific master (pass the table_name as <master_code>)

--format defaults to json. Use --format csv for CSV output; the CSV
payload is written to stdout (or --output FILE / DIR/).`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemMasterData,
}

var systemMasterImportCmd = &cobra.Command{
	Use:   "import <master_code>",
	Short: "Import rows into a simple master",
	Long: `Import data rows into a simple master via
PUT /api/v1/system/master/{master_code}/data.

--data takes a JSON array of {"code","value"} objects, either inline,
as a file path, or - for stdin.
Pass --overwrite to replace existing data instead of appending.`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemMasterImport,
}

var systemMasterUploadCmd = &cobra.Command{
	Use:   "upload <master_table_name>",
	Short: "Upload a CSV for a user-specific master's import staging",
	Long: `Upload a CSV via POST /multiapi/v1/system/master/{master_table_name}/data.

The upload only stages the file; the import itself is run later from
the admin site's task management (manually or by schedule).`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemMasterUpload,
}

func init() {
	systemCmd.AddCommand(systemMasterCmd)
	systemMasterCmd.AddCommand(systemMasterListCmd)
	systemMasterCmd.AddCommand(systemMasterShowCmd)
	systemMasterCmd.AddCommand(systemMasterDataCmd)
	systemMasterCmd.AddCommand(systemMasterImportCmd)
	systemMasterCmd.AddCommand(systemMasterUploadCmd)

	lf := systemMasterListCmd.Flags()
	lf.StringVarP(&systemMasterListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	lf.StringVar(&systemMasterListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	sf := systemMasterShowCmd.Flags()
	sf.StringVarP(&systemMasterShowOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	sf.StringVar(&systemMasterShowJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	df := systemMasterDataCmd.Flags()
	df.IntVar(&systemMasterDataType, "type", -1, "master_type: 0=simple master, 1=user-specific master (required)")
	df.StringVar(&systemMasterDataFormat, "format", "json", "output format: json | csv")
	df.StringVar(&systemMasterDataFileName, "file-name", "", "CSV file name hint (CSV only; default: {master_code}.csv)")
	df.StringVar(&systemMasterDataDelimiter, "delimiter", "", "CSV delimiter: comma | tab (CSV only; default comma)")
	df.BoolVar(&systemMasterDataTitle, "title", false, "CSV only (user-specific master): include field names on the first row (default: true)")
	df.BoolVar(&systemMasterDataNoTitle, "no-title", false, "CSV only (user-specific master): omit field names from the first row")
	df.StringVar(&systemMasterDataFields, "fields", "", "CSV only (simple master): comma-separated list of field names to include")
	df.StringVarP(&systemMasterDataOutput, "output", "o", "", "output path: FILE, DIR/, - for stdout (default: stdout)")
	df.StringVar(&systemMasterDataJQ, "jq", "", "apply a gojq filter to the JSON response (JSON format only)")
	_ = systemMasterDataCmd.MarkFlagRequired("type")

	imf := systemMasterImportCmd.Flags()
	imf.BoolVar(&systemMasterImportOverwrite, "overwrite", false, "replace existing simple master data instead of appending")
	imf.StringVar(&systemMasterImportData, "data", "", "JSON array of {\"code\",\"value\"} rows: inline, file path, or - for stdin (required)")
	imf.StringVar(&systemMasterImportJQ, "jq", "", "apply a gojq filter to the JSON response")
	_ = systemMasterImportCmd.MarkFlagRequired("data")

	uf := systemMasterUploadCmd.Flags()
	uf.StringVar(&systemMasterUploadFile, "file", "", "path to the CSV file to upload, or - for stdin (required)")
	uf.BoolVar(&systemMasterUploadOverwrite, "overwrite", false, "overwrite the existing staged CSV for this master")
	_ = systemMasterUploadCmd.MarkFlagRequired("file")
}

func runSystemMasterList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.ListMasters(cmd.Context())
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(systemMasterListOutput), systemMasterListJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "TYPE\tTYPE_NAME\tCODE\tTABLE_NAME\tITEMS\tNAME\tREMARKS")
		for _, m := range res.Master {
			code := m.Code
			if code == "" {
				code = "-"
			}
			tbl := m.TableName
			if tbl == "" {
				tbl = "-"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%s\t%s\n",
				m.Type, m.TypeName, code, tbl, m.ItemCount, m.Name, m.Remarks,
			)
		}
		return nil
	})
}

func runSystemMasterShow(cmd *cobra.Command, args []string) error {
	tableName := strings.TrimSpace(args[0])
	if tableName == "" {
		return fmt.Errorf("master_table_name is required")
	}
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.GetUserMasterInfo(cmd.Context(), tableName)
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(systemMasterShowOutput), systemMasterShowJQ, func() error {
		fmt.Fprintf(os.Stdout, "TABLE: %s\n", res.TableName)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "ID\tTYPE\tLENGTH\tPK\tINDEX")
		for _, f := range res.Fields {
			fmt.Fprintf(w, "%s\t%s\t%v\t%t\t%t\n", f.ID, f.Type, f.Length, f.PrimaryKey, f.Index)
		}
		return nil
	})
}

func runSystemMasterData(cmd *cobra.Command, args []string) error {
	masterCode := strings.TrimSpace(args[0])
	if masterCode == "" {
		return fmt.Errorf("master_code is required")
	}
	if systemMasterDataType != 0 && systemMasterDataType != 1 {
		return fmt.Errorf("--type must be 0 (simple) or 1 (user-specific), got %d", systemMasterDataType)
	}
	format := strings.ToLower(strings.TrimSpace(systemMasterDataFormat))
	switch format {
	case "", "json":
		format = "json"
	case "csv":
	default:
		return fmt.Errorf("unknown --format %q (must be json or csv)", systemMasterDataFormat)
	}
	if systemMasterDataTitle && systemMasterDataNoTitle {
		return fmt.Errorf("--title and --no-title are mutually exclusive")
	}

	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	if format == "json" {
		merged, err := fetchAllMasterDataJSON(cmd.Context(), client, masterCode, systemMasterDataType)
		if err != nil {
			return err
		}
		if systemMasterDataJQ != "" {
			return runJQ(json.RawMessage(merged), systemMasterDataJQ)
		}
		switch systemMasterDataOutput {
		case "", "-":
			_, werr := os.Stdout.Write(merged)
			return werr
		}
		dst := resolveDownloadPath(systemMasterDataOutput, masterCode+".json", 0)
		if err := os.WriteFile(dst, merged, 0o600); err != nil {
			return fmt.Errorf("write master data: %w", err)
		}
		fmt.Fprintf(os.Stderr, "saved: %s (%d bytes)\n", dst, len(merged))
		return nil
	}

	p := xpoint.MasterDataParams{
		MasterType: systemMasterDataType,
		FileName:   systemMasterDataFileName,
		Delimiter:  systemMasterDataDelimiter,
		Fields:     systemMasterDataFields,
	}
	if systemMasterDataNoTitle {
		b := false
		p.Title = &b
	} else if cmd.Flags().Changed("title") {
		v := systemMasterDataTitle
		p.Title = &v
	}
	filename, body, err := fetchAllMasterDataCSV(cmd.Context(), client, masterCode, p)
	if err != nil {
		return err
	}

	if systemMasterDataOutput == "" || systemMasterDataOutput == "-" {
		_, werr := os.Stdout.Write(body)
		return werr
	}
	dst := resolveDownloadPath(systemMasterDataOutput, fallbackName(filename, masterCode+".csv"), 0)
	if err := os.WriteFile(dst, body, 0o600); err != nil {
		return fmt.Errorf("write csv: %w", err)
	}
	fmt.Fprintf(os.Stderr, "saved: %s (%d bytes)\n", dst, len(body))
	return nil
}

func runSystemMasterImport(cmd *cobra.Command, args []string) error {
	masterCode := strings.TrimSpace(args[0])
	if masterCode == "" {
		return fmt.Errorf("master_code is required")
	}
	raw, err := loadStringInput(systemMasterImportData)
	if err != nil {
		return fmt.Errorf("load --data: %w", err)
	}
	var items []xpoint.SimpleMasterDataItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return fmt.Errorf("--data must be a JSON array of {\"code\",\"value\"}: %w", err)
	}

	req := xpoint.ImportSimpleMasterRequest{Data: items}
	if cmd.Flags().Changed("overwrite") {
		v := systemMasterImportOverwrite
		req.Overwrite = &v
	}

	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	out, err := client.ImportSimpleMasterData(cmd.Context(), masterCode, req)
	if err != nil {
		return err
	}
	if systemMasterImportJQ != "" {
		return runJQ(out, systemMasterImportJQ)
	}
	return writeJSON(os.Stdout, out)
}

func runSystemMasterUpload(cmd *cobra.Command, args []string) error {
	tableName := strings.TrimSpace(args[0])
	if tableName == "" {
		return fmt.Errorf("master_table_name is required")
	}
	content, fileName, err := readUploadFile(systemMasterUploadFile)
	if err != nil {
		return fmt.Errorf("read --file: %w", err)
	}

	var overwrite *bool
	if cmd.Flags().Changed("overwrite") {
		v := systemMasterUploadOverwrite
		overwrite = &v
	}

	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.UploadUserMasterCSV(cmd.Context(), tableName, fileName, content, overwrite)
	if err != nil {
		return err
	}
	return writeJSON(os.Stdout, res)
}

// fetchAllMasterDataJSON fetches master data as JSON with rows=1000, paging
// until total_count (or a short/empty page) is reached. The merged JSON keeps
// the first page's envelope with the concatenated data array.
func fetchAllMasterDataJSON(ctx context.Context, client *xpoint.Client, masterCode string, masterType int) ([]byte, error) {
	rows := masterDataPageSize
	offset := 0
	p := xpoint.MasterDataParams{MasterType: masterType, Rows: &rows, Offset: &offset}

	var (
		firstMaster map[string]json.RawMessage
		allData     []json.RawMessage
		totalCount  int
		haveTotal   bool
	)

	for {
		p.Offset = &offset
		_, body, _, err := client.GetMasterData(ctx, masterCode, "json", p)
		if err != nil {
			return nil, err
		}
		var env struct {
			Master map[string]json.RawMessage `json:"master"`
		}
		if err := json.Unmarshal(body, &env); err != nil {
			return nil, fmt.Errorf("parse master data: %w", err)
		}

		if offset == 0 {
			firstMaster = env.Master
			if raw, ok := env.Master["total_count"]; ok {
				if err := json.Unmarshal(raw, &totalCount); err == nil {
					haveTotal = true
				}
			}
		}

		var data []json.RawMessage
		if raw, ok := env.Master["data"]; ok {
			if err := json.Unmarshal(raw, &data); err != nil {
				return nil, fmt.Errorf("parse master.data: %w", err)
			}
		}
		allData = append(allData, data...)

		if haveTotal && len(allData) >= totalCount {
			break
		}
		if len(data) < masterDataPageSize {
			break
		}
		offset += masterDataPageSize
	}

	if firstMaster == nil {
		firstMaster = map[string]json.RawMessage{}
	}
	encoded, err := json.Marshal(allData)
	if err != nil {
		return nil, err
	}
	firstMaster["data"] = encoded
	return json.Marshal(struct {
		Master map[string]json.RawMessage `json:"master"`
	}{Master: firstMaster})
}

// fetchAllMasterDataCSV fetches master data as CSV with rows=1000, paging
// until the server returns an empty page. The first page honors the caller's
// title setting; subsequent pages force title=false so the server omits the
// header and the bodies can simply be concatenated.
func fetchAllMasterDataCSV(ctx context.Context, client *xpoint.Client, masterCode string, base xpoint.MasterDataParams) (string, []byte, error) {
	rows := masterDataPageSize
	offset := 0
	p := base
	p.Rows = &rows
	p.Offset = &offset

	titleFalse := false
	var (
		firstFilename string
		csvBuf        bytes.Buffer
	)
	for {
		p.Offset = &offset
		if offset > 0 {
			p.Title = &titleFalse
		}
		filename, body, _, err := client.GetMasterData(ctx, masterCode, "csv", p)
		if err != nil {
			return "", nil, err
		}
		if offset == 0 {
			firstFilename = filename
			csvBuf.Write(body)
		} else {
			if len(bytes.TrimSpace(body)) == 0 {
				break
			}
			csvBuf.Write(body)
		}
		offset += masterDataPageSize
	}
	return firstFilename, csvBuf.Bytes(), nil
}

// fallbackName returns name when non-empty, else alt.
func fallbackName(name, alt string) string {
	if name != "" {
		return name
	}
	return alt
}

// readUploadFile reads the CSV file contents and returns the bytes plus a
// suggested filename for the multipart form-data part. "-" reads from stdin
// and yields a synthetic "upload.csv" filename.
func readUploadFile(path string) ([]byte, string, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, "", fmt.Errorf("read stdin: %w", err)
		}
		return b, "upload.csv", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return b, filepath.Base(path), nil
}
