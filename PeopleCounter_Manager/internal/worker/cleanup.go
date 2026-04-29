package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func StartLogCleanup(ctx context.Context, pool *pgxpool.Pool, keepDays int) {
	slog.Info("Запущена фоновая задача очистки логов", "keep_days", keepDays)

	// Запускаем сразу при старте
	cleanLogs(pool, keepDays)

	// И затем каждый день
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Остановка задачи очистки логов")
				return
			case <-ticker.C:
				cleanLogs(pool, keepDays)
			}
		}
	}()
}

func cleanLogs(pool *pgxpool.Pool, keepDays int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := fmt.Sprintf("DELETE FROM logs WHERE timestamp < NOW() - INTERVAL '%d days'", keepDays)

	res, err := pool.Exec(ctx, query)
	if err != nil {
		slog.Error("Ошибка при удалении старых логов", "error", err.Error())
		return
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected > 0 {
		slog.Info("Успешно удалены старые логи", "deleted_rows", rowsAffected)
	}
}
