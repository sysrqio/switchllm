package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sysrqio/switchllm/internal/config"
	"github.com/sysrqio/switchllm/internal/metrics"
	"github.com/sysrqio/switchllm/internal/router"
	"github.com/sysrqio/switchllm/internal/server"
	"github.com/sysrqio/switchllm/internal/upstream"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "start":
		os.Exit(runStart(os.Args[2:]))
	case "version":
		fmt.Println("switchllm 0.1.0")
		os.Exit(0)
	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`switchllm — local OpenAI-compatible smart routing proxy

Usage:
  switchllm start [flags]
  switchllm version

Start flags:
  --port          HTTP listen port (default 8080)
  --config        Path to YAML config (default switchllm.yaml)
  --threshold     Complexity threshold for premium routing (default 0.35)
  --log-level     debug|info|warn|error (default info)
  --show-savings  Log estimated cost savings when routing to simple model
  --dry-run       Use mock upstream (no API keys required)

Environment (BYOK):
  OPENAI_API_KEY     OpenAI-compatible upstream
  ANTHROPIC_API_KEY  Reserved for future Anthropic routing

No telemetry is collected or uploaded.`)
}

func runStart(args []string) int {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP listen port")
	configPath := fs.String("config", "switchllm.yaml", "config file path")
	threshold := fs.Float64("threshold", 0.35, "complexity threshold for premium model")
	logLevel := fs.String("log-level", "info", "log level")
	showSavings := fs.Bool("show-savings", false, "log estimated savings")
	dryRun := fs.Bool("dry-run", false, "mock upstream responses")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *port > 0 {
		cfg.Server.Port = *port
	}
	th := *threshold
	rtr := router.New(cfg, &th)

	level := parseLevel(*logLevel)
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	openAIKey := os.Getenv("OPENAI_API_KEY")
	var client upstream.Client
	useMock := *dryRun || openAIKey == ""
	if useMock {
		if !*dryRun && openAIKey == "" {
			log.Warn("OPENAI_API_KEY not set; enabling dry-run mock upstream")
		}
		client = upstream.NewMock()
	} else {
		client = upstream.NewOpenAI(cfg.Providers.OpenAI.BaseURL, openAIKey)
	}

	m := metrics.New(*showSavings)
	m.SetDryRun(useMock)

	srv := server.New(rtr, client, m, log, *showSavings)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpSrv := &http.Server{Addr: addr, Handler: srv.Handler()}

	log.Info("switchllm listening", "addr", addr, "dry_run", useMock, "threshold", th)
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		log.Info("anthropic key present (MVP routes via OpenAI client only)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	_ = httpSrv.Shutdown(context.Background())
	log.Info("shutdown complete")
	return 0
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
