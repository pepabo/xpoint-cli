package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pepabo/xpoint-cli/internal/schema"
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema [service.resource.method]",
	Short: "Show the schema for an X-point operation",
	Long: `Print the schema object for a given X-point endpoint.

Supported aliases map to the CLI's commands:
  form.list         GET    /api/v1/forms
  approval.list     GET    /api/v1/approvals
  document.search   POST   /api/v1/search/documents
  document.create   POST   /api/v1/documents
  document.get      GET    /api/v1/documents/{docid}
  document.update   PATCH  /api/v1/documents/{docid}
  document.delete   DELETE /api/v1/documents/{docid}

Run without arguments to list supported aliases.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSchema,
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}

func runSchema(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stdout, "Supported aliases:")
		for _, a := range schema.Aliases() {
			fmt.Fprintf(os.Stdout, "  %s\n", a)
		}
		return nil
	}
	alias := strings.TrimSpace(args[0])
	op, err := schema.Lookup(alias)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(op)
}
