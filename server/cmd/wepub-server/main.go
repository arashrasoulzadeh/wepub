package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/arashrasoulzadeh/wepub/server/internal/backup"
	"github.com/arashrasoulzadeh/wepub/server/internal/server"
	"github.com/arashrasoulzadeh/wepub/server/internal/store"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "backup":
			db, _ := store.Open(os.Args[2])
			defer db.Close()
			if err := backup.SQLiteBackup(db, os.Args[3]); err != nil {
				fmt.Println(err); os.Exit(1)
			}
			fmt.Println("ok")
			return
		case "restore":
			// minimal: copy file
			src, dst := os.Args[3], os.Args[5]
			data, _ := os.ReadFile(src)
			if _, err := os.Stat(dst); err == nil {
				fmt.Println("target not empty"); os.Exit(1)
			}
			os.WriteFile(dst, data, 0644)
			fmt.Println("ok")
			return
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dd := os.Getenv("WEPUB_DATA_DIR")
	if dd == "" { dd = "./data" }
	cfg := server.Config{Addr: ":8080", Token: os.Getenv("WEPUB_TOKEN"), DataDir: dd}
	if err := server.Run(ctx, cfg); err != nil {
		os.Exit(1)
	}
	_ = sql.Open
}
