package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	formListOutput string
	formListJQ     string
)

var formCmd = &cobra.Command{
	Use:   "form",
	Short: "Manage X-point forms",
}

var formListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available forms",
	Long:  "List available forms via GET /api/v1/forms.",
	RunE:  runFormList,
}

func init() {
	rootCmd.AddCommand(formCmd)
	formCmd.AddCommand(formListCmd)
	formListCmd.Flags().StringVarP(&formListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	formListCmd.Flags().StringVar(&formListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runFormList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags()
	if err != nil {
		return err
	}
	res, err := client.ListAvailableForms(cmd.Context())
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(formListOutput), formListJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "GROUP_ID\tGROUP_NAME\tFORM_ID\tFORM_CODE\tFORM_NAME")
		for _, g := range res.FormGroup {
			if len(g.Form) == 0 {
				fmt.Fprintf(w, "%d\t%s\t-\t-\t-\n", g.ID, g.Name)
				continue
			}
			for _, f := range g.Form {
				fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\n", g.ID, g.Name, f.ID, f.Code, f.Name)
			}
		}
		return nil
	})
}

func newClientFromFlags() (*xpoint.Client, error) {
	sub, err := resolveSubdomain()
	if err != nil {
		return nil, err
	}
	auth, err := resolveAuth()
	if err != nil {
		return nil, err
	}
	return xpoint.NewClient(sub, auth), nil
}
