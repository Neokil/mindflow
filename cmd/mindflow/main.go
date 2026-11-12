package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"mindflow/internal/app"
	"mindflow/internal/store"
)

func main() {
	dir := os.Getenv("MINDFLOW_DATA")
	if dir == "" {
		dir = "data"
	}
	db, err := store.Open(filepath.Join(dir, "mindflow.sqlite"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	token := make([]byte, 24)
	if _, err = rand.Read(token); err != nil {
		log.Fatal(err)
	}
	var initialized int
	if err = db.QueryRow("SELECT initialized FROM app_settings WHERE id=1").Scan(&initialized); err != nil {
		log.Fatal(err)
	}
	bootstrap := hex.EncodeToString(token)
	if initialized == 0 {
		log.Printf("First-run setup credential (operator only): %s", bootstrap)
	}
	addr := os.Getenv("MINDFLOW_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	a := app.New(db, app.Config{Dev: os.Getenv("MINDFLOW_DEV") == "1", Bootstrap: bootstrap})
	srv := &http.Server{Addr: addr, Handler: a, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go a.RunWorkers(ctx)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Printf("mindflow listening on %s", addr)
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
