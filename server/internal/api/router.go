package api

import (
	"encoding/json"
	"net/http"
)

const APIVersion = 1

func NewRouter() http.Handler {
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
	mux.Handle("/v1/", http.StripPrefix("/v1", v1))
	return mux
}
