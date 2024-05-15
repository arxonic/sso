package main

import (
	"cmd/sso/main.go/internal/app"
	"cmd/sso/main.go/internal/config"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"

	ssov1 "github.com/arxonic/protos/gen/go/sso"
)

var (
	tokenTTL    = 1 * time.Hour
	httpAddress = "localhost:8088"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("staring application")

	application := app.New(log, cfg.Grpc.Port, cfg.StoragePath, tokenTTL)
	// обернуть в приложение бд

	ctx := context.Background()

	go application.GRPCSrv.MustRun()
	go func() {
		if err := startHttpServer(ctx, log); err != nil {
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	notify := <-stop

	application.GRPCSrv.Stop()

	log.Info("application stopped", slog.String("signal", notify.String()))
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case "local":
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "prod":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log
}

func startHttpServer(ctx context.Context, log *slog.Logger) error {
	mux := runtime.NewServeMux()

	withCors := cors.New(cors.Options{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"ACCEPT", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler(mux)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	err := ssov1.RegisterAuthHandlerFromEndpoint(ctx, mux, ":44044", opts)
	if err != nil {
		return err
	}

	log.Info("starting REST API running", slog.String("addr", httpAddress))

	return http.ListenAndServe(httpAddress, withCors)
}
