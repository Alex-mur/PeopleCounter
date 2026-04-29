package storage

import (
	"PeopleCounter_Manager/internal/models"
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Storage struct {
	pool *pgxpool.Pool
}

func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

func (s *Storage) InitDB(ctx context.Context) {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		login VARCHAR(50) UNIQUE NOT NULL,
		name VARCHAR(100) NOT NULL,
		email VARCHAR(100),
		description TEXT,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(20) NOT NULL DEFAULT 'viewer',
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS counter_services (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		api_url VARCHAR(255) NOT NULL,
		api_key VARCHAR(100) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS counters (
		id SERIAL PRIMARY KEY,
		service_id INTEGER REFERENCES counter_services(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		url TEXT NOT NULL,
	    vid_stride INTEGER DEFAULT '4',
		lines JSONB DEFAULT '[]',
		groups JSONB DEFAULT '[]',
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS user_counters (
		user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
		counter_id INTEGER REFERENCES counters(id) ON DELETE CASCADE,
		assigned_at TIMESTAMPTZ DEFAULT NOW(),
		PRIMARY KEY (user_id, counter_id)
	);

	CREATE TABLE IF NOT EXISTS hourly_stats (
		counter_id INTEGER REFERENCES counters(id) ON DELETE CASCADE,
		group_id INTEGER NOT NULL,
		hour_bucket TIMESTAMPTZ NOT NULL,
		passes INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (counter_id, group_id, hour_bucket)
	);`

	if _, err := s.pool.Exec(ctx, schema); err != nil {
		log.Fatalf("Ошибка при создании таблиц: %v", err)
	}

	var count int
	s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), 12)
		s.pool.Exec(ctx, `
			INSERT INTO users (login, name, email, description, password_hash, role) 
			VALUES ($1, $2, $3, $4, $5, $6)`,
			"admin", "System Administrator", "admin@localhost", "Root user", string(hash), "admin")
		log.Println("Создан пользователь по умолчанию: login 'admin', password 'admin'")
	}
}

func (s *Storage) GetUserByLogin(ctx context.Context, login string) (models.User, string, error) {
	var user models.User
	var hash string
	err := s.pool.QueryRow(ctx,
		"SELECT id, role, password_hash FROM users WHERE login = $1", login).
		Scan(&user.ID, &user.Role, &hash)
	return user, hash, err
}

func (s *Storage) GetUserByID(ctx context.Context, id int) (models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		"SELECT id, login, name, email, description, role FROM users WHERE id = $1", id).
		Scan(&u.ID, &u.Login, &u.Name, &u.Email, &u.Description, &u.Role)
	return u, err
}

func (s *Storage) GetUserRoleByID(ctx context.Context, id int) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, "SELECT role FROM users WHERE id = $1", id).Scan(&role)
	return role, err
}
