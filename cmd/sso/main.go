package main

import (
	"cmd/sso/main.go/internal/app"
	"cmd/sso/main.go/internal/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("staring application")

	application := app.New(log, cfg.Grpc.Port, cfg.StoragePath, 10)

	go application.GRPCSrv.MustRun()

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
