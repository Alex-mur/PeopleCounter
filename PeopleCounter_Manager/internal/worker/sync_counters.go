package worker

import (
	"context"
	"log/slog"
	"time"

	"PeopleCounter_Manager/internal/service"
	"PeopleCounter_Manager/internal/storage"
)

func StartSyncWorker(ctx context.Context, store *storage.Storage, manager *service.CounterManager) {
	ticker := time.NewTicker(60 * time.Minute)
	slog.Info("Запущен воркер мониторинга и синхронизации счетчиков (интервал: 60 мин)")

	go func() {
		syncAllCounters(ctx, store, manager)

		for {
			select {
			case <-ctx.Done():
				slog.Info("Воркер синхронизации счетчиков остановлен")
				ticker.Stop()
				return
			case <-ticker.C:
				syncAllCounters(ctx, store, manager)
			}
		}
	}()
}

func syncAllCounters(ctx context.Context, store *storage.Storage, manager *service.CounterManager) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	services, err := store.GetAllServices(timeoutCtx)
	if err != nil {
		slog.Error("Синхронизация: ошибка получения списка сервисов", "error", err)
		return
	}

	for _, srv := range services {
		err := manager.SyncCounters(timeoutCtx, srv.ID)
		if err != nil {
			slog.Warn("Ошибка синхронизации для сервиса", "service_id", srv.ID, "service_name", srv.Name, "error", err)
		}
	}
}
