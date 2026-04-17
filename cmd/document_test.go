package cmd

import (
	"bytes"
	"context"
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
	clear := func() {
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
		flagDomainCode = ""
	}
	clear()
	t.Cleanup(clear)
}

func TestBuildSearchBody_TitleAndForm(t *testing.T) {
	resetSearchFlags(t)
	docSearchTitle = "経費"
	docSearchFormName = "稟議"
	docSearchFormID = 42
	docSearchFGID = 7

	raw, err := buildSearchBodyFromFlags("")
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

	raw, err := buildSearchBodyFromFlags("")
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

func TestBuildSearchBody_MeCodeAppendsWriter(t *testing.T) {
	resetSearchFlags(t)

	raw, err := buildSearchBodyFromFlags("alice")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"alice"`) {
		t.Errorf("body missing me user: %s", string(raw))
	}
}

func TestBuildSearchBody_SinceUntil(t *testing.T) {
	resetSearchFlags(t)
	docSearchSince = "2024-01-15"
	docSearchUntil = "2024-12-31"

	raw, err := buildSearchBodyFromFlags("")
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

	_, err := buildSearchBodyFromFlags("")
	if err == nil || !strings.Contains(err.Error(), "--since") {
		t.Errorf("err = %v", err)
	}
}

func TestResolveCurrentUserCode_UsesFlagUser(t *testing.T) {
	resetSearchFlags(t)
	flagUser = "alice"

	code, err := resolveCurrentUserCode(context.Background(), nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if code != "alice" {
		t.Errorf("code = %q, want alice", code)
	}
}

func TestResolveCurrentUserCode_UsesEnvUser(t *testing.T) {
	resetSearchFlags(t)
	t.Setenv("XPOINT_USER", "bob")

	code, err := resolveCurrentUserCode(context.Background(), nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if code != "bob" {
		t.Errorf("code = %q, want bob", code)
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

func TestRunDocumentOpen_InvalidDocID(t *testing.T) {
	err := runDocumentOpen(documentOpenCmd, []string{"0"})
	if err == nil || !strings.Contains(err.Error(), "invalid docid") {
		t.Errorf("err = %v", err)
	}
}

func TestRunDocumentOpen_NoBrowserPrintsURL(t *testing.T) {
	docOpenNoBrowser = true
	t.Cleanup(func() { docOpenNoBrowser = false })
	t.Setenv("XPOINT_SUBDOMAIN", "acme")

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	if err := runDocumentOpen(documentOpenCmd, []string{"266248"}); err != nil {
		t.Fatalf("runDocumentOpen: %v", err)
	}
	_ = w.Close()
	got := strings.TrimSpace(<-done)
	want := "https://acme.atledcloud.jp/xpoint/form.do?act=view&docid=266248"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunDocumentEdit_InvalidDocID(t *testing.T) {
	err := runDocumentEdit(documentEditCmd, []string{"not-a-number"})
	if err == nil || !strings.Contains(err.Error(), "invalid docid") {
		t.Errorf("err = %v", err)
	}
}

func TestParseDocumentStatus_PopulatesFields(t *testing.T) {
	raw := json.RawMessage(`{
		"document": {
			"docid": 12345,
			"title1": "経費申請 2024-01",
			"form": {"id": 99, "code": "form01", "name": "経費申請書"},
			"route": {"code": "r1", "name": "標準ルート"},
			"type": "workflow",
			"status": {"code": 1, "name": "承認中"},
			"step": {"max": 5, "current": 2},
			"current_version": 3,
			"writer": {"usercode": "u001", "username": "山田太郎", "datetime": "2024-01-15 10:00:00"},
			"lastaprv": {"usercode": "u002", "username": "佐藤花子", "datetime": "2024-01-16 11:00:00"},
			"flow_versions": [
				{
					"flow_results": [
						{"stepno": 1, "steptitle": "申請", "aprvusers": [
							{"aprv": {"usercode": "u001", "username": "山田太郎", "datetime": "2024-01-15 10:00:00"}, "statuscode": 0, "status": "申請"}
						]},
						{"stepno": 2, "steptitle": "一次承認", "aprvusers": [
							{"aprv": {"usercode": "u002", "username": "佐藤花子", "datetime": "2024-01-16 11:00:00"}, "statuscode": 1, "status": "承認"}
						]}
					]
				}
			]
		}
	}`)

	view, err := parseDocumentStatus(raw)
	if err != nil {
		t.Fatalf("parseDocumentStatus: %v", err)
	}
	d := view.Document
	if d.DocID != 12345 {
		t.Errorf("docid = %d", d.DocID)
	}
	if d.Form.Code != "form01" || d.Form.Name != "経費申請書" {
		t.Errorf("form = %+v", d.Form)
	}
	if d.Status.Code != 1 || d.Status.Name != "承認中" {
		t.Errorf("status = %+v", d.Status)
	}
	if d.Step.Current != 2 || d.Step.Max != 5 {
		t.Errorf("step = %+v", d.Step)
	}
	if len(d.FlowVersions) != 1 || len(d.FlowVersions[0].FlowResults) != 2 {
		t.Fatalf("flow_versions = %+v", d.FlowVersions)
	}
	first := d.FlowVersions[0].FlowResults[1]
	if first.StepTitle != "一次承認" || len(first.AprvUsers) != 1 {
		t.Errorf("flow[1] = %+v", first)
	}
	if first.AprvUsers[0].Aprv.UserName != "佐藤花子" {
		t.Errorf("approver = %+v", first.AprvUsers[0])
	}
}

func TestPrintDocumentStatusTable_IncludesMetaAndFlow(t *testing.T) {
	view := &documentStatusView{
		Document: documentStatusDocument{
			DocID:          12345,
			Title1:         "経費申請",
			Form:           documentStatusForm{ID: 99, Code: "form01", Name: "経費申請書"},
			Route:          documentStatusRoute{Code: "r1", Name: "標準ルート"},
			Type:           "workflow",
			Status:         documentStatusState{Code: 1, Name: "承認中"},
			Step:           documentStatusStep{Max: 5, Current: 2},
			CurrentVersion: 3,
			Writer: documentStatusUser{
				UserCode: "u001", UserName: "山田太郎",
				DateTime: "2024-01-15 10:00:00",
			},
			LastAprv: documentStatusUser{
				UserCode: "u002", UserName: "佐藤花子",
				DateTime: "2024-01-16 11:00:00",
			},
			FlowVersions: []documentStatusFlowVer{
				{FlowResults: []documentStatusFlowStep{
					{StepNo: 1, StepTitle: "申請", AprvUsers: []documentStatusAprvUser{
						{Aprv: documentStatusAprvDetail{UserCode: "u001", UserName: "山田太郎", DateTime: "2024-01-15 10:00:00"}, Status: "申請"},
					}},
					{StepNo: 2, StepTitle: "一次承認", AprvUsers: []documentStatusAprvUser{
						{Aprv: documentStatusAprvDetail{UserCode: "u002", UserName: "佐藤花子"}, Status: "未処理"},
						{Aprv: documentStatusAprvDetail{UserCode: "u003", UserName: "鈴木次郎"}, Status: "未処理"},
					}},
				}},
			},
		},
	}

	var buf bytes.Buffer
	printDocumentStatusTable(&buf, view)
	out := buf.String()

	for _, want := range []string{
		"DOCID:", "12345",
		"TITLE1:", "経費申請",
		"FORM:", "経費申請書",
		"ROUTE:", "標準ルート",
		"STATUS:", "承認中",
		"STEP:", "2/5",
		"WRITER:", "山田太郎", "2024-01-15 10:00:00",
		"LASTAPRV:", "佐藤花子",
		"承認フロー:",
		"STEP", "TITLE", "USER", "STATUS", "DATETIME",
		"申請", "一次承認",
		"山田太郎", "佐藤花子", "鈴木次郎",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
	// TYPE / VERSION should no longer be rendered.
	for _, unwant := range []string{"TYPE:", "VERSION:"} {
		if strings.Contains(out, unwant) {
			t.Errorf("unwanted label %q present in output:\n%s", unwant, out)
		}
	}
	// FORM should not display the form code (only the name).
	if strings.Contains(out, "form01") {
		t.Errorf("FORM line should not include the form code, got:\n%s", out)
	}
	// STATUS should not include the numeric status code.
	if strings.Contains(out, "(1)") {
		t.Errorf("STATUS line should not include numeric code, got:\n%s", out)
	}
	// Neither the flow table USER column nor WRITER/LASTAPRV should include
	// the parenthesized user code.
	for _, unwant := range []string{"(u001)", "(u002)", "(u003)"} {
		if strings.Contains(out, unwant) {
			t.Errorf("user code %q should not appear anywhere in output:\n%s", unwant, out)
		}
	}
	// Current step (2) should be marked with "*2"; non-current step (1)
	// gets a leading space " 1" to keep numbers column-aligned.
	if !strings.Contains(out, "*2") {
		t.Errorf("current step marker \"*2\" missing:\n%s", out)
	}
	if !strings.Contains(out, " 1") {
		t.Errorf("non-current step \" 1\" missing:\n%s", out)
	}
	// Trailing 承認完了 row should be appended.
	if !strings.Contains(out, "承認完了") {
		t.Errorf("trailing 承認完了 row missing:\n%s", out)
	}
	// The second approver row should have blank STEP/TITLE because it
	// shares the step with the first approver.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	var suzukiLine string
	for _, l := range lines {
		if strings.Contains(l, "鈴木次郎") {
			suzukiLine = l
			break
		}
	}
	if suzukiLine == "" {
		t.Fatalf("鈴木次郎 row missing in output:\n%s", out)
	}
	if strings.Contains(suzukiLine, "一次承認") || strings.Contains(suzukiLine, "*2") {
		t.Errorf("grouped row should have blank STEP/TITLE, got: %q", suzukiLine)
	}
}

func TestBuildCompletionRow_CompletedShowsLastAprv(t *testing.T) {
	d := documentStatusDocument{
		Status: documentStatusState{Code: 6, Name: "承認完了"},
		LastAprv: documentStatusUser{
			UserCode: "u9", UserName: "承認太郎",
			DateTime: "2024-02-01 09:00:00",
		},
	}
	r := buildCompletionRow(d)
	if r == nil {
		t.Fatal("completion row = nil, want populated row")
	}
	if r.user != "承認太郎" {
		t.Errorf("user = %q", r.user)
	}
	if r.status != "承認完了" {
		t.Errorf("status = %q", r.status)
	}
	if r.datetime != "2024-02-01 09:00:00" {
		t.Errorf("datetime = %q", r.datetime)
	}
	if !r.current {
		t.Errorf("current = false, want true for 承認完了")
	}
}

func TestBuildCompletionRow_NotCompletedShowsDashes(t *testing.T) {
	d := documentStatusDocument{
		Status:   documentStatusState{Code: 1, Name: "承認中"},
		LastAprv: documentStatusUser{},
	}
	r := buildCompletionRow(d)
	if r == nil {
		t.Fatal("completion row = nil")
	}
	if r.user != "-" || r.status != "-" || r.datetime != "-" {
		t.Errorf("row = %+v, want all dashes", r)
	}
	if r.current {
		t.Errorf("current = true, want false for non-completed")
	}
}

func TestFlowCurrentStepNo_SuppressedWhenCompleted(t *testing.T) {
	d := documentStatusDocument{
		Status: documentStatusState{Code: 6, Name: "承認完了"},
		Step:   documentStatusStep{Max: 3, Current: 3},
	}
	if got := flowCurrentStepNo(d); got != 0 {
		t.Errorf("completed doc: got %d, want 0", got)
	}
}

func TestFlowCurrentStepNo_UsesStepCurrent(t *testing.T) {
	d := documentStatusDocument{
		Status: documentStatusState{Code: 1, Name: "承認中"},
		Step:   documentStatusStep{Max: 3, Current: 2},
	}
	if got := flowCurrentStepNo(d); got != 2 {
		t.Errorf("pending doc: got %d, want 2", got)
	}
}

func TestPrintDocumentStatusTable_CompletedMarksFinalRow(t *testing.T) {
	view := &documentStatusView{
		Document: documentStatusDocument{
			DocID:  42,
			Form:   documentStatusForm{Code: "f", Name: "F"},
			Status: documentStatusState{Code: 6, Name: "承認完了"},
			Step:   documentStatusStep{Max: 3, Current: 3},
			LastAprv: documentStatusUser{
				UserCode: "u9", UserName: "承認太郎",
				DateTime: "2024-02-01 09:00:00",
			},
			FlowVersions: []documentStatusFlowVer{
				{FlowResults: []documentStatusFlowStep{
					{StepNo: 1, StepTitle: "申請", AprvUsers: []documentStatusAprvUser{
						{Aprv: documentStatusAprvDetail{UserCode: "u1", UserName: "A"}, Status: "申請"},
					}},
					{StepNo: 3, StepTitle: "最終承認", AprvUsers: []documentStatusAprvUser{
						{Aprv: documentStatusAprvDetail{UserCode: "u9", UserName: "承認太郎"}, Status: "承認"},
					}},
				}},
			},
		},
	}

	var buf bytes.Buffer
	printDocumentStatusTable(&buf, view)
	out := buf.String()

	if !strings.Contains(out, "*4") {
		t.Errorf("承認完了 row should carry \"*4\" marker (last+1), got:\n%s", out)
	}
	// The real last step (3) should NOT be starred when the doc is completed.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	var lastStepLine string
	for _, l := range lines {
		if strings.Contains(l, "最終承認") {
			lastStepLine = l
			break
		}
	}
	if lastStepLine == "" {
		t.Fatalf("最終承認 row missing:\n%s", out)
	}
	if strings.Contains(lastStepLine, "*3") {
		t.Errorf("real step row should not be starred when completed, got: %q", lastStepLine)
	}
	if !strings.Contains(lastStepLine, " 3") {
		t.Errorf("real step row should have \" 3\" prefix, got: %q", lastStepLine)
	}
}

func TestFormatFlowStepNo_StarAndSpacePrefix(t *testing.T) {
	if got := formatFlowStepNo(2, 2); got != "*2" {
		t.Errorf("current step: got %q, want \"*2\"", got)
	}
	if got := formatFlowStepNo(1, 2); got != " 1" {
		t.Errorf("non-current step: got %q, want \" 1\"", got)
	}
	if got := formatFlowStepNo(3, 0); got != " 3" {
		t.Errorf("currentStepNo=0: got %q, want \" 3\"", got)
	}
}

func TestPrintDocumentStatusTable_IncludesHistories(t *testing.T) {
	view := &documentStatusView{
		Document: documentStatusDocument{
			DocID:  42,
			Form:   documentStatusForm{Code: "f", Name: "F"},
			Status: documentStatusState{Code: 6, Name: "承認完了"},
			Step:   documentStatusStep{Max: 2, Current: 2},
			Histories: []documentStatusHistory{
				{Version: 1, FlowResults: []documentStatusFlowStep{
					{StepNo: 1, StepTitle: "申請", AprvUsers: []documentStatusAprvUser{
						{Aprv: documentStatusAprvDetail{UserCode: "u1", UserName: "A"}, Status: "申請"},
					}},
				}},
				{Version: 2, FlowResults: []documentStatusFlowStep{
					{StepNo: 1, StepTitle: "申請", AprvUsers: []documentStatusAprvUser{
						{Aprv: documentStatusAprvDetail{UserCode: "u1", UserName: "A"}, Status: "申請"},
					}},
				}},
			},
		},
	}

	var buf bytes.Buffer
	printDocumentStatusTable(&buf, view)
	out := buf.String()

	for _, want := range []string{
		"承認履歴 (version 1):",
		"承認履歴 (version 2):",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestParseDocumentStatus_AcceptsStringNumbers(t *testing.T) {
	// Some X-point endpoints return integer-valued fields as JSON strings.
	raw := json.RawMessage(`{
		"document": {
			"docid": "265941",
			"form": {"id": "99", "code": "f", "name": "F"},
			"status": {"code": "1", "name": "承認中"},
			"step": {"max": "3", "current": "2"},
			"current_version": "4",
			"flow_versions": [
				{"flow_results": [
					{"stepno": "1", "steptitle": "申請", "aprvusers": [
						{"aprv": {"usercode": "u1", "username": "A"}, "statuscode": "0", "status": "申請"}
					]}
				]}
			],
			"histories": [
				{"version": "1", "flow_results": []}
			]
		}
	}`)

	view, err := parseDocumentStatus(raw)
	if err != nil {
		t.Fatalf("parseDocumentStatus: %v", err)
	}
	d := view.Document
	if d.DocID != 265941 {
		t.Errorf("docid = %d", d.DocID)
	}
	if d.Status.Code != 1 {
		t.Errorf("status.code = %d", d.Status.Code)
	}
	if d.Step.Current != 2 || d.Step.Max != 3 {
		t.Errorf("step = %+v", d.Step)
	}
	if d.CurrentVersion != 4 {
		t.Errorf("current_version = %d", d.CurrentVersion)
	}
	if len(d.FlowVersions) != 1 || d.FlowVersions[0].FlowResults[0].StepNo != 1 {
		t.Errorf("flow_versions = %+v", d.FlowVersions)
	}
	if d.Histories[0].Version != 1 {
		t.Errorf("histories[0].version = %d", d.Histories[0].Version)
	}
}

func TestFlexInt_EmptyStringBecomesZero(t *testing.T) {
	var v struct {
		X flexInt `json:"x"`
	}
	if err := json.Unmarshal([]byte(`{"x":""}`), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v.X != 0 {
		t.Errorf("x = %d, want 0", v.X)
	}
}

func TestFormatRawJSONIndent_PrettyPrints(t *testing.T) {
	raw := json.RawMessage(`{"a":1,"b":[1,2]}`)
	got := string(formatRawJSONIndent(raw))
	if !strings.Contains(got, "\n  \"a\": 1") {
		t.Errorf("not indented: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("missing trailing newline: %q", got)
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
