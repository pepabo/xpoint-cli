package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/itchyny/gojq"
)

// resolveOutputFormat returns "json" or "table" based on the flag and TTY state.
func resolveOutputFormat(flag string) string {
	if flag != "" {
		return flag
	}
	if isTerminal(os.Stdout) {
		return "table"
	}
	return "json"
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// render dispatches to the appropriate renderer based on jq/format.
func render(v any, format, jqExpr string, table func() error) error {
	if jqExpr != "" {
		return runJQ(v, jqExpr)
	}
	switch format {
	case "json":
		return writeJSON(os.Stdout, v)
	case "table":
		return table()
	default:
		return fmt.Errorf("unknown output format: %q (supported: table, json)", format)
	}
}

func writeJSON(out *os.File, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func runJQ(v any, expr string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal response for jq: %w", err)
	}
	var input any
	if err := json.Unmarshal(b, &input); err != nil {
		return fmt.Errorf("unmarshal response for jq: %w", err)
	}

	q, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid --jq filter: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	iter := q.Run(input)
	for {
		val, ok := iter.Next()
		if !ok {
			break
		}
		if e, ok := val.(error); ok {
			return fmt.Errorf("jq error: %w", e)
		}
		if err := enc.Encode(val); err != nil {
			return err
		}
	}
	return nil
}
