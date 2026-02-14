package main

import (
	"auth/internal/handlers"
	"auth/internal/storage"
	"auth/pkg/middleware"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Error("Ошибка загрузки .env", "ошибка", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	router := http.NewServeMux()
	server := &http.Server{
		Addr: ":" + os.Getenv("SERVER_PORT"),
		Handler: middleware.Recover(
			middleware.Logging(router)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	var mdb storage.ManagerDB

	DBType := os.Getenv("DB_TYPE")
	switch DBType {
	case "postgres":
		mdb = storage.NewPsql()
	default:
		mdb = storage.NewPsql()
	}
	if err := mdb.Init(); err != nil {
		slog.Error("ошибка подключения к БД", "error", err)
		return
	}

	authHandler := handlers.NewAuthHandler(mdb)
	regHandler := handlers.NewRegHandler(mdb)
	router.Handle("POST /auth", authHandler)
	router.Handle("POST /reg", regHandler)
	router.HandleFunc("/health", handlers.Health)

	go func() {
		slog.Info("Сервер запущен", "port", os.Getenv("SERVER_PORT"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Ошибка при запуске сервера", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	slog.Info("Получен сигнал остановки, начинаем graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Ошибка при остановке сервера", "error", err)
	} else {
		slog.Info("HTTP сервер успешно остановлен")
	}

	if err := mdb.Close(); err != nil {
		slog.Error("Ошибка при закрытии БД", "error", err)
	} else {
		slog.Info("Соединение с БД успешно закрыто")
	}

	slog.Info("Приложение успешно остановлено")
}
