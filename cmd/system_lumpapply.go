package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	systemLumpapplyListOutput string
	systemLumpapplyListJQ     string
	systemLumpapplyShowJQ     string
)

var systemLumpapplyCmd = &cobra.Command{
	Use:   "lumpapply",
	Short: "Manage X-point auto-apply (lumpapply) definitions",
}

var systemLumpapplyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List auto-apply (lumpapply) registrations",
	Long:  "List auto-apply registrations via GET /api/v1/system/lumpapply. Requires an administrator account.",
	RunE:  runSystemLumpapplyList,
}

var systemLumpapplyShowCmd = &cobra.Command{
	Use:   "show <lumpapplyid>",
	Short: "Show an auto-apply (lumpapply) definition",
	Long: `Fetch definition details via GET /api/v1/system/lumpapply/{lumpapplyid}.

The argument is the numeric lumpapplyid (shown by ` + "`" + `system lumpapply list` + "`" + `).
The response is returned as JSON (shape: csv_format / apply / create / lastaprv / ...).
Requires an administrator account.`,
	Args: cobra.ExactArgs(1),
	RunE: runSystemLumpapplyShow,
}

func init() {
	systemCmd.AddCommand(systemLumpapplyCmd)
	systemLumpapplyCmd.AddCommand(systemLumpapplyListCmd)
	systemLumpapplyCmd.AddCommand(systemLumpapplyShowCmd)

	lf := systemLumpapplyListCmd.Flags()
	lf.StringVarP(&systemLumpapplyListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	lf.StringVar(&systemLumpapplyListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	sf := systemLumpapplyShowCmd.Flags()
	sf.StringVar(&systemLumpapplyShowJQ, "jq", "", "apply a gojq filter to the JSON response")
}

func runSystemLumpapplyList(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.ListLumpapply(cmd.Context())
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(systemLumpapplyListOutput), systemLumpapplyListJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "ID\tNAME\tFORM_ID\tFORM_CODE\tFORM_NAME\tROUTE_ID\tROUTE_CODE\tROUTE_NAME")
		for _, l := range res.Lumpapply {
			fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%d\t%s\t%s\n",
				l.ID, l.Name, l.FormID, l.FormCode, l.FormName, l.RouteID, l.RouteCode, l.RouteName,
			)
		}
		return nil
	})
}

func runSystemLumpapplyShow(cmd *cobra.Command, args []string) error {
	idArg := strings.TrimSpace(args[0])
	lumpapplyID, err := strconv.Atoi(idArg)
	if err != nil || lumpapplyID <= 0 {
		return fmt.Errorf("lumpapplyid must be a positive integer: %q", args[0])
	}

	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	raw, err := client.GetLumpapply(cmd.Context(), lumpapplyID)
	if err != nil {
		return err
	}
	if systemLumpapplyShowJQ != "" {
		return runJQ(raw, systemLumpapplyShowJQ)
	}
	return writeJSON(os.Stdout, raw)
}
