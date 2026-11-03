package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRejectMissing(t *testing.T) {
	h := Bearer("secret")(http.HandlerFunc(CheckHandler))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestRejectWrongAcceptCorrect(t *testing.T) {
	h := Bearer("secret")(http.HandlerFunc(CheckHandler))
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer wrong")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong status %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer secret")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("correct status %d", rr.Code)
	}
}
