// Command mindflow serves the canvas application and operator maintenance commands.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mindflow/internal/mindflow"
	"mindflow/internal/mindflow/storage"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	healthTimeout   = 3 * time.Second
	bootstrapBytes  = 24
	headerTimeout   = 10 * time.Second
	readTimeout     = time.Minute
	idleTimeout     = 90 * time.Second
	shutdownTimeout = 30 * time.Second
	defaultAddress  = "0.0.0.0:8080"
)

var errHealth = errors.New("application is not ready")

const maxHeaderBytes = 64 << 10

func main() { os.Exit(runMain()) }

func runMain() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := run(ctx)
	if err != nil {
		slog.Error("mindflow stopped", "error", err)

		return 1
	}

	return 0
}

func run(ctx context.Context) error {
	address := os.Getenv("MINDFLOW_ADDR")
	if address == "" {
		address = defaultAddress
	}
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		return healthcheck(ctx, address)
	}
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
		switch command {
		case "migrate", "revoke-credentials", "check-data":
		default:
			return errors.ErrUnsupported
		}
	}
	directory := os.Getenv("MINDFLOW_DATA")
	if directory == "" {
		directory = "data"
	}
	connection, err := storage.Open(ctx, filepath.Join(directory, "mindflow.sqlite"))
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()
	err = connection.Migrate(ctx)
	if err != nil {
		return err
	}
	if command == "migrate" {
		slog.Info("All domain migrations are current.")

		return nil
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "revoke-credentials":
			err = connection.RevokeCredentials(ctx)
			if err == nil {
				slog.Info("Sessions, account links, queued token emails, and editor control revoked.")
			}

			return err
		case "check-data":
			err = connection.Check(ctx)
			if err == nil {
				slog.Info("Database integrity and foreign keys: OK")
			}

			return err
		default:
			return errors.ErrUnsupported
		}
	}
	token := make([]byte, bootstrapBytes)
	_, err = rand.Read(token)
	if err != nil {
		return wrap("generate setup credential", err)
	}
	bootstrap := hex.EncodeToString(token)
	initialized, err := connection.SetupComplete(ctx)
	if err != nil {
		return err
	}
	if !initialized {
		slog.Info("First-run setup credential (operator only): " + bootstrap)
	}
	service, err := mindflow.New(ctx, connection, mindflow.Config{Dev: os.Getenv("MINDFLOW_DEV") == "1", Bootstrap: bootstrap})
	if err != nil {
		return err
	}
	server := &http.Server{Addr: address, Handler: service, MaxHeaderBytes: maxHeaderBytes, ReadHeaderTimeout: headerTimeout, ReadTimeout: readTimeout, IdleTimeout: idleTimeout}
	go service.RunWorkers(ctx)
	go func() {
		<-ctx.Done()
		service.Close()
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	slog.Info("mindflow listening", "address", address)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return wrap("serve HTTP", err)
}

func healthcheck(ctx context.Context, address string) error {
	if strings.HasPrefix(address, ":") {
		address = "127.0.0.1" + address
	}
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		return wrap("healthcheck address", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return wrap("healthcheck port", err)
	}
	// Health checks always target this container's loopback interface.
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health/ready", port), nil) //nolint:gosec // G704: target is fixed loopback and the only variable is a parsed integer port.
	if err != nil {
		return wrap("healthcheck request", err)
	}
	client := http.Client{Timeout: healthTimeout}
	response, err := client.Do(request) //nolint:gosec // G704: the request above is constructed for fixed loopback with a parsed integer port.
	if err != nil {
		return wrap("healthcheck connection", err)
	}
	defer func() { _ = response.Body.Close() }()
	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return wrap("healthcheck response", err)
	}
	if response.StatusCode != http.StatusOK {
		return errHealth
	}

	return nil
}

func wrap(operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}
