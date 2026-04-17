package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	proxyOutput string
	proxyJQ     string
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Show delegation (proxy) info for the authenticated user",
	Long: `Fetch delegation info via GET /api/v1/proxy.

Lists the users who have delegated apply/approve authority to the
authenticated user.`,
	RunE: runProxy,
}

func init() {
	rootCmd.AddCommand(proxyCmd)
	proxyCmd.Flags().StringVarP(&proxyOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	proxyCmd.Flags().StringVar(&proxyJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runProxy(cmd *cobra.Command, _ []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.GetProxyInfo(cmd.Context())
	if err != nil {
		return err
	}
	return render(res, resolveOutputFormat(proxyOutput), proxyJQ, func() error {
		if len(res.Proxy) == 0 {
			fmt.Fprintln(os.Stdout, "(no proxy delegations)")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "USER_CODE\tUSER_NAME\tAPPLY\tAPRV")
		for _, p := range res.Proxy {
			fmt.Fprintf(w, "%s\t%s\t%t\t%t\n", p.Use.Code, p.Use.Name, p.Apply, p.Aprv)
		}
		return nil
	})
}
