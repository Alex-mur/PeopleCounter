package service

import (
	"PeopleCounter_Manager/internal/models"
	"PeopleCounter_Manager/internal/storage"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type CounterManager struct {
	storage *storage.Storage
	client  *http.Client
}

func NewCounterManager(store *storage.Storage) *CounterManager {
	return &CounterManager{
		storage: store,
		client: &http.Client{
			Timeout: 10 * time.Second, // Таймаут для запросов к воркерам
		},
	}
}

func (m *CounterManager) sendWorkerRequest(ctx context.Context, method string, service models.CounterService, endpoint string, payload any) error {
	var bodyReader io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("ошибка сериализации payload: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := fmt.Sprintf("%s%s", service.APIUrl, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Api-Key", service.APIKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка соединения с воркером %s: %w", service.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("воркер вернул ошибку %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (m *CounterManager) CreateService(ctx context.Context, req models.CounterService) (int, error) {
	return m.storage.CreateService(ctx, req)
}

func (m *CounterManager) UpdateService(ctx context.Context, id int, req models.CounterService) error {
	return m.storage.UpdateService(ctx, id, req)
}

func (m *CounterManager) ListServices(ctx context.Context) ([]models.CounterService, error) {
	return m.storage.GetAllServices(ctx)
}

func (m *CounterManager) GetServiceByID(ctx context.Context, id int) (models.CounterService, error) {
	return m.storage.GetServiceByID(ctx, id)
}

func (m *CounterManager) DeleteService(ctx context.Context, id int) error {
	return m.storage.DeleteService(ctx, id)
}

func (m *CounterManager) CreateCounter(ctx context.Context, req models.Counter) (int, error) {
	id, err := m.storage.CreateCounter(ctx, req)
	if err != nil {
		return 0, err
	}
	req.ID = id

	targetService, err := m.storage.GetServiceByID(ctx, req.ServiceID)
	if err != nil {
		_ = m.storage.DeleteCounter(ctx, id) // Откат
		return 0, fmt.Errorf("привязанный воркер не найден: %v", err)
	}

	err = m.sendWorkerRequest(ctx, http.MethodPost, targetService, "/api/counters", req)
	if err != nil {
		slog.Error("Ошибка отправки конфигурации на воркер, откат транзакции",
			"counter_id", id,
			"error", err.Error(),
		)

		rollbackErr := m.storage.DeleteCounter(ctx, id)
		if rollbackErr != nil {
			slog.Error("КРИТИЧЕСКАЯ ОШИБКА: Не удалось удалить счетчик при откате",
				"counter_id", id,
				"error", rollbackErr.Error(),
			)
			return 0, fmt.Errorf("воркер отклонил запрос (%v), и произошла ошибка при откате БД: %v", err, rollbackErr)
		}

		return 0, fmt.Errorf("воркер недоступен или отклонил конфигурацию: %v", err)
	}

	return id, nil
}

func (m *CounterManager) ListCounters(ctx context.Context, userID int, role string) ([]models.Counter, error) {
	if role == "admin" {
		return m.storage.GetAllCounters(ctx)
	}
	return m.storage.GetUserCounters(ctx, userID)
}

func (m *CounterManager) getWorkerCounters(ctx context.Context, service models.CounterService) ([]models.Counter, error) {
	url := fmt.Sprintf("%s/api/counters", service.APIUrl)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header["X-Api-Key"] = []string{service.APIKey}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("воркер вернул код %d", resp.StatusCode)
	}

	var counters []models.Counter
	if err := json.NewDecoder(resp.Body).Decode(&counters); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа воркера: %v", err)
	}

	return counters, nil
}

func (m *CounterManager) DeleteCounter(ctx context.Context, id int) error {
	counter, err := m.storage.GetCounterByID(ctx, id)
	if err != nil {
		return fmt.Errorf("ошибка получения камеры из БД: %w", err)
	}

	targetService, err := m.storage.GetServiceByID(ctx, counter.ServiceID)
	if err != nil {
		return fmt.Errorf("сервис не найден в БД: %w", err)
	}

	endpoint := fmt.Sprintf("/api/counters/%d", id)
	err = m.sendWorkerRequest(ctx, http.MethodDelete, targetService, endpoint, nil)
	if err != nil {
		slog.Warn("Не удалось удалить камеру на воркере (возможно он недоступен)", "id", id, "error", err.Error())
	}

	return m.storage.DeleteCounter(ctx, id)
}

func (m *CounterManager) UpdateCounter(ctx context.Context, id int, req models.Counter) error {
	existingCounter, err := m.storage.GetCounterByID(ctx, id)
	if err != nil {
		return fmt.Errorf("камера не найдена в БД: %v", err)
	}

	targetService, err := m.storage.GetServiceByID(ctx, existingCounter.ServiceID)
	if err != nil {
		return errors.New("привязанный воркер не найден")
	}

	req.ID = id

	endpoint := fmt.Sprintf("/api/counters/%d", id)
	err = m.sendWorkerRequest(ctx, http.MethodPut, targetService, endpoint, req)
	if err != nil {
		slog.Error("Воркер отклонил обновление камеры", "counter_id", id, "error", err.Error())
		return fmt.Errorf("воркер недоступен или отклонил обновление: %v", err)
	}

	err = m.storage.UpdateCounter(ctx, id, req)
	if err != nil {
		slog.Error("КРИТИЧЕСКАЯ ОШИБКА: Воркер обновился, но БД менеджера не смогла сохранить изменения",
			"counter_id", id,
			"error", err.Error(),
		)

		_ = m.sendWorkerRequest(ctx, http.MethodPut, targetService, endpoint, existingCounter)

		return fmt.Errorf("ошибка сохранения в БД после успешного обновления на воркере: %v", err)
	}

	return nil
}

func (m *CounterManager) SyncCounters(ctx context.Context, serviceID int) error {
	targetService, err := m.storage.GetServiceByID(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("воркер не найден: %v", err)
	}

	dbCounters, err := m.storage.GetCountersByServiceID(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("ошибка получения камер из БД: %v", err)
	}

	workerCounters, err := m.getWorkerCounters(ctx, targetService)
	if err != nil {
		return fmt.Errorf("не удалось получить камеры от воркера: %v", err)
	}

	dbMap := make(map[int]models.Counter)
	for _, c := range dbCounters {
		dbMap[c.ID] = c
	}

	workerMap := make(map[int]models.Counter)
	for _, c := range workerCounters {
		workerMap[c.ID] = c
	}

	var syncErrors []error

	for dbID, dbCounter := range dbMap {
		workerCounter, existsOnWorker := workerMap[dbID]

		if existsOnWorker {
			if !configsMatch(dbCounter, workerCounter) {
				endpoint := fmt.Sprintf("/api/counters/%d", dbID)
				err := m.sendWorkerRequest(ctx, http.MethodPut, targetService, endpoint, dbCounter)
				if err != nil {
					syncErrors = append(syncErrors, fmt.Errorf("не удалось обновить счетчик %d: %v", dbID, err))
				} else {
					slog.Info("Синхронизация: конфигурация счетчика обновлена на воркере", "id", dbID, "worker", targetService.Name)
				}
			} else {
				slog.Info("Синхронизация: счетчик в актуальном состоянии, обновление не требуется", "id", dbID, "worker", targetService.Name)
			}
		} else {
			err := m.sendWorkerRequest(ctx, http.MethodPost, targetService, "/api/counters", dbCounter)
			if err != nil {
				syncErrors = append(syncErrors, fmt.Errorf("не удалось создать счетчик %d: %v", dbID, err))
			} else {
				slog.Info("Синхронизация: счетчик создан на воркере", "id", dbID, "worker", targetService.Name)
			}
		}
	}

	for workerID := range workerMap {
		_, existsInDB := dbMap[workerID]
		if !existsInDB {
			endpoint := fmt.Sprintf("/api/counters/%d", workerID)
			err := m.sendWorkerRequest(ctx, http.MethodDelete, targetService, endpoint, nil)
			if err != nil {
				syncErrors = append(syncErrors, fmt.Errorf("не удалось удалить неизвестный счетчик %d: %v", workerID, err))
			} else {
				slog.Info("Синхронизация: неизвестный счетчик удален", "id", workerID, "worker", targetService.Name)
			}
		}
	}

	if len(syncErrors) > 0 {
		return fmt.Errorf("ошибки при синхронизации: %v", syncErrors)
	}

	return nil
}

func configsMatch(c1, c2 models.Counter) bool {
	if c1.Name != c2.Name ||
		c1.Description != c2.Description ||
		c1.Url != c2.Url ||
		c1.VidStride != c2.VidStride {
		return false
	}

	b1Groups, _ := json.Marshal(c1.Groups)
	b2Groups, _ := json.Marshal(c2.Groups)
	if string(b1Groups) != string(b2Groups) {
		return false
	}

	b1Lines, _ := json.Marshal(c1.Lines)
	b2Lines, _ := json.Marshal(c2.Lines)
	if string(b1Lines) != string(b2Lines) {
		return false
	}

	return true
}

func (m *CounterManager) CollectStatsForCounter(ctx context.Context, targetService models.CounterService, counterID int) error {
	endpoint := fmt.Sprintf("/api/counters/%d/stats?period=hour", counterID)
	url := fmt.Sprintf("%s%s", targetService.APIUrl, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header["X-API-Key"] = []string{targetService.APIKey}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка запроса к воркеру: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("воркер вернул код %d", resp.StatusCode)
	}

	var stats models.CounterStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	loc := time.Local

	for _, record := range stats.History {
		if record.Period == nil {
			continue
		}

		parsedTime, err := time.ParseInLocation("2006-01-02T15:04:05", *record.Period, loc)
		if err != nil {
			parsedTime, err = time.Parse(time.RFC3339, *record.Period)
			if err != nil {
				slog.Warn("Не удалось распарсить время статистики", "time", *record.Period)
				continue
			}
		}

		err = m.storage.UpsertStats(ctx, counterID, record.GroupID, parsedTime, record.Passes)
		if err != nil {
			slog.Error("Ошибка сохранения статистики", "counter_id", counterID, "error", err)
		}
	}

	return nil
}

func (m *CounterManager) GetVideoStream(ctx context.Context, counterID int) (*http.Response, error) {
	counter, err := m.storage.GetCounterByID(ctx, counterID)
	if err != nil {
		return nil, fmt.Errorf("камера не найдена: %v", err)
	}

	targetService, err := m.storage.GetServiceByID(ctx, counter.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("воркер не найден: %v", err)
	}

	endpoint := fmt.Sprintf("/api/counters/%d/stream", counterID)
	url := fmt.Sprintf("%s%s", targetService.APIUrl, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header["X-API-Key"] = []string{targetService.APIKey}

	streamClient := &http.Client{
		Timeout: 0,
	}

	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к потоку: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("воркер вернул статус %d", resp.StatusCode)
	}

	return resp, nil
}

func (m *CounterManager) GetCounterStats(ctx context.Context, counterID int, period string, dateStart, dateEnd *time.Time) ([]models.CounterStatsRecord, error) {
	_, err := m.storage.GetCounterByID(ctx, counterID)
	if err != nil {
		return nil, fmt.Errorf("камера не найдена: %v", err)
	}
	return m.storage.GetAggregatedStats(ctx, counterID, period, dateStart, dateEnd)
}
