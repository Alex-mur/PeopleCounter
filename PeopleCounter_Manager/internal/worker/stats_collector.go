package worker

import (
	"context"
	"log/slog"
	"time"

	"PeopleCounter_Manager/internal/service"
	"PeopleCounter_Manager/internal/storage"
)

func StartStatsCollector(ctx context.Context, store *storage.Storage, manager *service.CounterManager) {
	ticker := time.NewTicker(30 * time.Minute)

	slog.Info("Запущена фоновая задача сбора статистики (каждые 30 минут)")

	go func() {
		collectAllStats(ctx, store, manager)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Остановка сборщика статистики")
				ticker.Stop()
				return
			case <-ticker.C:
				collectAllStats(ctx, store, manager)
			}
		}
	}()
}

func collectAllStats(ctx context.Context, store *storage.Storage, manager *service.CounterManager) {
	slog.Info("Начат сбор статистики со всех камер...")

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	services, err := store.GetAllServices(timeoutCtx)
	if err != nil {
		slog.Error("Не удалось получить список сервисов для сбора статистики", "error", err)
		return
	}

	for _, srv := range services {
		counters, err := store.GetCountersByServiceID(timeoutCtx, srv.ID)
		if err != nil {
			slog.Error("Не удалось получить камеры", "service_id", srv.ID, "error", err)
			continue
		}

		for _, counter := range counters {
			err := manager.CollectStatsForCounter(timeoutCtx, srv, counter.ID)
			if err != nil {
				slog.Warn("Ошибка при сборе статистики",
					"service", srv.Name,
					"counter_id", counter.ID,
					"error", err,
				)
			}
		}
	}
	slog.Info("Сбор статистики завершен")
}
