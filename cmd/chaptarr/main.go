// Command chaptarr is a self-hosted audiobook manager: it follows authors,
// finds releases on your indexers, hands them to a download client and files
// the results into your library.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ncecere/chaptarr/internal/api"
	"github.com/ncecere/chaptarr/internal/logbuf"
	"github.com/ncecere/chaptarr/internal/service"
	"github.com/ncecere/chaptarr/internal/store"
	"github.com/ncecere/chaptarr/web"
)

// version and commit are set at build time with
// -ldflags "-X main.version=... -X main.commit=...".
var (
	version = "dev"
	commit  = "unknown"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "chaptarr")
	}
	return "data"
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "chaptarr:", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", envOr("CHAPTARR_ADDR", ":8789"), "address to listen on (env CHAPTARR_ADDR)")
	dataDir := flag.String("data", envOr("CHAPTARR_DATA", defaultDataDir()), "directory for the database (env CHAPTARR_DATA)")
	debug := flag.Bool("debug", os.Getenv("CHAPTARR_DEBUG") != "", "log debug messages (env CHAPTARR_DEBUG)")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	// `chaptarr version` prints "chaptarr v1.2.3 (abc1234)"; the release
	// pipeline checks a tagged image against it.
	if *showVersion || flag.Arg(0) == "version" {
		short := commit
		if len(short) > 7 {
			short = short[:7]
		}
		fmt.Printf("chaptarr %s (%s)\n", version, short)
		return nil
	}

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logs := logbuf.NewBuffer(2000)
	log := logbuf.NewLogger(os.Stderr, logs, level)

	abs, err := filepath.Abs(*dataDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}
	st, err := store.Open(abs)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer st.Close()

	svc := service.New(st, log, version, abs)
	srv := &http.Server{
		Handler:           api.New(svc, logs, web.Dist()).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go svc.Run(ctx)

	log.Info("chaptarr started", "version", version, "listen", ln.Addr().String(), "data", abs)
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
