package storage

import (
	"PeopleCounter_Manager/internal/models"
	"context"
	"errors"
)

func (s *Storage) CreateUser(ctx context.Context, u models.UserCreateRequest, passwordHash string) (int, error) {
	var id int
	query := `
		INSERT INTO users (login, name, email, description, password_hash, role)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	err := s.pool.QueryRow(ctx, query,
		u.Login, u.Name, u.Email, u.Description, passwordHash, u.Role,
	).Scan(&id)

	return id, err
}

func (s *Storage) GetAllUsers(ctx context.Context) ([]models.User, error) {
	query := `SELECT id, login, name, email, description, role FROM users`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Login, &u.Name, &u.Email, &u.Description, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *Storage) UpdateUser(ctx context.Context, id int, u models.UserUpdateRequest) error {
	query := `
		UPDATE users 
		SET name = $1, email = $2, description = $3, role = $4
		WHERE id = $5`

	tag, err := s.pool.Exec(ctx, query, u.Name, u.Email, u.Description, u.Role, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("пользователь не найден")
	}
	return nil
}

func (s *Storage) UpdateUserPassword(ctx context.Context, id int, passwordHash string) error {
	tag, err := s.pool.Exec(ctx, "UPDATE users SET password_hash = $1 WHERE id = $2", passwordHash, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("пользователь не найден")
	}
	return nil
}

func (s *Storage) DeleteUser(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("пользователь не найден")
	}
	return nil
}

func (s *Storage) SetUserCounters(ctx context.Context, userID int, counterIDs []int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM user_counters WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

	if len(counterIDs) > 0 {
		query := `
			INSERT INTO user_counters (user_id, counter_id)
			SELECT $1, unnest($2::integer[])
		`
		_, err = tx.Exec(ctx, query, userID, counterIDs)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Storage) GetUserCounterIDs(ctx context.Context, userID int) ([]int, error) {
	query := `SELECT counter_id FROM user_counters WHERE user_id = $1`
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
