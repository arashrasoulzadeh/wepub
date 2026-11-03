package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/arashrasoulzadeh/wepub/server/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg := server.Config{Addr: ":8080"}
	if err := server.Run(ctx, cfg); err != nil {
		os.Exit(1)
	}
}
