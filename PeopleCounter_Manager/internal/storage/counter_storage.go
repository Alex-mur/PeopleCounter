package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"PeopleCounter_Manager/internal/models"
)

func (s *Storage) CreateService(ctx context.Context, c models.CounterService) (int, error) {
	var id int
	query := `
		INSERT INTO counter_services (name, description, api_url, api_key, created_at)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := s.pool.QueryRow(ctx, query, c.Name, c.Description, c.APIUrl, c.APIKey, time.Now()).Scan(&id)
	return id, err
}

func (s *Storage) UpdateService(ctx context.Context, id int, c models.CounterService) error {
	query := `
		UPDATE counter_services 
		SET name = $1, description = $2, api_url = $3, api_key = $4
		WHERE id = $5`

	tag, err := s.pool.Exec(ctx, query,
		c.Name, c.Description, c.APIUrl, c.APIKey, id,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return context.DeadlineExceeded
	}

	return nil
}

func (s *Storage) GetAllServices(ctx context.Context) ([]models.CounterService, error) {
	query := `SELECT id, name, description, api_url, api_key, created_at FROM counter_services`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.CounterService
	for rows.Next() {
		var srv models.CounterService
		if err := rows.Scan(&srv.ID, &srv.Name, &srv.Description, &srv.APIUrl, &srv.APIKey, &srv.CreatedAt); err != nil {
			return nil, err
		}
		services = append(services, srv)
	}
	return services, nil
}

func (s *Storage) GetServiceByID(ctx context.Context, id int) (models.CounterService, error) {
	query := `
		SELECT id, name, description, api_url, api_key, created_at 
		FROM counter_services 
		WHERE id = $1`

	var srv models.CounterService
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&srv.ID, &srv.Name, &srv.Description, &srv.APIUrl, &srv.APIKey, &srv.CreatedAt,
	)

	if err != nil {
		return models.CounterService{}, err // Вернет pgx.ErrNoRows, если запись не найдена
	}

	return srv, nil
}

func (s *Storage) DeleteService(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM counter_services WHERE id = $1", id)
	return err
}

func (s *Storage) CreateCounter(ctx context.Context, c models.Counter) (int, error) {
	linesJSON, _ := json.Marshal(c.Lines)
	groupsJSON, _ := json.Marshal(c.Groups)

	var id int
	query := `
		INSERT INTO counters (service_id, name, description, url, vid_stride, lines, groups, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	err := s.pool.QueryRow(ctx, query,
		c.ServiceID, c.Name, c.Description, c.Url, c.VidStride, linesJSON, groupsJSON, time.Now(),
	).Scan(&id)
	return id, err
}

func (s *Storage) GetAllCounters(ctx context.Context) ([]models.Counter, error) {
	query := `SELECT id, service_id, name, description, url, vid_stride, lines, groups, created_at FROM counters`
	return s.queryCounters(ctx, query)
}

func (s *Storage) GetUserCounters(ctx context.Context, userID int) ([]models.Counter, error) {
	query := `
		SELECT c.id, c.service_id, c.name, c.description, c.url, c.vid_stride, c.lines, c.groups, c.created_at
		FROM counters c
		JOIN user_counters uc ON c.id = uc.counter_id
		WHERE uc.user_id = $1`
	return s.queryCounters(ctx, query, userID)
}

func (s *Storage) GetCountersByServiceID(ctx context.Context, serviceID int) ([]models.Counter, error) {
	query := `SELECT id, service_id, name, description, url, vid_stride, lines, groups, created_at FROM counters WHERE service_id = $1`
	return s.queryCounters(ctx, query, serviceID)
}

func (s *Storage) queryCounters(ctx context.Context, query string, args ...any) ([]models.Counter, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counters []models.Counter
	for rows.Next() {
		var c models.Counter
		var linesJSON, groupsJSON []byte

		err := rows.Scan(&c.ID, &c.ServiceID, &c.Name, &c.Description, &c.Url, &c.VidStride, &linesJSON, &groupsJSON, &c.CreatedAt)
		if err != nil {
			return nil, err
		}

		_ = json.Unmarshal(linesJSON, &c.Lines)
		_ = json.Unmarshal(groupsJSON, &c.Groups)

		if c.Lines == nil {
			c.Lines = []models.Line{}
		}
		if c.Groups == nil {
			c.Groups = []models.LinesGroup{}
		}

		counters = append(counters, c)
	}
	return counters, nil
}

func (s *Storage) DeleteCounter(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM counters WHERE id = $1", id)
	return err
}

func (s *Storage) UpdateCounter(ctx context.Context, id int, c models.Counter) error {
	linesJSON, _ := json.Marshal(c.Lines)
	groupsJSON, _ := json.Marshal(c.Groups)

	query := `
		UPDATE counters 
		SET service_id = $1, name = $2, description = $3, url = $4, vid_stride=$5, lines = $6, groups = $7
		WHERE id = $8`

	tag, err := s.pool.Exec(ctx, query,
		c.ServiceID, c.Name, c.Description, c.Url, c.VidStride, linesJSON, groupsJSON, id,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return context.DeadlineExceeded
	}

	return nil
}

func (s *Storage) GetCounterByID(ctx context.Context, id int) (models.Counter, error) {
	query := `SELECT id, service_id, name, description, url, vid_stride, lines, groups, created_at FROM counters WHERE id = $1`

	counters, err := s.queryCounters(ctx, query, id)
	if err != nil {
		return models.Counter{}, err
	}

	if len(counters) == 0 {
		return models.Counter{}, context.DeadlineExceeded
	}

	return counters[0], nil
}

func (s *Storage) UpsertStats(ctx context.Context, counterID, groupID int, period time.Time, passes int) error {
	query := `
		INSERT INTO hourly_stats (counter_id, group_id, hour_bucket, passes)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (counter_id, group_id, hour_bucket)
		DO UPDATE SET passes = EXCLUDED.passes`

	_, err := s.pool.Exec(ctx, query, counterID, groupID, period, passes)
	return err
}

func (s *Storage) GetAggregatedStats(ctx context.Context, counterID int, period string, dateStart, dateEnd *time.Time) ([]models.CounterStatsRecord, error) {
	validPeriods := map[string]bool{
		"hour": true, "day": true, "week": true, "month": true, "year": true,
	}
	if !validPeriods[period] {
		return nil, fmt.Errorf("недопустимый период: %s", period)
	}

	baseQuery := fmt.Sprintf(`
		SELECT 
			date_trunc('%s', hour_bucket) AS bucket,
			group_id,
			SUM(passes) AS total_passes
		FROM hourly_stats
		WHERE counter_id = $1
	`, period)

	args := []interface{}{counterID}
	argId := 2

	if dateStart != nil {
		baseQuery += fmt.Sprintf(" AND hour_bucket >= $%d", argId)
		args = append(args, *dateStart)
		argId++
	}

	if dateEnd != nil {
		baseQuery += fmt.Sprintf(" AND hour_bucket <= $%d", argId)
		args = append(args, *dateEnd)
		argId++
	}

	baseQuery += `
		GROUP BY bucket, group_id
		ORDER BY bucket DESC, group_id ASC
	`
	rows, err := s.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.CounterStatsRecord
	for rows.Next() {
		var r models.CounterStatsRecord
		var bucket time.Time
		var totalPasses int

		if err := rows.Scan(&bucket, &r.GroupID, &totalPasses); err != nil {
			return nil, err
		}

		timeStr := bucket.Format(time.RFC3339)
		r.Period = &timeStr
		r.Passes = totalPasses

		records = append(records, r)
	}

	return records, nil
}
