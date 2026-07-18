package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
	"github.com/getio0909/voice-asset-mcp/internal/config"
	"github.com/getio0909/voice-asset-mcp/internal/mcpserver"
)

var version = "dev"

const (
	maxMCPRequestBytes = 2 << 20
	mcpSessionTimeout  = 15 * time.Minute
)

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.FromEnv()
	flag.StringVar(&cfg.Transport, "transport", cfg.Transport, "transport: stdio or http")
	flag.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "Streamable HTTP listen address")
	flag.StringVar(&cfg.ServerBaseURL, "server-url", cfg.ServerBaseURL, "VoiceAsset Server base URL")
	flag.StringVar(&cfg.TLSCertFile, "tls-cert", cfg.TLSCertFile, "TLS certificate file for Streamable HTTP")
	flag.StringVar(&cfg.TLSKeyFile, "tls-key", cfg.TLSKeyFile, "TLS private key file for Streamable HTTP")
	flag.BoolVar(&cfg.EnableWrites, "enable-writes", cfg.EnableWrites, "enable state-changing MCP tools")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return nil
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration: %w", err)
	}

	client, err := backend.NewClient(cfg.ServerBaseURL, cfg.ServerToken, nil)
	if err != nil {
		return fmt.Errorf("create API client: %w", err)
	}
	server := mcpserver.NewWithOptions(client, version, mcpserver.Options{EnableWrites: cfg.EnableWrites})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cfg.Transport {
	case config.TransportStdio:
		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("serve stdio: %w", err)
		}
		return nil
	case config.TransportHTTP:
		return serveHTTP(ctx, server, cfg)
	default:
		return fmt.Errorf("unsupported transport %q", cfg.Transport)
	}
}

func serveHTTP(ctx context.Context, server *mcp.Server, cfg config.Config) error {
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           newHTTPHandler(server, cfg),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	var err error
	if cfg.TLSCertFile != "" {
		err = httpServer.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	} else {
		err = httpServer.ListenAndServe()
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve Streamable HTTP: %w", err)
	}
	<-shutdownDone
	return nil
}

func newHTTPHandler(server *mcp.Server, cfg config.Config) http.Handler {
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{SessionTimeout: mcpSessionTimeout},
	)
	protected := http.NewCrossOriginProtection().Handler(bearerAuth(handler, cfg.HTTPBearerToken))
	rateLimit := cfg.RateLimitPerMin
	if rateLimit <= 0 {
		rateLimit = 120
	}
	limited := newRateLimiter(rateLimit, time.Minute).Handler(protected)
	mux := http.NewServeMux()
	mux.Handle("/mcp", limitRequestBody(limited, maxMCPRequestBytes))
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return mux
}

func limitRequestBody(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBytes {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func bearerAuth(next http.Handler, token string) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("Authorization")
		expected := "Bearer " + token
		if len(provided) != len(expected) || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
