package service

import (
	"PeopleCounter_Manager/internal/models"
	"PeopleCounter_Manager/internal/storage"
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type UserManager struct {
	storage *storage.Storage
}

func NewUserManager(store *storage.Storage) *UserManager {
	return &UserManager{storage: store}
}

func (m *UserManager) CreateUser(ctx context.Context, req models.UserCreateRequest) (int, error) {
	if req.Password == "" || req.Login == "" {
		return 0, errors.New("логин и пароль обязательны")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return 0, err
	}

	return m.storage.CreateUser(ctx, req, string(hash))
}

func (m *UserManager) ListUsers(ctx context.Context) ([]models.User, error) {
	return m.storage.GetAllUsers(ctx)
}

func (m *UserManager) UpdateUser(ctx context.Context, id int, req models.UserUpdateRequest) error {
	err := m.storage.UpdateUser(ctx, id, req)
	if err != nil {
		return err
	}

	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), 12)
		if err != nil {
			return err
		}
		err = m.storage.UpdateUserPassword(ctx, id, string(hash))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *UserManager) DeleteUser(ctx context.Context, id int) error {
	return m.storage.DeleteUser(ctx, id)
}

func (m *UserManager) SetUserCounters(ctx context.Context, userID int, counterIDs []int) error {
	return m.storage.SetUserCounters(ctx, userID, counterIDs)
}

func (m *UserManager) GetUserCounterIDs(ctx context.Context, userID int) ([]int, error) {
	return m.storage.GetUserCounterIDs(ctx, userID)
}
