package sync

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/arashrasoulzadeh/wepub/server/internal/store"
)

type pushReq struct {
	Ops []Op `json:"ops"`
}

func NewPushHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req pushReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		now := time.Now()
		for _, op := range req.Ops {
			if err := ValidateOp(op, now); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		seqs, err := store.AppendOps(db, req.Ops)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var max int64
		for _, s := range seqs {
			if s > max {
				max = s
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"accepted": len(req.Ops), "seq_max": max})
	}
}

func NewPullHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cursor, _ := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)
		limit := 500
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		ops, next, hasMore, err := store.PullOps(db, cursor, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ops": ops, "next_cursor": next, "has_more": hasMore})
	}
}
