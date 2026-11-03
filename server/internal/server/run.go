package server

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/arashrasoulzadeh/wepub/server/internal/api"
	"github.com/arashrasoulzadeh/wepub/server/internal/store"
)

type Config struct {
	Addr    string
	Token   string
	DataDir string
}

func Run(ctx context.Context, cfg Config) error {
	var db *sql.DB
	if cfg.DataDir != "" {
		d, err := store.Open(cfg.DataDir + "/wepub.db")
		if err != nil {
			return err
		}
		if err := store.Migrate(d); err != nil {
			d.Close()
			return err
		}
		db = d
	}
	srv := &http.Server{Addr: cfg.Addr, Handler: api.NewRouter(cfg.Token, db)}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	case err := <-errCh:
		return err
	}
}
