package server

import (
	"context"
	"net/http"
	"time"

	"github.com/arashrasoulzadeh/wepub/server/internal/api"
)

type Config struct {
	Addr  string
	Token string
}

func Run(ctx context.Context, cfg Config) error {
	srv := &http.Server{Addr: cfg.Addr, Handler: api.NewRouter(cfg.Token)}
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
