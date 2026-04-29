package service

import (
	"PeopleCounter_Manager/internal/models"
	"PeopleCounter_Manager/internal/storage"
	"context"
	"errors"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	storage   *storage.Storage
	tokenAuth *jwtauth.JWTAuth
}

func NewAuthService(store *storage.Storage, tokenAuth *jwtauth.JWTAuth) *AuthService {
	return &AuthService{storage: store, tokenAuth: tokenAuth}
}

func (s *AuthService) Login(ctx context.Context, login, password string) (string, string, error) {
	user, hash, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", "", errors.New("неверный логин или пароль")
	}
	return s.generateTokens(user.ID, user.Role)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	token, err := s.tokenAuth.Decode(refreshToken)
	if err != nil || token == nil {
		return "", "", errors.New("недействительный токен")
	}

	var tokenType string
	if err := token.Get("type", &tokenType); err != nil {
		return "", "", errors.New("отсутствует тип токена")
	}
	if tokenType != "refresh" {
		return "", "", errors.New("неверный тип токена")
	}

	var userIDFloat float64
	if err := token.Get("user_id", &userIDFloat); err != nil {
		return "", "", errors.New("отсутствует user_id")
	}
	userID := int(userIDFloat)

	role, err := s.storage.GetUserRoleByID(ctx, userID)
	if err != nil {
		return "", "", errors.New("пользователь не найден")
	}

	return s.generateTokens(userID, role)
}

func (s *AuthService) GetUserProfile(ctx context.Context, userID int) (models.User, error) {
	return s.storage.GetUserByID(ctx, userID)
}

func (s *AuthService) generateTokens(userID int, role string) (string, string, error) {
	accessClaims := map[string]interface{}{"user_id": userID, "role": role, "type": "access"}
	jwtauth.SetExpiry(accessClaims, time.Now().Add(1*time.Hour))
	_, accessToken, err := s.tokenAuth.Encode(accessClaims)
	if err != nil {
		return "", "", err
	}

	refreshClaims := map[string]interface{}{"user_id": userID, "type": "refresh"}
	jwtauth.SetExpiry(refreshClaims, time.Now().Add(7*24*time.Hour))
	_, refreshToken, err := s.tokenAuth.Encode(refreshClaims)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
