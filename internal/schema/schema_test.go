package schema

import (
	"strings"
	"testing"
)

func TestAliases_Sorted(t *testing.T) {
	got := Aliases()
	want := []string{
		"approval.hidden",
		"approval.list",
		"approval.wait",
		"document.attachment.add",
		"document.attachment.delete",
		"document.attachment.get",
		"document.attachment.list",
		"document.attachment.update",
		"document.comment.add",
		"document.comment.delete",
		"document.comment.edit",
		"document.comment.get",
		"document.create",
		"document.delete",
		"document.docview",
		"document.docview.upload",
		"document.download",
		"document.get",
		"document.openview",
		"document.search",
		"document.status",
		"document.statusview",
		"document.update",
		"form.list",
		"form.show",
		"query.exec",
		"query.list",
	}
	if len(got) != len(want) {
		t.Fatalf("aliases = %v", got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("aliases[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestLookup_DocumentCreate(t *testing.T) {
	op, err := Lookup("document.create")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "POST" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents" {
		t.Errorf("path = %v", op["path"])
	}
	body, _ := op["requestBody"].(map[string]any)
	props, _ := body["properties"].(map[string]any)
	rc, _ := props["route_code"].(map[string]any)
	if rc["required"] != true {
		t.Errorf("route_code.required = %v", rc["required"])
	}
}

func TestLookup_Unknown(t *testing.T) {
	_, err := Lookup("nope.missing")
	if err == nil || !strings.Contains(err.Error(), "unknown schema alias") {
		t.Errorf("err = %v", err)
	}
}

func TestLookup_FormList(t *testing.T) {
	op, err := Lookup("form.list")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/forms" {
		t.Errorf("path = %v", op["path"])
	}
	// form.id must be integer per our curated schema (upstream spec has a bug).
	resp, _ := op["response"].(map[string]any)
	props, _ := resp["properties"].(map[string]any)
	fg, _ := props["form_group"].(map[string]any)
	fgItems, _ := fg["items"].(map[string]any)
	fgProps, _ := fgItems["properties"].(map[string]any)
	formArr, _ := fgProps["form"].(map[string]any)
	formItems, _ := formArr["items"].(map[string]any)
	formProps, _ := formItems["properties"].(map[string]any)
	formID, _ := formProps["id"].(map[string]any)
	if formID["type"] != "integer" {
		t.Errorf("form.id type = %v, want integer", formID["type"])
	}
}

func TestLookup_ApprovalList_RequiredStat(t *testing.T) {
	op, err := Lookup("approval.list")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	params, _ := op["parameters"].([]any)
	if len(params) == 0 {
		t.Fatal("no parameters")
	}
	first, _ := params[0].(map[string]any)
	if first["name"] != "stat" || first["required"] != true {
		t.Errorf("first param = %v", first)
	}
}

func TestLookup_DocumentGet(t *testing.T) {
	op, err := Lookup("document.get")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_DocumentUpdate(t *testing.T) {
	op, err := Lookup("document.update")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "PATCH" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}" {
		t.Errorf("path = %v", op["path"])
	}
	body, _ := op["requestBody"].(map[string]any)
	props, _ := body["properties"].(map[string]any)
	rc, _ := props["route_code"].(map[string]any)
	if rc["required"] != true {
		t.Errorf("route_code.required = %v", rc["required"])
	}
}

func TestLookup_DocumentDownload(t *testing.T) {
	op, err := Lookup("document.download")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}/pdf" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_QueryList(t *testing.T) {
	op, err := Lookup("query.list")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/query/" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_QueryExec(t *testing.T) {
	op, err := Lookup("query.exec")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/query/{query_code}" {
		t.Errorf("path = %v", op["path"])
	}
	params, _ := op["parameters"].([]any)
	if len(params) < 1 {
		t.Fatalf("parameters = %v", params)
	}
	first, _ := params[0].(map[string]any)
	if first["name"] != "query_code" || first["required"] != true {
		t.Errorf("first param = %v", first)
	}
}

func TestLookup_DocumentStatus(t *testing.T) {
	op, err := Lookup("document.status")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}/status" {
		t.Errorf("path = %v", op["path"])
	}
	params, _ := op["parameters"].([]any)
	if len(params) != 2 {
		t.Fatalf("parameters = %v", params)
	}
	docid, _ := params[0].(map[string]any)
	if docid["name"] != "docid" || docid["required"] != true {
		t.Errorf("docid param = %v", docid)
	}
	history, _ := params[1].(map[string]any)
	if history["name"] != "history" || history["in"] != "query" {
		t.Errorf("history param = %v", history)
	}
}

func TestLookup_DocumentDelete(t *testing.T) {
	op, err := Lookup("document.delete")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "DELETE" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_FormShow(t *testing.T) {
	op, err := Lookup("form.show")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/forms/{fid}" {
		t.Errorf("path = %v", op["path"])
	}
	params, _ := op["parameters"].([]any)
	if len(params) != 1 {
		t.Fatalf("parameters = %v", params)
	}
	first, _ := params[0].(map[string]any)
	if first["name"] != "fid" || first["required"] != true || first["type"] != "integer" {
		t.Errorf("fid param = %v", first)
	}
}

func TestLookup_DocumentAttachmentAdd(t *testing.T) {
	op, err := Lookup("document.attachment.add")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "POST" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/multiapi/v1/attachments/{docid}" {
		t.Errorf("path = %v", op["path"])
	}
	body, _ := op["requestBody"].(map[string]any)
	if body["contentType"] != "multipart/form-data" {
		t.Errorf("contentType = %v", body["contentType"])
	}
}

func TestLookup_DocumentAttachmentList(t *testing.T) {
	op, err := Lookup("document.attachment.list")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/attachments/{docid}" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_DocumentAttachmentGet(t *testing.T) {
	op, err := Lookup("document.attachment.get")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/attachments/{docid}/{attach_seq}" {
		t.Errorf("path = %v", op["path"])
	}
	params, _ := op["parameters"].([]any)
	if len(params) != 2 {
		t.Fatalf("parameters = %v", params)
	}
}

func TestLookup_DocumentAttachmentUpdate(t *testing.T) {
	op, err := Lookup("document.attachment.update")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "PATCH" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/multiapi/v1/attachments/{docid}/{attach_seq}" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_DocumentAttachmentDelete(t *testing.T) {
	op, err := Lookup("document.attachment.delete")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "PATCH" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/multiapi/v1/attachments/{docid}/{attach_seq}" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_DocumentCommentAdd(t *testing.T) {
	op, err := Lookup("document.comment.add")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "POST" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}/comments" {
		t.Errorf("path = %v", op["path"])
	}
	body, _ := op["requestBody"].(map[string]any)
	props, _ := body["properties"].(map[string]any)
	content, _ := props["content"].(map[string]any)
	if content["required"] != true {
		t.Errorf("content.required = %v", content["required"])
	}
}

func TestLookup_DocumentCommentGet(t *testing.T) {
	op, err := Lookup("document.comment.get")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "GET" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}/comments" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_DocumentCommentEdit(t *testing.T) {
	op, err := Lookup("document.comment.edit")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "PATCH" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}/comments/{seq}" {
		t.Errorf("path = %v", op["path"])
	}
	params, _ := op["parameters"].([]any)
	if len(params) != 2 {
		t.Fatalf("parameters = %v", params)
	}
}

func TestLookup_DocumentCommentDelete(t *testing.T) {
	op, err := Lookup("document.comment.delete")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "DELETE" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/documents/{docid}/comments/{seq}" {
		t.Errorf("path = %v", op["path"])
	}
}

func TestLookup_DocumentSearch(t *testing.T) {
	op, err := Lookup("document.search")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if op["method"] != "POST" {
		t.Errorf("method = %v", op["method"])
	}
	if op["path"] != "/api/v1/search/documents" {
		t.Errorf("path = %v", op["path"])
	}
	if _, ok := op["requestBody"].(map[string]any); !ok {
		t.Errorf("requestBody missing")
	}
}
