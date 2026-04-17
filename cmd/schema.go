package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var schemaJQ string

var schemaCmd = &cobra.Command{
	Use:   "schema [service.resource.method]",
	Short: "Show the schema for an X-point operation",
	Long: `Print the schema object for a given X-point endpoint.

Supported aliases map to the CLI's commands:
  form.list         GET  /api/v1/forms
  approval.list     GET  /api/v1/approvals
  document.search   POST /api/v1/search/documents

Run without arguments to list supported aliases.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSchema,
}

func init() {
	rootCmd.AddCommand(schemaCmd)
	schemaCmd.Flags().StringVar(&schemaJQ, "jq", "", "apply a gojq filter to the schema")
}

func runSchema(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stdout, "Supported aliases:")
		for _, a := range xpoint.SchemaAliases() {
			fmt.Fprintf(os.Stdout, "  %s\n", a)
		}
		return nil
	}
	alias := strings.TrimSpace(args[0])
	op, err := xpoint.LookupOperation(alias)
	if err != nil {
		return err
	}
	if schemaJQ != "" {
		return runJQ(op, schemaJQ)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(op)
}
