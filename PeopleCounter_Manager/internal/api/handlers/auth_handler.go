package handlers

import (
	"PeopleCounter_Manager/internal/models"
	"PeopleCounter_Manager/internal/service"
	"encoding/json"
	"github.com/go-chi/jwtauth/v5"
	"net/http"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// Login godoc
// @Summary Вход в систему (Авторизация)
// @Description Аутентифицирует пользователя по логину и паролю и возвращает пару JWT токенов (access и refresh).
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Учетные данные пользователя"
// @Success 200 {object} models.TokenResponse "Успешная авторизация"
// @Failure 400 {string} string "Неверный формат запроса"
// @Failure 401 {string} string "Неверный логин или пароль"
// @Router /api/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	accessToken, refreshToken, err := h.service.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Refresh godoc
// @Summary Обновление токена доступа
// @Description Принимает действующий refresh токен и возвращает новую пару токенов (access и refresh).
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.RefreshRequest true "Refresh токен"
// @Success 200 {object} models.TokenResponse "Новая пара токенов"
// @Failure 400 {string} string "Неверный формат запроса"
// @Failure 401 {string} string "Недействительный или истекший refresh токен"
// @Router /api/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат", http.StatusBadRequest)
		return
	}

	accessToken, refreshToken, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Profile godoc
// @Summary Получить профиль пользователя
// @Description Возвращает данные текущего авторизованного пользователя на основе его JWT токена.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User "Профиль пользователя"
// @Failure 401 {string} string "Не авторизован (отсутствует или неверный токен)"
// @Failure 404 {string} string "Пользователь не найден в БД"
// @Router /api/user [get]
func (h *AuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID := int(claims["user_id"].(float64))

	user, err := h.service.GetUserProfile(r.Context(), userID)
	if err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
