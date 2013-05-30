// Test me with "aet ./samples/simple/myapp"
package myapp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tu "github.com/crhym3/aegot/testutils"
)

func TestGetOk(t *testing.T) {
	const itemId = "valid-id"

	r, deleteContext := tu.NewTestRequest("GET", "/"+itemId, nil)
	defer deleteContext()
	w := httptest.NewRecorder()

	get(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	body := string(w.Body.Bytes())
	if body != itemId {
		t.Errorf("Expected %q, got %q", itemId, body)
	}
}

func TestGetErrors(t *testing.T) {
	tt := []*struct {
		path string
		code int
	}{
		{"/does-not-exist", 404},
		{"/error", 500},
	}
	for _, ti := range tt {
		r, deleteContext := tu.NewTestRequest("GET", ti.path, nil)
		defer deleteContext()
		w := httptest.NewRecorder()

		get(w, r)

		if w.Code != ti.code {
			t.Errorf("Expected %d, got %d", ti.code, w.Code)
		}
	}
}
