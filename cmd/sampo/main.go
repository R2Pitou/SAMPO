package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"sampo/internal/app"
	"sampo/internal/catalog"
	"sampo/internal/gateway"
	"sampo/internal/host"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sampo:", err)
		os.Exit(1)
	}
}

func run() error {
	defaultDir, err := defaultDataDir()
	if err != nil {
		return err
	}
	dataDir := flag.String("data-dir", defaultDir, "SAMPO application-data directory")
	noBrowser := flag.Bool("no-browser", false, "do not open the default browser; print the one-time launch URL")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		return fmt.Errorf("create application data directory: %w", err)
	}
	instance, err := host.AcquireInstance()
	if err != nil {
		return err
	}
	defer instance.Close()

	logger, closeLog, err := newLogger(*dataDir)
	if err != nil {
		return err
	}
	defer closeLog()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := catalog.Open(ctx, filepath.Join(*dataDir, "catalogue.sqlite3"))
	if err != nil {
		return err
	}
	defer store.Close()

	service := app.New(store)
	localGateway, err := gateway.New(service, ctx, logger)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("bind loopback Gateway: %w", err)
	}
	defer listener.Close()

	baseURL := "http://" + listener.Addr().String()
	handler, err := localGateway.Handler(baseURL)
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 * 1024,
		ErrorLog:          logger,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Printf("gateway start bind=%q mode=loopback-only", listener.Addr().String())
		serverErr <- server.Serve(listener)
	}()

	launchURL := localGateway.BootstrapURL(baseURL)
	if *noBrowser {
		fmt.Println("Open this one-time SAMPO URL:")
		fmt.Println(launchURL)
	} else if err := host.OpenBrowser(launchURL); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		logger.Printf("shutdown reason=signal next=graceful-stop")
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve Gateway: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown Gateway: %w", err)
	}
	if err := localGateway.Wait(shutdownCtx); err != nil {
		return fmt.Errorf("wait for background scans: %w", err)
	}
	return nil
}

func defaultDataDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate per-user application data: %w", err)
	}
	return filepath.Join(base, "sampo"), nil
}

func newLogger(dataDir string) (*log.Logger, func(), error) {
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(logDir, "sampo.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open application log: %w", err)
	}
	logger := log.New(io.MultiWriter(os.Stderr, file), "sampo ", log.Ldate|log.Ltime|log.LUTC)
	return logger, func() { _ = file.Close() }, nil
}
