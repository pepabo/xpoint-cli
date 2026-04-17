package cmd

import (
	"fmt"
	"os"
	"strings"

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

	approvalWaitFGID   int
	approvalWaitFID    int
	approvalWaitStep   int
	approvalWaitOutput string
	approvalWaitJQ     string

	approvalHiddenShow      bool
	approvalHiddenProxyUser string
	approvalHiddenOutput    string
	approvalHiddenJQ        string
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

var approvalWaitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Get approval counts and latest pending documents",
	Long:  "Fetch approval counts and the latest pending documents via GET /api/v1/approvals/wait.",
	RunE:  runApprovalWait,
}

var approvalHiddenCmd = &cobra.Command{
	Use:   "hidden <docid> [<docid>...]",
	Short: "Hide or show completed approval documents",
	Long: `Set the hidden flag on completed approval documents via PUT /api/v1/approvals/hidden.

By default the given document IDs are hidden. Pass --show to restore them.
Only documents in the "承認完了" (completed) status may be specified.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runApprovalHidden,
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

	approvalCmd.AddCommand(approvalWaitCmd)
	fw := approvalWaitCmd.Flags()
	fw.IntVar(&approvalWaitFGID, "fgid", 0, "form group ID (0 = omit)")
	fw.IntVar(&approvalWaitFID, "fid", 0, "form ID (0 = omit)")
	fw.IntVar(&approvalWaitStep, "step", 0, "approval step (0 = omit)")
	fw.StringVarP(&approvalWaitOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	fw.StringVar(&approvalWaitJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	approvalCmd.AddCommand(approvalHiddenCmd)
	fh := approvalHiddenCmd.Flags()
	fh.BoolVar(&approvalHiddenShow, "show", false, "show (un-hide) the given documents instead of hiding them")
	fh.StringVar(&approvalHiddenProxyUser, "proxy-user", "", "proxy approver user code (required when toggling proxy-approved documents)")
	fh.StringVarP(&approvalHiddenOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	fh.StringVar(&approvalHiddenJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runApprovalList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
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
		fmt.Fprintf(os.Stdout, "total: %d\n", res.TotalCount)
		w := newTable(os.Stdout,
			"DOCID", "STATUS", "FORM_NAME", "APPLY_USER", "APPROVERS", "APPLY_DATETIME", "TITLE1")
		for _, a := range res.ApprovalList {
			w.AddRow(a.DocID, a.DisplayStatus, a.FormName, a.ApplyUser,
				strings.Join(a.ApprovalUser, ","), a.ApplyDatetime, a.Title1)
		}
		w.Print()
		return nil
	})
}

func runApprovalWait(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	var params xpoint.ApprovalsWaitParams
	if approvalWaitFGID != 0 {
		v := approvalWaitFGID
		params.FormGroupID = &v
	}
	if approvalWaitFID != 0 {
		v := approvalWaitFID
		params.FormID = &v
	}
	if approvalWaitStep != 0 {
		v := approvalWaitStep
		params.Step = &v
	}

	res, err := client.GetApprovalsWait(cmd.Context(), params)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(approvalWaitOutput), approvalWaitJQ, func() error {
		w := newTable(os.Stdout, "TYPE", "NAME", "COUNT")
		for _, s := range res.StatusList {
			w.AddRow(s.Type, s.Name, s.Count)
		}
		w.Print()

		if len(res.WaitList) > 0 {
			fmt.Fprintln(os.Stdout)
			fmt.Fprintln(os.Stdout, "最新の承認待ち書類:")
			w2 := newTable(os.Stdout, "DOCID", "FORM_NAME", "WRITER", "WRITE_DATE", "TITLE")
			for _, d := range res.WaitList {
				w2.AddRow(d.DocID, d.Name, d.WriterName, d.WriteDate, d.Title)
			}
			w2.Print()
		}
		return nil
	})
}

func runApprovalHidden(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	docIDs := make([]int, 0, len(args))
	for _, a := range args {
		id, err := parseDocID(a)
		if err != nil {
			return fmt.Errorf("invalid docid %q: %w", a, err)
		}
		docIDs = append(docIDs, id)
	}

	req := xpoint.SetApprovalsHiddenRequest{
		Hidden:    !approvalHiddenShow,
		DocID:     docIDs,
		ProxyUser: approvalHiddenProxyUser,
	}

	res, err := client.SetApprovalsHidden(cmd.Context(), req)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(approvalHiddenOutput), approvalHiddenJQ, func() error {
		fmt.Fprintf(os.Stdout, "message: %s\n", res.Message)
		w := newTable(os.Stdout, "DOCID")
		for _, id := range res.DocID {
			w.AddRow(id)
		}
		w.Print()
		return nil
	})
}
