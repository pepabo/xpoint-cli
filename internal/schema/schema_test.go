package schema

import (
	"strings"
	"testing"
)

func TestAliases_Sorted(t *testing.T) {
	got := Aliases()
	want := []string{"approval.list", "document.search", "form.list"}
	if len(got) != len(want) {
		t.Fatalf("aliases = %v", got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("aliases[%d] = %q, want %q", i, got[i], w)
		}
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
