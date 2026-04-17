package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetSearchFlags returns the package-level docSearch* variables to their
// zero values so each test starts from a clean slate.
func resetSearchFlags(t *testing.T) {
	t.Helper()
	docSearchBody = ""
	docSearchTitle = ""
	docSearchFormName = ""
	docSearchFormID = 0
	docSearchFGID = 0
	docSearchWriters = nil
	docSearchGroups = nil
	docSearchMe = false
	docSearchSince = ""
	docSearchUntil = ""
	flagUser = ""
	t.Cleanup(func() {
		docSearchBody = ""
		docSearchTitle = ""
		docSearchFormName = ""
		docSearchFormID = 0
		docSearchFGID = 0
		docSearchWriters = nil
		docSearchGroups = nil
		docSearchMe = false
		docSearchSince = ""
		docSearchUntil = ""
		flagUser = ""
	})
}

func TestBuildSearchBody_TitleAndForm(t *testing.T) {
	resetSearchFlags(t)
	docSearchTitle = "経費"
	docSearchFormName = "稟議"
	docSearchFormID = 42
	docSearchFGID = 7

	raw, err := buildSearchBodyFromFlags()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["title"] != "経費" {
		t.Errorf("title = %v", got["title"])
	}
	if got["form_name"] != "稟議" {
		t.Errorf("form_name = %v", got["form_name"])
	}
	if got["fid"] != float64(42) {
		t.Errorf("fid = %v", got["fid"])
	}
	if got["fgid"] != float64(7) {
		t.Errorf("fgid = %v", got["fgid"])
	}
}

func TestBuildSearchBody_Writers(t *testing.T) {
	resetSearchFlags(t)
	docSearchWriters = []string{"u1", "u2"}
	docSearchGroups = []string{"g1"}

	raw, err := buildSearchBodyFromFlags()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var got struct {
		WriterList []struct {
			Type string `json:"type"`
			Code string `json:"code"`
		} `json:"writer_list"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.WriterList) != 3 {
		t.Fatalf("writer_list = %+v", got.WriterList)
	}
	if got.WriterList[0] != (struct {
		Type string `json:"type"`
		Code string `json:"code"`
	}{"user", "u1"}) {
		t.Errorf("writer[0] = %+v", got.WriterList[0])
	}
	if got.WriterList[2].Type != "group" || got.WriterList[2].Code != "g1" {
		t.Errorf("writer[2] = %+v", got.WriterList[2])
	}
}

func TestBuildSearchBody_MeRequiresUser(t *testing.T) {
	resetSearchFlags(t)
	docSearchMe = true

	_, err := buildSearchBodyFromFlags()
	if err == nil || !strings.Contains(err.Error(), "--me requires") {
		t.Errorf("err = %v", err)
	}
}

func TestBuildSearchBody_MeWithFlagUser(t *testing.T) {
	resetSearchFlags(t)
	docSearchMe = true
	flagUser = "alice"

	raw, err := buildSearchBodyFromFlags()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"alice"`) {
		t.Errorf("body missing current user: %s", string(raw))
	}
}

func TestBuildSearchBody_MeFromEnv(t *testing.T) {
	resetSearchFlags(t)
	docSearchMe = true
	t.Setenv("XPOINT_USER", "bob")

	raw, err := buildSearchBodyFromFlags()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"bob"`) {
		t.Errorf("body missing env user: %s", string(raw))
	}
}

func TestBuildSearchBody_SinceUntil(t *testing.T) {
	resetSearchFlags(t)
	docSearchSince = "2024-01-15"
	docSearchUntil = "2024-12-31"

	raw, err := buildSearchBodyFromFlags()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["date_type"] != "cr_dt" {
		t.Errorf("date_type = %v", got["date_type"])
	}
	if got["dt_cond_type"] != "1" {
		t.Errorf("dt_cond_type = %v", got["dt_cond_type"])
	}
	if got["lower_year"] != float64(2024) || got["lower_month"] != float64(1) || got["lower_day"] != float64(15) {
		t.Errorf("lower_* = %v / %v / %v", got["lower_year"], got["lower_month"], got["lower_day"])
	}
	if got["upper_year"] != float64(2024) || got["upper_month"] != float64(12) || got["upper_day"] != float64(31) {
		t.Errorf("upper_* = %v / %v / %v", got["upper_year"], got["upper_month"], got["upper_day"])
	}
}

func TestBuildSearchBody_InvalidSince(t *testing.T) {
	resetSearchFlags(t)
	docSearchSince = "2024/01/15"

	_, err := buildSearchBodyFromFlags()
	if err == nil || !strings.Contains(err.Error(), "--since") {
		t.Errorf("err = %v", err)
	}
}

func TestRunDocumentSearch_BodyAndFilterConflict(t *testing.T) {
	resetSearchFlags(t)
	t.Setenv("XPOINT_SUBDOMAIN", "acme")
	t.Setenv("XPOINT_API_ACCESS_TOKEN", "tok")
	t.Setenv("XPOINT_GENERIC_API_TOKEN", "")

	docSearchBody = `{"title":"x"}`
	docSearchTitle = "y"

	err := runDocumentSearch(documentSearchCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Errorf("err = %v", err)
	}
}

func TestResolveDownloadPath_DefaultUsesServerName(t *testing.T) {
	got := resolveDownloadPath("", "経費.pdf", 42)
	if got != "経費.pdf" {
		t.Errorf("got = %q, want 経費.pdf", got)
	}
}

func TestResolveDownloadPath_DefaultFallbackWhenEmpty(t *testing.T) {
	got := resolveDownloadPath("", "", 42)
	if got != "42.pdf" {
		t.Errorf("got = %q, want 42.pdf", got)
	}
}

func TestResolveDownloadPath_StripsPathTraversal(t *testing.T) {
	got := resolveDownloadPath("", "../../etc/passwd", 99)
	if got != "passwd" {
		t.Errorf("got = %q, want passwd", got)
	}
}

func TestResolveDownloadPath_ExplicitFile(t *testing.T) {
	got := resolveDownloadPath("out.pdf", "server.pdf", 1)
	if got != "out.pdf" {
		t.Errorf("got = %q, want out.pdf", got)
	}
}

func TestResolveDownloadPath_DirectorySuffix(t *testing.T) {
	got := resolveDownloadPath("sub/", "doc.pdf", 1)
	if got != filepath.Join("sub", "doc.pdf") {
		t.Errorf("got = %q", got)
	}
}

func TestResolveDownloadPath_ExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	got := resolveDownloadPath(dir, "server.pdf", 1)
	want := filepath.Join(dir, "server.pdf")
	if got != want {
		t.Errorf("got = %q, want %q", got, want)
	}
}

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
