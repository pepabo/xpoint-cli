package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
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

// tablePrinter renders tab-aligned rows to out. Column widths are computed
// from the widest cell (header or row) using go-runewidth, so CJK wide
// characters align correctly. When out is a TTY, header cells are wrapped in
// the ANSI underline SGR code (no other decoration).
type tablePrinter struct {
	out      io.Writer
	isTTY    bool
	headers  []string
	rows     [][]string
	showHead bool
}

// newTable creates a table printer with the given header labels.
func newTable(out io.Writer, headers ...string) *tablePrinter {
	return &tablePrinter{
		out:      out,
		isTTY:    isWriterTerminal(out),
		headers:  headers,
		showHead: true,
	}
}

// newList creates a no-header two-column printer for key/value output.
func newList(out io.Writer) *tablePrinter {
	return &tablePrinter{
		out:   out,
		isTTY: isWriterTerminal(out),
	}
}

func (t *tablePrinter) AddRow(vals ...any) {
	row := make([]string, len(vals))
	for i, v := range vals {
		row[i] = fmt.Sprint(v)
	}
	t.rows = append(t.rows, row)
}

func (t *tablePrinter) Print() {
	numCols := 0
	if t.showHead {
		numCols = len(t.headers)
	}
	for _, r := range t.rows {
		if len(r) > numCols {
			numCols = len(r)
		}
	}
	if numCols == 0 {
		return
	}

	widths := make([]int, numCols)
	if t.showHead {
		for i, h := range t.headers {
			if w := runewidth.StringWidth(h); w > widths[i] {
				widths[i] = w
			}
		}
	}
	for _, r := range t.rows {
		for i, c := range r {
			if w := runewidth.StringWidth(c); w > widths[i] {
				widths[i] = w
			}
		}
	}

	if t.showHead && len(t.headers) > 0 {
		t.writeRow(t.headers, widths, t.isTTY)
	}
	for _, r := range t.rows {
		t.writeRow(r, widths, false)
	}
}

// writeRow prints a single row. If underline is true, each non-empty cell is
// wrapped in the ANSI underline SGR so the header text itself is underlined
// without drawing a separator row.
func (t *tablePrinter) writeRow(row []string, widths []int, underline bool) {
	const gap = 2
	last := len(widths) - 1
	used := 0
	for i := 0; i < len(widths); i++ {
		var cell string
		if i < len(row) {
			cell = row[i]
		}
		pad := widths[i] - runewidth.StringWidth(cell)
		if pad < 0 {
			pad = 0
		}
		if i == last {
			trailing := 0
			if underline {
				cellW := runewidth.StringWidth(cell)
				if tw := t.termWidth(); tw > used+cellW {
					trailing = tw - used - cellW
				}
			}
			fmt.Fprint(t.out, decorate(cell+strings.Repeat(" ", trailing), underline))
		} else {
			padded := cell + strings.Repeat(" ", pad+gap)
			fmt.Fprint(t.out, decorate(padded, underline))
			used += widths[i] + gap
		}
	}
	fmt.Fprintln(t.out)
}

// termWidth returns the terminal width for the output writer, or 0 when
// unavailable (non-TTY or the fd cannot be queried).
func (t *tablePrinter) termWidth() int {
	if !t.isTTY {
		return 0
	}
	f, ok := t.out.(*os.File)
	if !ok {
		return 0
	}
	w, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0
	}
	return w
}

func decorate(s string, underline bool) string {
	if !underline || s == "" {
		return s
	}
	return "\x1b[4;32m" + s + "\x1b[0m"
}

func isWriterTerminal(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	return isTerminal(f)
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
