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
	approvalListStat          int
	approvalListFGID          int
	approvalListFID           int
	approvalListStep          int
	approvalListRecordNo      int
	approvalListGetLine       int
	approvalListProxyUser     string
	approvalListFilter        string
	approvalListShowHiddenDoc bool
	approvalListOutput        string
	approvalListJQ            string
)

var approvalCmd = &cobra.Command{
	Use:   "approval",
	Short: "Manage X-point approvals",
}

var approvalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List approval documents",
	Long:  "List approval documents via GET /api/v1/approvals.",
	RunE:  runApprovalList,
}

func init() {
	rootCmd.AddCommand(approvalCmd)
	approvalCmd.AddCommand(approvalListCmd)
	f := approvalListCmd.Flags()
	f.IntVar(&approvalListStat, "stat", 10, "approval status type (required): 10=承認待ち, 20=通知, 30=下書き等, 40=状況確認, 50=承認完了 and sub-types (see manual)")
	f.IntVar(&approvalListFGID, "fgid", 0, "form group ID (0 = omit)")
	f.IntVar(&approvalListFID, "fid", 0, "form ID (0 = omit; only valid with --fgid)")
	f.IntVar(&approvalListStep, "step", 0, "approval step (0 = omit; only valid with --fgid and --fid)")
	f.IntVar(&approvalListRecordNo, "record-no", 0, "starting record number")
	f.IntVar(&approvalListGetLine, "get-line", 0, "number of rows to fetch (0 = omit, server default 50; max 1000)")
	f.StringVar(&approvalListProxyUser, "proxy-user", "", "proxy user code")
	f.StringVar(&approvalListFilter, "filter", "", "date-range filter, e.g. cr_dt between \"2023-01-01\" and \"2023-12-31\"")
	f.BoolVar(&approvalListShowHiddenDoc, "show-hidden-doc", false, "include hidden completed documents")
	f.StringVarP(&approvalListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	f.StringVar(&approvalListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runApprovalList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags()
	if err != nil {
		return err
	}

	params := xpoint.ApprovalsListParams{
		Stat:      approvalListStat,
		ProxyUser: approvalListProxyUser,
		Filter:    approvalListFilter,
	}
	if approvalListFGID != 0 {
		v := approvalListFGID
		params.FormGroupID = &v
	}
	if approvalListFID != 0 {
		v := approvalListFID
		params.FormID = &v
	}
	if approvalListStep != 0 {
		v := approvalListStep
		params.Step = &v
	}
	if cmd.Flags().Changed("record-no") {
		v := approvalListRecordNo
		params.RecordNo = &v
	}
	if approvalListGetLine != 0 {
		v := approvalListGetLine
		params.GetLine = &v
	}
	if cmd.Flags().Changed("show-hidden-doc") {
		v := approvalListShowHiddenDoc
		params.ShowHiddenDoc = &v
	}

	res, err := client.ListApprovals(cmd.Context(), params)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(approvalListOutput), approvalListJQ, func() error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintf(os.Stdout, "total: %d\n", res.TotalCount)
		fmt.Fprintln(w, "DOCID\tSTATUS\tFORM_NAME\tAPPLY_USER\tAPPROVERS\tAPPLY_DATETIME\tTITLE1")
		for _, a := range res.ApprovalList {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
				a.DocID, a.DisplayStatus, a.FormName, a.ApplyUser,
				strings.Join(a.ApprovalUser, ","), a.ApplyDatetime, a.Title1,
			)
		}
		return nil
	})
}
