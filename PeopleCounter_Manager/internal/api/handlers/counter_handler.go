package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"PeopleCounter_Manager/internal/models"
	"PeopleCounter_Manager/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

type CounterHandler struct {
	manager *service.CounterManager
}

func NewCounterHandler(manager *service.CounterManager) *CounterHandler {
	return &CounterHandler{manager: manager}
}

// CreateService godoc
// @Summary Создать новый воркер (сервис)
// @Description Добавляет новый Python-воркер в систему для управления камерами.
// @Tags Services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CounterService true "Конфигурация сервиса (APIUrl, APIKey, Name)"
// @Success 201 {object} models.CounterService
// @Failure 400 {string} string "Неверный формат JSON"
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/services [post]
func (h *CounterHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	var req models.CounterService
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	id, err := h.manager.CreateService(r.Context(), req)
	if err != nil {
		slog.Error("Ошибка создания сервиса", "error", err.Error())
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	req.ID = id
	req.APIKey = ""

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

// UpdateService godoc
// @Summary Обновить воркер (сервис)
// @Description Обновляет данные существующего Python-воркера по его ID.
// @Tags Services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID сервиса"
// @Param request body models.CounterService true "Новые данные сервиса"
// @Success 200 {object} models.CounterService
// @Failure 400 {string} string "Неверный ID или формат JSON"
// @Failure 500 {string} string "Ошибка сервера или сервис не найден"
// @Router /api/services/{id} [put]
func (h *CounterHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	var req models.CounterService
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Неверный формат JSON при обновлении сервиса", "error", err.Error())
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	if err := h.manager.UpdateService(r.Context(), id, req); err != nil {
		slog.Error("Ошибка обновления сервиса", "id", id, "error", err.Error())
		http.Error(w, "Ошибка сервера или сервис не найден", http.StatusInternalServerError)
		return
	}

	req.ID = id
	req.APIKey = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

// ListServices godoc
// @Summary Получить список воркеров
// @Description Возвращает все зарегистрированные Python-воркеры (APIKey скрыт в ответе).
// @Tags Services
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.CounterService
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/services [get]
func (h *CounterHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.manager.ListServices(r.Context())
	if err != nil {
		slog.Error("Ошибка получения сервисов", "error", err.Error())
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	if services == nil {
		services = []models.CounterService{}
	}

	for i := range services {
		services[i].APIKey = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

// GetService godoc
// @Summary Получить информацию о воркере
// @Description Возвращает данные одного Python-воркера по его ID (APIKey скрыт в ответе).
// @Tags Services
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID сервиса"
// @Success 200 {object} models.CounterService
// @Failure 400 {string} string "Неверный ID"
// @Failure 404 {string} string "Сервис не найден"
// @Router /api/services/{id} [get]
func (h *CounterHandler) GetService(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	service, err := h.manager.GetServiceByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Сервис не найден", http.StatusNotFound)
		return
	}

	// Скрываем APIKey
	service.APIKey = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
}

// DeleteService godoc
// @Summary Удалить воркер (сервис)
// @Description Удаляет Python-воркер из БД. Связанные с ним камеры также будут каскадно удалены.
// @Tags Services
// @Security BearerAuth
// @Param id path int true "ID сервиса"
// @Success 204 "Успешное удаление"
// @Failure 400 {string} string "Неверный ID"
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/services/{id} [delete]
func (h *CounterHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	if err := h.manager.DeleteService(r.Context(), id); err != nil {
		slog.Error("Ошибка удаления сервиса", "error", err.Error())
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateCounter godoc
// @Summary Создать новый счетчик (камеру)
// @Description Сохраняет камеру в БД и отправляет её конфигурацию (линии, группы) на привязанный воркер.
// @Tags Counters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.Counter true "Конфигурация камеры (url, lines, groups, service_id)"
// @Success 201 {object} models.Counter
// @Failure 400 {string} string "Неверный формат JSON"
// @Failure 500 {string} string "Ошибка сервера или отказ воркера"
// @Router /api/counters [post]
func (h *CounterHandler) CreateCounter(w http.ResponseWriter, r *http.Request) {
	var req models.Counter
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	id, err := h.manager.CreateCounter(r.Context(), req)
	if err != nil {
		slog.Error("Ошибка создания камеры", "error", err.Error())
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	req.ID = id

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

// ListCounters godoc
// @Summary Получить список счетчиков
// @Description Возвращает все камеры (админам) или только назначенные пользователю
// @Tags Counters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Counter
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/counters [get]
func (h *CounterHandler) ListCounters(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	role, _ := claims["role"].(string)

	userIDFloat, _ := claims["user_id"].(float64)
	userID := int(userIDFloat)

	counters, err := h.manager.ListCounters(r.Context(), userID, role)
	if err != nil {
		slog.Error("Ошибка получения камер", "error", err.Error())
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	if counters == nil {
		counters = []models.Counter{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counters)
}

// DeleteCounter godoc
// @Summary Удалить счетчик (камеру)
// @Description Удаляет камеру из БД и отправляет команду на остановку потока на воркер.
// @Tags Counters
// @Security BearerAuth
// @Param id path int true "ID камеры"
// @Success 204 "Успешное удаление"
// @Failure 400 {string} string "Неверный ID"
// @Failure 404 {string} string "Камера не найдена в БД"
// @Router /api/counters/{id} [delete]
func (h *CounterHandler) DeleteCounter(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	if err := h.manager.DeleteCounter(r.Context(), id); err != nil {
		slog.Error("Ошибка удаления камеры", "error", err.Error())
		http.Error(w, "Камера не найдена в БД", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateCounter godoc
// @Summary Обновить счетчик (камеру)
// @Description Обновляет конфигурацию камеры (линии, группы, URL) на лету. Изменения применяются на воркере без остановки сервиса.
// @Tags Counters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID камеры"
// @Param request body models.Counter true "Полная обновленная конфигурация"
// @Success 200 {object} models.Counter
// @Failure 400 {string} string "Неверный ID или JSON"
// @Failure 500 {string} string "Ошибка сервера или отказ воркера"
// @Router /api/counters/{id} [put]
func (h *CounterHandler) UpdateCounter(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	var req models.Counter
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Неверный формат JSON при обновлении камеры", "error", err.Error())
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	if err := h.manager.UpdateCounter(r.Context(), id, req); err != nil {
		slog.Error("Ошибка обновления камеры", "id", id, "error", err.Error())
		http.Error(w, "Ошибка сервера или камера не найдена", http.StatusInternalServerError)
		return
	}

	req.ID = id

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

// SyncServiceCounters godoc
// @Summary Синхронизировать камеры с воркером
// @Description Принудительно сопоставляет камеры в БД с камерами на воркере. Удаляет лишние на воркере, создает недостающие, обновляет существующие.
// @Tags Services
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID сервиса (воркера)"
// @Success 200 {object} map[string]string "{\"status\":\"success\"}"
// @Failure 400 {string} string "Неверный ID сервиса"
// @Failure 500 {string} string "Синхронизация завершилась с ошибками"
// @Router /api/services/{id}/sync [post]
func (h *CounterHandler) SyncServiceCounters(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	serviceID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID сервиса", http.StatusBadRequest)
		return
	}

	if err := h.manager.SyncCounters(r.Context(), serviceID); err != nil {
		slog.Error("Синхронизация завершилась с ошибками", "service_id", serviceID, "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// StreamCounterVideo godoc
// @Summary Получить MJPEG видеопоток
// @Description Проксирует живой MJPEG видеопоток с разметкой от воркера к клиенту. Поддерживает токен в параметре запроса (?jwt=token).
// @Tags Counters
// @Produce multipart/x-mixed-replace
// @Security BearerAuth
// @Param id path int true "ID камеры"
// @Param jwt query string false "JWT Token (альтернатива заголовку Authorization)"
// @Success 200 {string} string "MJPEG Stream"
// @Failure 400 {string} string "Неверный ID"
// @Failure 502 {string} string "Не удалось подключиться к видеопотоку"
// @Router /api/counters/{id}/stream [get]
func (h *CounterHandler) StreamCounterVideo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	resp, err := h.manager.GetVideoStream(r.Context(), id)
	if err != nil {
		slog.Error("Ошибка получения стрима", "id", id, "error", err.Error())
		http.Error(w, "Не удалось подключиться к видеопотоку", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "no-cache, private")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	_, err = io.Copy(w, resp.Body)

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
			slog.Info("Клиент отключился от стрима", "id", id)
		} else {
			slog.Warn("Стрим прервался", "id", id, "error", err.Error())
		}
	}
}

func parseDateFlexible(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05", // Формат без таймзоны
		"2006-01-02",          // Просто дата
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("неподдерживаемый формат даты")
}

// GetStats godoc
// @Summary Получить статистику счетчика
// @Description Возвращает агрегированную историю проходов для камеры, сгруппированную по указанному периоду и ID группы.
// @Tags Counters
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID камеры"
// @Param period query string false "Период агрегации (hour, day, week, month, year). По умолчанию 'day'"
// @Param date_start query string false "Начало диапазона (например: 2026-03-01T00:00:00Z)"
// @Param date_end query string false "Конец диапазона (например: 2026-03-31T23:59:59Z)"
// @Success 200 {object} models.APIStatsResponse
// @Failure 400 {string} string "Неверный запрос"
// @Failure 500 {string} string "Ошибка формирования статистики"
// @Router /api/counters/{id}/stats [get]
func (h *CounterHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	counterID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID камеры", http.StatusBadRequest)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	var dateStart, dateEnd *time.Time

	if ds := r.URL.Query().Get("date_start"); ds != "" {
		t, err := parseDateFlexible(ds)
		if err != nil {
			http.Error(w, "Неверный формат date_start (ожидается YYYY-MM-DDTHH:MM:SS или RFC3339)", http.StatusBadRequest)
			return
		}
		dateStart = &t
	}

	if de := r.URL.Query().Get("date_end"); de != "" {
		t, err := parseDateFlexible(de)
		if err != nil {
			http.Error(w, "Неверный формат date_end (ожидается YYYY-MM-DDTHH:MM:SS или RFC3339)", http.StatusBadRequest)
			return
		}
		dateEnd = &t
	}

	records, err := h.manager.GetCounterStats(r.Context(), counterID, period, dateStart, dateEnd)
	if err != nil {
		slog.Error("Ошибка получения статистики", "counter_id", counterID, "error", err.Error())
		http.Error(w, "Ошибка формирования статистики: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if records == nil {
		records = []models.CounterStatsRecord{}
	}

	response := models.APIStatsResponse{
		CounterID: counterID,
		Period:    period,
		Data:      records,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
