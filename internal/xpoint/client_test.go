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
