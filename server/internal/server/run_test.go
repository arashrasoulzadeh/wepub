package server

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRunStartsAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, Config{Addr: "127.0.0.1:0"}) }()
	// give it time to start then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not stop")
	}
	// ensure healthz reachable not required
	_ = http.StatusOK
}
