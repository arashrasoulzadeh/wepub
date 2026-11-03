package api

import (
	"encoding/json"
	"net/http"

	"github.com/arashrasoulzadeh/wepub/server/internal/auth"
)

const APIVersion = 1

func NewRouter(token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]bool{"ok": true})
		w.Write(b)
	})
	v1 := http.NewServeMux()
	v1.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]int{"api": APIVersion})
		w.Write(b)
	})
	v1.Handle("/auth/check", auth.Bearer(token)(http.HandlerFunc(auth.CheckHandler)))
	mux.Handle("/v1/", auth.Bearer(token)(http.StripPrefix("/v1", v1)))
	// version is public per S1.1 (except /healthz and /v1/version)
	mux.Handle("/v1/version", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]int{"api": APIVersion})
		w.Write(b)
	}))
	return mux
}
