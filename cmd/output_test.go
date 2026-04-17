package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
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
