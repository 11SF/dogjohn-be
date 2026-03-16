package main

import (
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/11SF/dogjohn-be/config"
	"github.com/11SF/dogjohn-be/router"

	"gitdev.devops.krungthai.com/starwolf/backend/common/logger"
	"gitdev.devops.krungthai.com/starwolf/backend/common/shutdown"

	_ "embed"
	_ "time/tzdata"
)

const (
	gracefulShutdownDuration = 10 * time.Second
	serverReadHeaderTimeout  = 5 * time.Second
	serverReadTimeout        = 5 * time.Second
	serverWriteTimeout       = 10 * time.Second // request hangup after this durations
	handlerTimeout           = serverWriteTimeout - (time.Millisecond * 100)
)

// go build -ldflags "-X main.commit=123456"
var commit string

//go:embed VERSION
var version string

func init() {
	if os.Getenv("GOMAXPROCS") != "" {
		runtime.GOMAXPROCS(0) // GOMAXPROCS
	} else {
		runtime.GOMAXPROCS(1) // 0 - 999m
	}
	if os.Getenv("GOMEMLIMIT") != "" {
		debug.SetMemoryLimit(-1) // GOMEMLIMIT
	}
}

func main() {
	cfg := config.C(config.Env)
	_ = logger.New(logger.GCPKeyReplacer)

	r, stop := router.New(cfg, version, commit, handlerTimeout)
	defer stop()

	srv := newServer(cfg, r)

	go shutdown.Graceful(srv, gracefulShutdownDuration)

	slog.Info("run", "port", cfg.Server.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("HTTP server ListenAndServe", "error", err)
		return
	}

	slog.Info("bye")
}

func newServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		MaxHeaderBytes:    1 << 20,
	}
}
