package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitLogsTable(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
	CREATE TABLE IF NOT EXISTS logs (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMPTZ DEFAULT NOW(),
		level VARCHAR(10) NOT NULL,
		message TEXT NOT NULL,
		attributes JSONB
	);
	CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);`

	_, err := pool.Exec(ctx, schema)
	return err
}

type DBHandler struct {
	pool *pgxpool.Pool
	opts slog.HandlerOptions
}

func NewDBHandler(pool *pgxpool.Pool, opts slog.HandlerOptions) *DBHandler {
	return &DBHandler{
		pool: pool,
		opts: opts,
	}
}

func (h *DBHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *DBHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	var attrsJSON []byte
	if len(attrs) > 0 {
		attrsJSON, _ = json.Marshal(attrs)
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, _ = h.pool.Exec(bgCtx,
			"INSERT INTO logs (timestamp, level, message, attributes) VALUES ($1, $2, $3, $4)",
			r.Time, r.Level.String(), r.Message, attrsJSON)
	}()

	return nil
}

func (h *DBHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *DBHandler) WithGroup(name string) slog.Handler {
	return h
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			_ = handler.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return NewMultiHandler(handlers...)
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return NewMultiHandler(handlers...)
}
