package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
)

type fakeFormLister struct {
	res    *xpoint.FormsListResponse
	err    error
	called int
}

func (f *fakeFormLister) ListAvailableForms(_ context.Context) (*xpoint.FormsListResponse, error) {
	f.called++
	return f.res, f.err
}

func TestResolveFormID_Numeric(t *testing.T) {
	// nil lister: should not be called when arg is numeric.
	id, err := resolveFormID(context.Background(), nil, "412")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if id != 412 {
		t.Errorf("id = %d, want 412", id)
	}
}

func TestResolveFormID_Code(t *testing.T) {
	lister := &fakeFormLister{res: &xpoint.FormsListResponse{
		FormGroup: []xpoint.FormGroup{{
			ID: 1, Name: "g",
			Form: []xpoint.Form{
				{ID: 1, Code: "foo", Name: "Foo"},
				{ID: 412, Code: "TORIHIKISAKI_a", Name: "取引先審査"},
			},
		}},
	}}
	id, err := resolveFormID(context.Background(), lister, "TORIHIKISAKI_a")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if id != 412 {
		t.Errorf("id = %d, want 412", id)
	}
	if lister.called != 1 {
		t.Errorf("lister called %d times", lister.called)
	}
}

func TestResolveFormID_CodeNotFound(t *testing.T) {
	lister := &fakeFormLister{res: &xpoint.FormsListResponse{}}
	_, err := resolveFormID(context.Background(), lister, "missing_code")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("err = %v, want contains 'not found'", err)
	}
}

func TestResolveFormID_ListError(t *testing.T) {
	lister := &fakeFormLister{err: errors.New("boom")}
	_, err := resolveFormID(context.Background(), lister, "any_code")
	if err == nil || !strings.Contains(err.Error(), "resolve form code") {
		t.Errorf("err = %v, want 'resolve form code'", err)
	}
}
