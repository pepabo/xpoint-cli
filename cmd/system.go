package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	systemFormListOutput string
	systemFormListJQ     string
	systemFormShowOutput string
	systemFormShowJQ     string
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "X-point system (admin) APIs",
}

var systemFormCmd = &cobra.Command{
	Use:   "form",
	Short: "Manage X-point forms via admin APIs",
}

var systemFormListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registration forms (admin)",
	Long:  "List all registered forms via GET /api/v1/system/forms. Requires an administrator account.",
	RunE:  runSystemFormList,
}

var systemFormShowCmd = &cobra.Command{
	Use:   "show <form_code|form_id>",
	Short: "Show a form definition (admin)",
	Long: `Fetch field definitions via GET /api/v1/system/forms/{fid}.

The argument may be a form_code (e.g. "TORIHIKISAKI_a") or a numeric
form_id. When a form_code is given, the CLI first calls
/api/v1/system/forms to resolve the id. Requires an administrator
account.`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemFormShow,
}

func init() {
	rootCmd.AddCommand(systemCmd)
	systemCmd.AddCommand(systemFormCmd)
	systemFormCmd.AddCommand(systemFormListCmd)
	systemFormCmd.AddCommand(systemFormShowCmd)

	lf := systemFormListCmd.Flags()
	lf.StringVarP(&systemFormListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	lf.StringVar(&systemFormListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	sf := systemFormShowCmd.Flags()
	sf.StringVarP(&systemFormShowOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	sf.StringVar(&systemFormShowJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runSystemFormList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.ListSystemForms(cmd.Context())
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(systemFormListOutput), systemFormListJQ, func() error {
		w := newTable(os.Stdout, "GROUP_ID", "GROUP_NAME", "FORMS", "FORM_ID", "FORM_CODE", "FORM_NAME", "PAGES", "TABLE")
		for _, g := range res.FormGroup {
			if len(g.Form) == 0 {
				w.AddRow(g.ID, g.Name, g.FormCount, "-", "-", "-", "-", "-")
				continue
			}
			for _, f := range g.Form {
				w.AddRow(g.ID, g.Name, g.FormCount, f.ID, f.Code, f.Name, f.PageCount, f.TableName)
			}
		}
		w.Print()
		return nil
	})
}

func runSystemFormShow(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	formID, err := resolveSystemFormID(cmd.Context(), client, args[0])
	if err != nil {
		return err
	}

	res, err := client.GetSystemFormDetail(cmd.Context(), formID)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(systemFormShowOutput), systemFormShowJQ, func() error {
		form := res.Form
		fmt.Fprintf(os.Stdout, "FORM: %s  %s  MAX_STEP: %d\n", form.Code, form.Name, form.MaxStep)
		w := newTable(os.Stdout, "PAGE", "FIELD_ID", "TYPE", "REQUIRED", "UNIQUE", "ARRAYSIZE", "LABEL")
		for _, p := range form.Pages {
			for _, f := range p.Fields {
				w.AddRow(p.PageNo, f.FieldID, f.FieldType, f.Required, f.Unique, f.ArraySize, f.Label)
			}
		}
		w.Print()
		return nil
	})
}

type systemFormLister interface {
	ListSystemForms(ctx context.Context) (*xpoint.SystemFormsListResponse, error)
}

// resolveSystemFormID mirrors resolveFormID but consults the admin forms list.
func resolveSystemFormID(ctx context.Context, lister systemFormLister, arg string) (int, error) {
	if id, err := strconv.Atoi(arg); err == nil {
		return id, nil
	}
	forms, err := lister.ListSystemForms(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolve form code: %w", err)
	}
	for _, g := range forms.FormGroup {
		for _, f := range g.Form {
			if f.Code == arg {
				return f.ID, nil
			}
		}
	}
	return 0, fmt.Errorf("form code %q not found", arg)
}
