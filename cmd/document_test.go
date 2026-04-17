package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSearchBody_Empty(t *testing.T) {
	b, err := loadSearchBody("")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if b != nil {
		t.Errorf("body = %q, want nil", string(b))
	}
}

func TestLoadSearchBody_InlineObject(t *testing.T) {
	b, err := loadSearchBody(`{"title":"x"}`)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if string(b) != `{"title":"x"}` {
		t.Errorf("body = %s", string(b))
	}
}

func TestLoadSearchBody_InlineArray(t *testing.T) {
	b, err := loadSearchBody(`[1,2,3]`)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if string(b) != `[1,2,3]` {
		t.Errorf("body = %s", string(b))
	}
}

func TestLoadSearchBody_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	content := `{"form_name":"経費"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	b, err := loadSearchBody(path)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if string(b) != content {
		t.Errorf("body = %q", string(b))
	}
}

func TestLoadSearchBody_FileMissing(t *testing.T) {
	_, err := loadSearchBody(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil || !strings.Contains(err.Error(), "read --body file") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadSearchBody_InvalidJSON(t *testing.T) {
	_, err := loadSearchBody(`{not json}`)
	if err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadSearchBody_Stdin(t *testing.T) {
	orig := os.Stdin
	t.Cleanup(func() { os.Stdin = orig })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = r

	want := `{"from":"stdin"}`
	done := make(chan error, 1)
	go func() {
		_, werr := io.WriteString(w, want)
		_ = w.Close()
		done <- werr
	}()

	b, err := loadSearchBody("-")
	if werr := <-done; werr != nil {
		t.Fatalf("write to stdin: %v", werr)
	}
	if err != nil {
		t.Fatalf("loadSearchBody: %v", err)
	}
	if string(b) != want {
		t.Errorf("body = %q", string(b))
	}
}
