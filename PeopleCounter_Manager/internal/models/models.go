package models

import (
	"time"
)

type Config struct {
	APIPort      int    `json:"api_port"`
	JWTSecret    string `json:"jwt_secret"`
	DBConnString string `json:"db_conn_string"`
	KeepLogDays  int    `json:"keep_log_days"`
}

type User struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Description string `json:"description"`
	Role        string `json:"role"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LinesGroup struct {
	ID          int    `json:"id"`
	Description string `json:"description,omitempty"`
}

type Line struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Start   []int  `json:"start"` // [x, y]
	End     []int  `json:"end"`   // [x, y]
	GroupID int    `json:"group_id"`
}

type CounterService struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	APIUrl      string    `json:"api_url"`
	APIKey      string    `json:"api_key"`
	CreatedAt   time.Time `json:"created_at"`
}

type Counter struct {
	ID          int          `json:"id"`
	ServiceID   int          `json:"service_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Url         string       `json:"url"`
	VidStride   int          `json:"vid_stride"`
	Lines       []Line       `json:"lines"`
	Groups      []LinesGroup `json:"groups"`
	CreatedAt   time.Time    `json:"created_at"`
}

type CounterStatsResponse struct {
	CounterID         int                  `json:"counter_id"`
	AggregationPeriod string               `json:"aggregation_period"`
	History           []CounterStatsRecord `json:"history"`
	Realtime          []CounterStatsRecord `json:"realtime_current_hour"`
}

type CounterStatsRecord struct {
	Period  *string `json:"period,omitempty"`
	GroupID int     `json:"group_id"`
	Passes  int     `json:"passes"`
}

type APIStatsResponse struct {
	CounterID int                  `json:"counter_id"`
	Period    string               `json:"period"`
	Data      []CounterStatsRecord `json:"data"`
}

type UserCreateRequest struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Description string `json:"description"`
	Password    string `json:"password"`
	Role        string `json:"role"` // "admin" или "viewer"
}

type UserUpdateRequest struct {
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	Description string  `json:"description"`
	Role        string  `json:"role"`
	Password    *string `json:"password,omitempty"` // Указатель: если nil, пароль не меняется
}

type UserCountersRequest struct {
	CounterIDs []int `json:"counter_ids"`
}
