package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := NewRouter("secret", nil, t.TempDir())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if rr.Body.String() != `{"ok":true}` {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestVersion(t *testing.T) {
	h := NewRouter("secret", nil, t.TempDir())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if rr.Body.String() != `{"api":1}` {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestUnversioned404(t *testing.T) {
	h := NewRouter("secret", nil, t.TempDir())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}
