package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	runErr := fn()
	_ = w.Close()
	<-done
	os.Stdout = orig
	return buf.String(), runErr
}

func TestResolveOutputFormat_ExplicitFlagWins(t *testing.T) {
	if got := resolveOutputFormat("json"); got != "json" {
		t.Errorf("resolveOutputFormat(\"json\") = %q", got)
	}
	if got := resolveOutputFormat("table"); got != "table" {
		t.Errorf("resolveOutputFormat(\"table\") = %q", got)
	}
}

func TestResolveOutputFormat_NonTTYDefaultsToJSON(t *testing.T) {
	// In `go test`, stdout is not a TTY, so the default is "json".
	if got := resolveOutputFormat(""); got != "json" {
		t.Errorf("default = %q, want json", got)
	}
}

func TestRunJQ_SimpleFilter(t *testing.T) {
	input := map[string]any{
		"form_group": []any{
			map[string]any{"id": 1.0, "name": "g1"},
			map[string]any{"id": 2.0, "name": "g2"},
		},
	}
	out, err := captureStdout(t, func() error {
		return runJQ(input, ".form_group[].name")
	})
	if err != nil {
		t.Fatalf("runJQ: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2: %q", len(lines), out)
	}
	// Each line is JSON-encoded; parse to compare the string value.
	for i, want := range []string{"g1", "g2"} {
		var got string
		if err := json.Unmarshal([]byte(lines[i]), &got); err != nil {
			t.Fatalf("line[%d] not JSON: %v", i, err)
		}
		if got != want {
			t.Errorf("line[%d] = %q, want %q", i, got, want)
		}
	}
}

func TestRunJQ_InvalidExpression(t *testing.T) {
	err := runJQ(map[string]any{}, ".[")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --jq filter") {
		t.Errorf("error = %v", err)
	}
}

func TestRunJQ_RuntimeError(t *testing.T) {
	// Dividing a string triggers a runtime jq error.
	err := runJQ(map[string]any{"v": "abc"}, ".v / 2")
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
	if !strings.Contains(err.Error(), "jq error") {
		t.Errorf("error = %v", err)
	}
}

func TestRender_JSONPath(t *testing.T) {
	payload := map[string]any{"k": "v"}
	out, err := captureStdout(t, func() error {
		return render(payload, "json", "", func() error {
			t.Error("table fn should not be called when format=json")
			return nil
		})
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output not JSON: %v (%s)", err, out)
	}
	if decoded["k"] != "v" {
		t.Errorf("decoded = %v", decoded)
	}
}

func TestRender_TablePath(t *testing.T) {
	called := false
	_, err := captureStdout(t, func() error {
		return render("unused", "table", "", func() error {
			called = true
			return nil
		})
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !called {
		t.Error("table fn was not invoked")
	}
}

func TestRender_JQTakesPrecedence(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return render(map[string]any{"k": "v"}, "table", ".k", func() error {
			t.Error("table fn should not be called when --jq is set")
			return nil
		})
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.TrimSpace(out) != `"v"` {
		t.Errorf("output = %q, want \"v\"", out)
	}
}

func TestRender_UnknownFormat(t *testing.T) {
	err := render("x", "yaml", "", func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), "unknown output format") {
		t.Errorf("err = %v", err)
	}
}

func TestTablePrinter_NoAnsiWhenNotTTY(t *testing.T) {
	var buf bytes.Buffer
	w := newTable(&buf, "A", "B")
	w.AddRow(1, "foo")
	w.AddRow(22, "barbaz")
	w.Print()
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("expected no ANSI codes on non-TTY writer, got:\n%q", buf.String())
	}
	// Header labels and cell values should be present.
	for _, want := range []string{"A", "B", "foo", "barbaz"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("missing %q in output:\n%s", want, buf.String())
		}
	}
}

func TestTablePrinter_AlignmentCJK(t *testing.T) {
	var buf bytes.Buffer
	w := newTable(&buf, "A", "B")
	w.AddRow("あ", "いう")
	w.AddRow("xx", "y")
	w.Print()
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines (header + 2 rows), got:\n%s", buf.String())
	}
	// A column width = 2 (widest of "A"=1, "あ"=2, "xx"=2); gap = 2. So
	// the B column must start at display offset 4 on every row.
	skipCells := func(s string, n int) string {
		w := 0
		for i, r := range s {
			if w >= n {
				return s[i:]
			}
			w += runewidth.RuneWidth(r)
		}
		return ""
	}
	wantBCol := []string{"B", "いう", "y"}
	for i, l := range lines {
		if got := skipCells(l, 4); got != wantBCol[i] {
			t.Errorf("line %d B column = %q, want %q (full line: %q)",
				i, got, wantBCol[i], l)
		}
	}
}

func TestTablePrinter_HeaderUnderlinedOnTTY(t *testing.T) {
	f := ttyFile(t)
	defer func() { _ = f.Close() }()
	// Write directly to the TTY through newTable; we cannot read it back, so
	// instead verify the helper path by calling decorate directly.
	if got := decorate("HEAD", true); !strings.HasPrefix(got, "\x1b[4;32m") || !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("expected underlined+green header, got %q", got)
	}
	if got := decorate("HEAD", false); got != "HEAD" {
		t.Errorf("plain path changed output: %q", got)
	}
	if got := decorate("", true); got != "" {
		t.Errorf("empty cell should stay empty, got %q", got)
	}
	_ = f // already used as TTY availability check
}

func TestList_NoHeaderAndPlain(t *testing.T) {
	var buf bytes.Buffer
	w := newList(&buf)
	w.AddRow("user_code:", "abc")
	w.AddRow("display_name:", "alice")
	w.Print()
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Errorf("list should not emit ANSI on non-TTY writer:\n%s", out)
	}
	for _, want := range []string{"user_code:", "display_name:", "abc", "alice"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

// ttyFile returns an *os.File that isTerminal reports as a TTY, or skips the
// test if /dev/tty isn't available (e.g. CI).
func ttyFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("cannot open /dev/tty: %v", err)
	}
	if !isTerminal(f) {
		_ = f.Close()
		t.Skip("/dev/tty is not reported as a terminal in this env")
	}
	return f
}
