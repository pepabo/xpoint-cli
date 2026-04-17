package xpoint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthApply_AccessToken(t *testing.T) {
	auth := Auth{AccessToken: "tok123"}
	req, err := http.NewRequest(http.MethodGet, "https://example.test/path", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	auth.apply(req)

	if got := req.Header.Get("Authorization"); got != "Bearer tok123" {
		t.Errorf("Authorization header = %q, want %q", got, "Bearer tok123")
	}
	if got := req.Header.Get("X-ATLED-Generic-API-Token"); got != "" {
		t.Errorf("X-ATLED-Generic-API-Token header should be empty, got %q", got)
	}
}

func TestAuthApply_GenericAPIToken(t *testing.T) {
	auth := Auth{
		DomainCode:      "dom",
		User:            "u001",
		GenericAPIToken: "secret",
	}
	req, err := http.NewRequest(http.MethodGet, "https://example.test/path", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	auth.apply(req)

	want := base64.StdEncoding.EncodeToString([]byte("dom:u001:secret"))
	if got := req.Header.Get("X-ATLED-Generic-API-Token"); got != want {
		t.Errorf("X-ATLED-Generic-API-Token = %q, want %q", got, want)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization header should be empty, got %q", got)
	}
}

func TestApprovalsListParams_Query(t *testing.T) {
	fg, fi, st, rn, gl := 10, 20, 3, 5, 100
	hidden := true
	p := ApprovalsListParams{
		Stat:          10,
		FormGroupID:   &fg,
		FormID:        &fi,
		Step:          &st,
		RecordNo:      &rn,
		GetLine:       &gl,
		ProxyUser:     "proxyU",
		Filter:        `cr_dt between "2023-01-01" and "2023-12-31"`,
		ShowHiddenDoc: &hidden,
	}
	q := p.query()
	wants := map[string]string{
		"stat":            "10",
		"fgid":            "10",
		"fid":             "20",
		"step":            "3",
		"record_no":       "5",
		"get_line":        "100",
		"proxy_user":      "proxyU",
		"filter":          `cr_dt between "2023-01-01" and "2023-12-31"`,
		"show_hidden_doc": "true",
	}
	for k, want := range wants {
		if got := q.Get(k); got != want {
			t.Errorf("query[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestApprovalsListParams_Query_OmitsZero(t *testing.T) {
	p := ApprovalsListParams{Stat: 10}
	q := p.query()
	if got := q.Get("stat"); got != "10" {
		t.Errorf("stat = %q, want 10", got)
	}
	for _, k := range []string{"fgid", "fid", "step", "record_no", "get_line", "proxy_user", "filter", "show_hidden_doc"} {
		if q.Has(k) {
			t.Errorf("query should not contain %q, got %q", k, q.Get(k))
		}
	}
}

func TestSearchDocumentsParams_Query(t *testing.T) {
	s, o, pg := 25, 100, 2
	p := SearchDocumentsParams{Size: &s, Offset: &o, Page: &pg}
	q := p.query()
	if got := q.Get("size"); got != "25" {
		t.Errorf("size = %q", got)
	}
	if got := q.Get("offset"); got != "100" {
		t.Errorf("offset = %q", got)
	}
	if got := q.Get("page"); got != "2" {
		t.Errorf("page = %q", got)
	}
}

// clientForServer wires a Client to an httptest server.
func clientForServer(srv *httptest.Server) *Client {
	c := NewClient("unused", Auth{AccessToken: "t"})
	c.baseURL = srv.URL
	c.http = srv.Client()
	return c
}

func TestListAvailableForms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/forms" {
			t.Errorf("path = %s, want /api/v1/forms", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer t" {
			t.Errorf("Authorization = %q, want Bearer t", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"form_group":[{"id":10,"name":"g1","form":[{"id":1,"name":"f1","code":"c1"}]}]}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	got, err := c.ListAvailableForms(context.Background())
	if err != nil {
		t.Fatalf("ListAvailableForms: %v", err)
	}
	if len(got.FormGroup) != 1 || got.FormGroup[0].ID != 10 || got.FormGroup[0].Name != "g1" {
		t.Fatalf("unexpected form_group: %+v", got.FormGroup)
	}
	if len(got.FormGroup[0].Form) != 1 || got.FormGroup[0].Form[0].Code != "c1" {
		t.Fatalf("unexpected form: %+v", got.FormGroup[0].Form)
	}
}

func TestGetFormDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/forms/412" {
			t.Errorf("path = %s, want /api/v1/forms/412", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"form":{"code":"TORIHIKISAKI_a","name":"取引先審査","max_step":3,"route":[],"pages":[{"page_no":1,"fields":[{"seq":1,"fieldid":"integerfield4","fieldtype":11,"maxlength":10,"label":"品目","arraysize":0,"required":true,"unique":false}]}]}}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	got, err := c.GetFormDetail(context.Background(), 412)
	if err != nil {
		t.Fatalf("GetFormDetail: %v", err)
	}
	if got.Form.Code != "TORIHIKISAKI_a" {
		t.Errorf("code = %q", got.Form.Code)
	}
	if got.Form.MaxStep != 3 {
		t.Errorf("max_step = %d", got.Form.MaxStep)
	}
	if len(got.Form.Pages) != 1 || len(got.Form.Pages[0].Fields) != 1 {
		t.Fatalf("pages = %+v", got.Form.Pages)
	}
	f := got.Form.Pages[0].Fields[0]
	if f.FieldID != "integerfield4" || f.FieldType != 11 || !f.Required {
		t.Errorf("field = %+v", f)
	}
}

func TestListApprovals_QueryAndDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/approvals" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("stat"); got != "10" {
			t.Errorf("stat = %q", got)
		}
		if got := r.URL.Query().Get("get_line"); got != "50" {
			t.Errorf("get_line = %q", got)
		}
		_, _ = w.Write([]byte(`{"total_count":1,"approval_list":[{"docid":100,"attachment":true,"comment":false,"title1":"t1","form_name":"fn","apply_user":"佐藤","approval_user":["加藤"]}]}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	gl := 50
	res, err := c.ListApprovals(context.Background(), ApprovalsListParams{Stat: 10, GetLine: &gl})
	if err != nil {
		t.Fatalf("ListApprovals: %v", err)
	}
	if res.TotalCount != 1 || len(res.ApprovalList) != 1 {
		t.Fatalf("unexpected response: %+v", res)
	}
	a := res.ApprovalList[0]
	if a.DocID != 100 || a.Title1 != "t1" || !a.Attachment || a.ApplyUser != "佐藤" || len(a.ApprovalUser) != 1 {
		t.Errorf("unexpected approval: %+v", a)
	}
}

func TestSearchDocuments_PostBodyAndQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/v1/search/documents" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("size"); got != "3" {
			t.Errorf("size = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, string(body))
		}
		if decoded["title"] != "経費" {
			t.Errorf("body.title = %v", decoded["title"])
		}
		_, _ = w.Write([]byte(`{"total_count":2,"items":[{"docid":1,"form":{"id":10,"code":"c","name":"f"},"writer":"w","title1":"t"},{"docid":2,"form":{"id":10,"code":"c","name":"f"},"writer":"w2","title1":"t2"}]}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	size := 3
	res, err := c.SearchDocuments(context.Background(),
		SearchDocumentsParams{Size: &size},
		json.RawMessage(`{"title":"経費"}`),
	)
	if err != nil {
		t.Fatalf("SearchDocuments: %v", err)
	}
	if res.TotalCount != 2 || len(res.Items) != 2 || res.Items[0].DocID != 1 {
		t.Fatalf("unexpected response: %+v", res)
	}
}

func TestCreateDocument_PostsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/documents" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, string(body))
		}
		if decoded["route_code"] != "r1" {
			t.Errorf("body.route_code = %v", decoded["route_code"])
		}
		if decoded["form_code"] != "f1" {
			t.Errorf("body.form_code = %v", decoded["form_code"])
		}
		_, _ = w.Write([]byte(`{"docid":999,"message_type":3,"message":"書類が提出されました (ID = 999)"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	res, err := c.CreateDocument(context.Background(), json.RawMessage(`{"form_code":"f1","route_code":"r1","datas":[]}`))
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if res.DocID != 999 || res.MessageType != 3 || res.Message == "" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestCreateDocument_RequiresBody(t *testing.T) {
	c := NewClient("unused", Auth{AccessToken: "t"})
	_, err := c.CreateDocument(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "body is required") {
		t.Errorf("err = %v", err)
	}
}

func TestSearchDocuments_DefaultBodyIsEmptyObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(body)) != `{}` {
			t.Errorf("default body = %q, want {}", string(body))
		}
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	if _, err := c.SearchDocuments(context.Background(), SearchDocumentsParams{}, nil); err != nil {
		t.Fatalf("SearchDocuments: %v", err)
	}
}

func TestDo_ErrorResponseSurfacesStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid stat"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	_, err := c.ListApprovals(context.Background(), ApprovalsListParams{Stat: 99})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "invalid stat") {
		t.Errorf("error should mention status and body, got: %v", err)
	}
}

func TestDo_AppliesAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-ATLED-Generic-API-Token"); got == "" {
			t.Error("X-ATLED-Generic-API-Token should be set")
		}
		_, _ = w.Write([]byte(`{"form_group":[]}`))
	}))
	defer srv.Close()

	c := NewClient("unused", Auth{DomainCode: "d", User: "u", GenericAPIToken: "t"})
	c.baseURL = srv.URL
	c.http = srv.Client()
	if _, err := c.ListAvailableForms(context.Background()); err != nil {
		t.Fatalf("ListAvailableForms: %v", err)
	}
}

func TestDocumentURL(t *testing.T) {
	c := NewClient("acme", Auth{AccessToken: "t"})
	got := c.DocumentURL(266248)
	want := "https://acme.atledcloud.jp/xpoint/form.do?act=view&docid=266248"
	if got != want {
		t.Errorf("DocumentURL = %q, want %q", got, want)
	}
	if c.DocumentURL(0) != "" {
		t.Errorf("DocumentURL(0) should be empty")
	}
}

func TestGetSelfInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/scim/v2/acme/Me" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/scim+json" {
			t.Errorf("Accept = %q, want application/scim+json", got)
		}
		w.Header().Set("Content-Type", "application/scim+json")
		_, _ = w.Write([]byte(`{"id":"100","userName":"u001","displayName":"田中","urn:atled:scim:schemas:1.0:User":{"userCode":"326"}}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	info, err := c.GetSelfInfo(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetSelfInfo: %v", err)
	}
	if info.UserName != "u001" || info.ID != "100" || info.DisplayName != "田中" {
		t.Errorf("info = %+v", info)
	}
	if info.AtledExt.UserCode != "326" {
		t.Errorf("AtledExt.UserCode = %q, want 326", info.AtledExt.UserCode)
	}
}

func TestGetSelfInfo_RequiresDomainCode(t *testing.T) {
	c := NewClient("unused", Auth{AccessToken: "t"})
	_, err := c.GetSelfInfo(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "domain code is required") {
		t.Errorf("err = %v", err)
	}
}

func TestListAvailableQueries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/query/" {
			t.Errorf("path = %s, want /api/v1/query/", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query_groups":[{"query_group_id":1,"query_group_name":"g1","queries":[{"query_id":100,"query_code":"q01","query_name":"n1","query_type":"list","query_type_name":"一覧","remarks":"","form_count":1,"fid":200,"form_name":"f1"}]}]}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	got, err := c.ListAvailableQueries(context.Background())
	if err != nil {
		t.Fatalf("ListAvailableQueries: %v", err)
	}
	if len(got.QueryGroups) != 1 || got.QueryGroups[0].QueryGroupID != 1 {
		t.Fatalf("unexpected query_groups: %+v", got.QueryGroups)
	}
	q := got.QueryGroups[0].Queries[0]
	if q.QueryCode != "q01" || q.QueryType != "list" || q.FID != 200 {
		t.Errorf("unexpected query: %+v", q)
	}
}

func TestGetQueryParams_Query(t *testing.T) {
	rows, offset := 100, 50
	p := GetQueryParams{ExecFlag: true, Rows: &rows, Offset: &offset}
	q := p.query()
	if got := q.Get("exec_flg"); got != "true" {
		t.Errorf("exec_flg = %q", got)
	}
	if got := q.Get("rows"); got != "100" {
		t.Errorf("rows = %q", got)
	}
	if got := q.Get("offset"); got != "50" {
		t.Errorf("offset = %q", got)
	}
}

func TestGetQueryParams_Query_OmitsZeroAndFalse(t *testing.T) {
	q := GetQueryParams{}.query()
	for _, k := range []string{"exec_flg", "rows", "offset"} {
		if q.Has(k) {
			t.Errorf("query should not contain %q, got %q", k, q.Get(k))
		}
	}
}

func TestGetQuery_EscapesCodeAndDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v1/query/q%20%2F01" {
			t.Errorf("path = %s", r.URL.EscapedPath())
		}
		if got := r.URL.Query().Get("exec_flg"); got != "true" {
			t.Errorf("exec_flg = %q", got)
		}
		_, _ = w.Write([]byte(`{"query":{"query_code":"q /01"},"exec_result":{"data":[]}}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	raw, err := c.GetQuery(context.Background(), "q /01", GetQueryParams{ExecFlag: true})
	if err != nil {
		t.Fatalf("GetQuery: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("raw not JSON: %v", err)
	}
	if _, ok := decoded["exec_result"]; !ok {
		t.Errorf("exec_result missing: %s", string(raw))
	}
}

func TestGetDocumentStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/documents/999/status" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Has("history") {
			t.Errorf("history query should be omitted, got %q", r.URL.Query().Get("history"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"document":{"docid":999,"status":{"code":1,"name":"承認中"}}}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	raw, err := c.GetDocumentStatus(context.Background(), 999, false)
	if err != nil {
		t.Fatalf("GetDocumentStatus: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("raw not JSON: %v (%s)", err, string(raw))
	}
	doc, _ := decoded["document"].(map[string]any)
	if doc["docid"].(float64) != 999 {
		t.Errorf("docid = %v", doc["docid"])
	}
}

func TestGetDocumentStatus_WithHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("history"); got != "true" {
			t.Errorf("history = %q, want true", got)
		}
		_, _ = w.Write([]byte(`{"document":{"docid":1}}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	if _, err := c.GetDocumentStatus(context.Background(), 1, true); err != nil {
		t.Fatalf("GetDocumentStatus: %v", err)
	}
}

func TestDownloadPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/documents/42/pdf" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="fallback.pdf"; filename*=UTF-8''%E7%B5%8C%E8%B2%BB.pdf`)
		_, _ = w.Write([]byte("%PDF-1.4 body"))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	name, data, err := c.DownloadPDF(context.Background(), 42)
	if err != nil {
		t.Fatalf("DownloadPDF: %v", err)
	}
	if name != "経費.pdf" {
		t.Errorf("filename = %q, want 経費.pdf", name)
	}
	if !strings.HasPrefix(string(data), "%PDF-1.4") {
		t.Errorf("data = %q", string(data))
	}
}

func TestDownloadPDF_PlainFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
		_, _ = w.Write([]byte("pdf"))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	name, _, err := c.DownloadPDF(context.Background(), 1)
	if err != nil {
		t.Fatalf("DownloadPDF: %v", err)
	}
	if name != "report.pdf" {
		t.Errorf("filename = %q", name)
	}
}

func TestDownloadPDF_NoContentDisposition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pdf"))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	name, data, err := c.DownloadPDF(context.Background(), 1)
	if err != nil {
		t.Fatalf("DownloadPDF: %v", err)
	}
	if name != "" {
		t.Errorf("filename = %q, want empty", name)
	}
	if string(data) != "pdf" {
		t.Errorf("data = %q", string(data))
	}
}

func TestDownloadPDF_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	_, _, err := c.DownloadPDF(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("err = %v", err)
	}
}

func TestAddComment_PostsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/documents/42/comments" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, string(body))
		}
		if decoded["content"] != "hello" {
			t.Errorf("content = %v", decoded["content"])
		}
		if decoded["attentionflg"].(float64) != 1 {
			t.Errorf("attentionflg = %v", decoded["attentionflg"])
		}
		_, _ = w.Write([]byte(`{"docid":42,"seq":3,"message_type":3,"message":"コメントが追加されました"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	res, err := c.AddComment(context.Background(), 42, AddCommentRequest{Content: "hello", AttentionFlg: 1})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if res.DocID != 42 || res.Seq != 3 {
		t.Errorf("res = %+v", res)
	}
}

func TestGetComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/documents/999/comments" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"docid":999,"comment_list":[{"seqno":"1","attentionflg":true,"content":"c1","writername":"田中","writedate":"2023/01/01 12:00:00"}]}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	res, err := c.GetComments(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}
	if res.DocID != 999 || len(res.CommentList) != 1 {
		t.Fatalf("res = %+v", res)
	}
	cm := res.CommentList[0]
	if cm.SeqNo != "1" || !cm.AttentionFlg || cm.Content != "c1" || cm.WriterName != "田中" {
		t.Errorf("comment = %+v", cm)
	}
}

func TestUpdateComment_PatchesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/v1/documents/100/comments/2" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("body not JSON: %v (%s)", err, string(body))
		}
		if decoded["content"] != "updated" {
			t.Errorf("content = %v", decoded["content"])
		}
		if _, ok := decoded["attentionflg"]; ok {
			t.Errorf("attentionflg should be omitted, got %v", decoded["attentionflg"])
		}
		_, _ = w.Write([]byte(`{"docid":100,"seq":2,"message_type":3,"message":"コメントが更新されました"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	content := "updated"
	res, err := c.UpdateComment(context.Background(), 100, 2, UpdateCommentRequest{Content: &content})
	if err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	if res.DocID != 100 || res.Seq != 2 {
		t.Errorf("res = %+v", res)
	}
}

func TestDeleteComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/documents/100/comments/5" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"docid":100,"seq":5,"message_type":3,"message":"コメントが削除されました"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	res, err := c.DeleteComment(context.Background(), 100, 5)
	if err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
	if res.DocID != 100 || res.Seq != 5 {
		t.Errorf("res = %+v", res)
	}
}

func TestNewClient_BaseURL(t *testing.T) {
	c := NewClient("acme", Auth{AccessToken: "t"})
	if !strings.HasPrefix(c.baseURL, "https://acme.atledcloud.jp/xpoint") {
		t.Errorf("baseURL = %q", c.baseURL)
	}
	// Ensure url.Parse does not choke.
	if _, err := url.Parse(c.baseURL); err != nil {
		t.Errorf("baseURL not parseable: %v", err)
	}
}
