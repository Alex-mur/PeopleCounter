package main

import (
	"PeopleCounter_Manager/internal/api"
	"PeopleCounter_Manager/internal/api/handlers"
	"PeopleCounter_Manager/internal/config"
	"PeopleCounter_Manager/internal/logger"
	"PeopleCounter_Manager/internal/service"
	"PeopleCounter_Manager/internal/storage"
	"PeopleCounter_Manager/internal/worker"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "PeopleCounter_Manager/docs"
	"github.com/go-chi/jwtauth/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// @title People Counter Manager API
// @version 1.0
// @description REST API для управления счетчиками посетителей и сбора статистики
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @host localhost:9000
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.LoadConfig("config.json")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DBConnString)
	if err != nil {
		slog.Error("Ошибка БД: %v\n", err)
	}
	defer pool.Close()

	if err := logger.InitLogsTable(ctx, pool); err != nil {
		fmt.Printf("Ошибка создания таблицы логов: %v\n", err)
		os.Exit(1)
	}
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	dbHandler := logger.NewDBHandler(pool, slog.HandlerOptions{Level: slog.LevelInfo})
	multiHandler := logger.NewMultiHandler(consoleHandler, dbHandler)
	slogLogger := slog.New(multiHandler)
	slog.SetDefault(slogLogger)

	slog.Info("Инициализация приложения...")
	store := storage.NewStorage(pool)
	store.InitDB(ctx)

	tokenAuth := jwtauth.New("HS256", []byte(cfg.JWTSecret), nil)
	authService := service.NewAuthService(store, tokenAuth)
	authHandler := handlers.NewAuthHandler(authService)
	counterManager := service.NewCounterManager(store)
	counterHandler := handlers.NewCounterHandler(counterManager)
	userManager := service.NewUserManager(store)
	usersHandler := handlers.NewUserHandler(userManager)
	router := api.NewRouter(authHandler, counterHandler, usersHandler, tokenAuth)

	worker.StartLogCleanup(ctx, pool, cfg.KeepLogDays)
	worker.StartStatsCollector(ctx, store, counterManager)
	worker.StartSyncWorker(ctx, store, counterManager)

	addr := fmt.Sprintf(":%d", cfg.APIPort)
	slog.Info("Сервер запущен", "address", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Info("Ошибка сервера", "err", err)
	}
}
