package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
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
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		addr := os.Getenv("MINDFLOW_ADDR")
		if addr == "" {
			addr = ":8080"
		}
		if addr[0] == ':' {
			addr = "127.0.0.1" + addr
		}
		if len(addr) > 8 && addr[:8] == "0.0.0.0:" {
			addr = "127.0.0.1:" + addr[8:]
		}
		c := http.Client{Timeout: 3 * time.Second}
		r, e := c.Get("http://" + addr + "/health/ready")
		if e != nil {
			os.Exit(1)
		}
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
		if r.StatusCode != 200 {
			os.Exit(1)
		}
		return
	}

	dir := os.Getenv("MINDFLOW_DATA")
	if dir == "" {
		dir = "data"
	}
	db, err := store.Open(filepath.Join(dir, "mindflow.sqlite"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "revoke-credentials":
			if err = store.RevokeCredentials(db); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Sessions, account links, queued token emails, and editor control revoked.")
			return
		case "check-data":
			if err = store.Check(db); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Database integrity and foreign keys: OK")
			return
		default:
			log.Fatal("unknown command")
		}
	}
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
	a, err := app.New(db, app.Config{Dev: os.Getenv("MINDFLOW_DEV") == "1", Bootstrap: bootstrap})
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Addr: addr, Handler: a, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go a.RunWorkers(ctx)
	go func() {
		<-ctx.Done()
		a.Close()
		shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Printf("mindflow listening on %s", addr)
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
