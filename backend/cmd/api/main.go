// Command api runs the Audio Preview Board backend, either as a normal HTTP
// server (local) or behind a Lambda Function URL (when running on Lambda).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"audiopreview/internal/config"
	"audiopreview/internal/httpapi"
	"audiopreview/internal/itunes"
	"audiopreview/internal/service"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svc := service.New(itunes.NewHTTPClient(cfg.ITunesBaseURL, cfg.UpstreamTimeout), cfg.CacheTTL)
	svc.StartJanitor(ctx, 5*time.Minute)
	handler := httpapi.NewRouter(svc, cfg.AllowedOrigin, log)

	// Lambda sets this variable; its Function URL payload matches the API Gateway v2 format.
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		log.Info("starting on lambda", "cache_ttl", cfg.CacheTTL.String(), "allowed_origin", cfg.AllowedOrigin)
		lambda.Start(httpadapter.NewV2(handler).ProxyWithContext)
		return
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Info("listening", "addr", srv.Addr, "cache_ttl", cfg.CacheTTL.String(), "allowed_origin", cfg.AllowedOrigin)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}
}
