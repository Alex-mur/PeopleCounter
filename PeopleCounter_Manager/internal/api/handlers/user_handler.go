package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"PeopleCounter_Manager/internal/models"
	"PeopleCounter_Manager/internal/service"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	manager *service.UserManager
}

func NewUserHandler(manager *service.UserManager) *UserHandler {
	return &UserHandler{manager: manager}
}

// Create godoc
// @Summary Создать нового пользователя
// @Description Добавляет нового пользователя в систему (только для администраторов). Пароль хэшируется перед сохранением.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UserCreateRequest true "Данные нового пользователя"
// @Success 201 {object} map[string]int "{\"id\": 1}"
// @Failure 400 {string} string "Неверный формат JSON"
// @Failure 500 {string} string "Ошибка сервера (возможно логин уже занят)"
// @Router /api/users [post]
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.UserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	id, err := h.manager.CreateUser(r.Context(), req)
	if err != nil {
		slog.Error("Ошибка создания пользователя", "error", err)
		http.Error(w, "Ошибка сервера (возможно логин уже занят)", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// List godoc
// @Summary Получить список пользователей
// @Description Возвращает список всех зарегистрированных пользователей системы (без хэшей паролей).
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.User
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/users [get]
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.manager.ListUsers(r.Context())
	if err != nil {
		slog.Error("Ошибка получения списка пользователей", "error", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	if users == nil {
		users = []models.User{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Update godoc
// @Summary Обновить данные пользователя
// @Description Обновляет профиль пользователя и его роль. Если передан новый пароль, он будет захэширован и обновлен.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID пользователя"
// @Param request body models.UserUpdateRequest true "Обновленные данные пользователя"
// @Success 200 {object} map[string]string "{\"status\":\"success\"}"
// @Failure 400 {string} string "Неверный ID или формат JSON"
// @Failure 500 {string} string "Ошибка обновления"
// @Router /api/users/{id} [put]
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	var req models.UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	if err := h.manager.UpdateUser(r.Context(), id, req); err != nil {
		slog.Error("Ошибка обновления пользователя", "id", id, "error", err)
		http.Error(w, "Ошибка обновления", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// Delete godoc
// @Summary Удалить пользователя
// @Description Удаляет пользователя из системы. Все привязки пользователя к камерам удаляются каскадно.
// @Tags Users
// @Security BearerAuth
// @Param id path int true "ID пользователя"
// @Success 204 "Успешное удаление"
// @Failure 400 {string} string "Неверный ID"
// @Failure 500 {string} string "Ошибка удаления"
// @Router /api/users/{id} [delete]
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	if err := h.manager.DeleteUser(r.Context(), id); err != nil {
		slog.Error("Ошибка удаления пользователя", "id", id, "error", err)
		http.Error(w, "Ошибка удаления", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCounters godoc
// @Summary Получить доступные камеры пользователя
// @Description Возвращает массив ID камер, к которым данному пользователю предоставлен доступ.
// @Tags Users Access
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID пользователя"
// @Success 200 {object} map[string][]int "{\"counter_ids\": [1, 2, 3]}"
// @Failure 400 {string} string "Неверный ID"
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/users/{id}/counters [get]
func (h *UserHandler) GetCounters(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	ids, err := h.manager.GetUserCounterIDs(r.Context(), userID)
	if err != nil {
		slog.Error("Ошибка получения прав пользователя", "user_id", userID, "error", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	if ids == nil {
		ids = []int{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]int{"counter_ids": ids})
}

// SetCounters godoc
// @Summary Назначить камеры пользователю
// @Description Полностью перезаписывает список камер, доступных пользователю. Старые привязки удаляются.
// @Tags Users Access
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID пользователя"
// @Param request body models.UserCountersRequest true "Массив ID камер"
// @Success 200 {object} map[string]string "{\"status\":\"success\"}"
// @Failure 400 {string} string "Неверный ID или формат JSON"
// @Failure 500 {string} string "Ошибка при сохранении прав"
// @Router /api/users/{id}/counters [post]
func (h *UserHandler) SetCounters(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	var req models.UserCountersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	if err := h.manager.SetUserCounters(r.Context(), userID, req.CounterIDs); err != nil {
		slog.Error("Ошибка назначения прав пользователя", "user_id", userID, "error", err)
		http.Error(w, "Ошибка при сохранении прав (возможно переданы несуществующие ID камер)", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
