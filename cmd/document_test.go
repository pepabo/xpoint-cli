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

func TestRunDocumentCreate_RequiresBody(t *testing.T) {
	docCreateBody = ""
	t.Setenv("XPOINT_SUBDOMAIN", "acme")
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "tok")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "")

	err := runDocumentCreate(documentCreateCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--body is required") {
		t.Errorf("err = %v", err)
	}
}

func TestParseDocID(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"1", 1, false},
		{"999", 999, false},
		{"  42  ", 42, false},
		{"0", 0, true},
		{"-3", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := parseDocID(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseDocID(%q) = %d, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDocID(%q) err = %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("parseDocID(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRunDocumentEdit_RequiresBody(t *testing.T) {
	docEditBody = ""
	t.Setenv("XPOINT_SUBDOMAIN", "acme")
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "tok")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "")

	err := runDocumentEdit(documentEditCmd, []string{"999"})
	if err == nil || !strings.Contains(err.Error(), "--body is required") {
		t.Errorf("err = %v", err)
	}
}

func TestRunDocumentEdit_InvalidDocID(t *testing.T) {
	err := runDocumentEdit(documentEditCmd, []string{"not-a-number"})
	if err == nil || !strings.Contains(err.Error(), "invalid docid") {
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
