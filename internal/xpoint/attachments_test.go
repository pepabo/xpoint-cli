package xpoint

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListAttachments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/attachments/999" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"attachments":[{"content_type":"text/csv","seq":1,"name":"a.csv","size":684,"remarks":"r1"}]}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	res, err := c.ListAttachments(context.Background(), 999)
	if err != nil {
		t.Fatalf("ListAttachments: %v", err)
	}
	if len(res.Attachments) != 1 {
		t.Fatalf("attachments = %+v", res.Attachments)
	}
	a := res.Attachments[0]
	if a.Seq != 1 || a.Name != "a.csv" || a.Size != 684 || a.Remarks != "r1" {
		t.Errorf("attachment = %+v", a)
	}
}

func TestGetAttachment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/attachments/42/3" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="report.csv"; filename*=UTF-8''%E7%B5%8C%E8%B2%BB.csv`)
		_, _ = w.Write([]byte("col1,col2\n1,2"))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	name, data, err := c.GetAttachment(context.Background(), 42, 3)
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if name != "経費.csv" {
		t.Errorf("filename = %q, want 経費.csv", name)
	}
	if !strings.HasPrefix(string(data), "col1,col2") {
		t.Errorf("data = %q", string(data))
	}
}

func TestGetAttachment_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	_, _, err := c.GetAttachment(context.Background(), 1, 1)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("err = %v", err)
	}
}

// readMultipart returns the form fields (including file parts' filenames/content)
// from the request body.
func readMultipart(t *testing.T, r *http.Request) (fields map[string]string, files map[string]struct {
	Filename string
	Data     []byte
}) {
	t.Helper()
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mt, "multipart/") {
		t.Fatalf("Content-Type = %q (%v)", r.Header.Get("Content-Type"), err)
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	fields = map[string]string{}
	files = map[string]struct {
		Filename string
		Data     []byte
	}{}
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("NextPart: %v", err)
		}
		data, _ := io.ReadAll(p)
		if p.FileName() != "" {
			files[p.FormName()] = struct {
				Filename string
				Data     []byte
			}{p.FileName(), data}
		} else {
			fields[p.FormName()] = string(data)
		}
	}
	return
}

func TestAddAttachment_MultipartBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/multiapi/v1/attachments/100" {
			t.Errorf("path = %s", r.URL.Path)
		}
		fields, files := readMultipart(t, r)
		if fields["user_code"] != "u001" {
			t.Errorf("user_code = %q", fields["user_code"])
		}
		if fields["file_name"] != "sample.txt" {
			t.Errorf("file_name = %q", fields["file_name"])
		}
		if fields["remarks"] != "memo" {
			t.Errorf("remarks = %q", fields["remarks"])
		}
		if fields["overwrite"] != "false" {
			t.Errorf("overwrite = %q", fields["overwrite"])
		}
		f, ok := files["file"]
		if !ok {
			t.Fatal("file part missing")
		}
		if f.Filename != "sample.txt" || string(f.Data) != "hello" {
			t.Errorf("file = %+v", f)
		}
		_, _ = w.Write([]byte(`{"docid":100,"seq":1,"message_type":0,"message":"added"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	overwrite := false
	res, err := c.AddAttachment(context.Background(), 100, AddAttachmentRequest{
		UserCode:    "u001",
		FileName:    "sample.txt",
		FileContent: []byte("hello"),
		Remarks:     "memo",
		Overwrite:   &overwrite,
	})
	if err != nil {
		t.Fatalf("AddAttachment: %v", err)
	}
	if res.DocID != 100 || res.Seq != 1 || res.Message != "added" {
		t.Errorf("res = %+v", res)
	}
}

func TestAddAttachment_Validation(t *testing.T) {
	c := NewClient("sub", Auth{AccessToken: "t"})
	if _, err := c.AddAttachment(context.Background(), 1, AddAttachmentRequest{}); err == nil {
		t.Errorf("expected validation error for empty request")
	}
}

func TestUpdateAttachment_UpdateWithFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/multiapi/v1/attachments/100/3" {
			t.Errorf("path = %s", r.URL.Path)
		}
		fields, files := readMultipart(t, r)
		if fields["user_code"] != "u001" {
			t.Errorf("user_code = %q", fields["user_code"])
		}
		if fields["delete"] != "false" {
			t.Errorf("delete = %q", fields["delete"])
		}
		if fields["file_name"] != "new.txt" {
			t.Errorf("file_name = %q", fields["file_name"])
		}
		if f, ok := files["file"]; !ok || string(f.Data) != "new" {
			t.Errorf("file part = %+v, ok=%v", f, ok)
		}
		_, _ = w.Write([]byte(`{"docid":100,"seq":3,"message_type":0,"message":"updated"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	res, err := c.UpdateAttachment(context.Background(), 100, 3, UpdateAttachmentRequest{
		UserCode:    "u001",
		FileName:    "new.txt",
		FileContent: []byte("new"),
	})
	if err != nil {
		t.Fatalf("UpdateAttachment: %v", err)
	}
	if res.Seq != 3 || res.Message != "updated" {
		t.Errorf("res = %+v", res)
	}
}

func TestUpdateAttachment_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields, files := readMultipart(t, r)
		if fields["delete"] != "true" {
			t.Errorf("delete = %q", fields["delete"])
		}
		if _, ok := files["file"]; ok {
			t.Errorf("file part should be omitted when deleting")
		}
		if fields["reason"] != "cleanup" {
			t.Errorf("reason = %q", fields["reason"])
		}
		_, _ = w.Write([]byte(`{"docid":100,"seq":3,"message_type":0,"message":"deleted"}`))
	}))
	defer srv.Close()

	c := clientForServer(srv)
	res, err := c.UpdateAttachment(context.Background(), 100, 3, UpdateAttachmentRequest{
		UserCode: "u001",
		Delete:   true,
		Reason:   "cleanup",
	})
	if err != nil {
		t.Fatalf("UpdateAttachment: %v", err)
	}
	if res.Message != "deleted" {
		t.Errorf("res = %+v", res)
	}
}

func TestUpdateAttachment_RequiresUserCode(t *testing.T) {
	c := NewClient("sub", Auth{AccessToken: "t"})
	if _, err := c.UpdateAttachment(context.Background(), 1, 1, UpdateAttachmentRequest{}); err == nil {
		t.Errorf("expected user_code validation error")
	}
}
