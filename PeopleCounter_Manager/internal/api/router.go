package api

import (
	"PeopleCounter_Manager/internal/api/handlers"
	appMiddleware "PeopleCounter_Manager/internal/api/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth/v5"
)

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, claims, _ := jwtauth.FromContext(r.Context())
		if role, ok := claims["role"].(string); !ok || role != "admin" {
			http.Error(w, "Доступ запрещен: требуются права администратора", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewRouter(
	authHandler *handlers.AuthHandler,
	counterHandler *handlers.CounterHandler,
	usersHandler *handlers.UserHandler,
	tokenAuth *jwtauth.JWTAuth,
) *chi.Mux {
	r := chi.NewRouter()

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Документация API
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Публичные эндпоинты
	r.Post("/api/login", authHandler.Login)
	r.Post("/api/refresh", authHandler.Refresh)

	// Защищенные эндпоинты
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return jwtauth.Verify(tokenAuth, jwtauth.TokenFromQuery, jwtauth.TokenFromHeader, jwtauth.TokenFromCookie)(next)
		})
		r.Use(jwtauth.Authenticator(tokenAuth))
		r.Use(appMiddleware.RequireAccessToken)

		// Доступно всем (с учетом фильтрации внутри обработчика)
		r.Get("/api/user", authHandler.Profile)
		r.Get("/api/counters", counterHandler.ListCounters)
		r.Get("/api/counters/{id}/stream", counterHandler.StreamCounterVideo)
		r.Get("/api/counters/{id}/stats", counterHandler.GetStats)

		// Доступно ТОЛЬКО администраторам
		r.Group(func(r chi.Router) {
			r.Use(RequireAdmin)

			// Управление Воркерами (Services)
			r.Get("/api/services", counterHandler.ListServices)
			r.Get("/api/services/{id}", counterHandler.GetService)
			r.Post("/api/services", counterHandler.CreateService)
			r.Put("/api/services/{id}", counterHandler.UpdateService)
			r.Delete("/api/services/{id}", counterHandler.DeleteService)
			r.Post("/api/services/{id}/sync", counterHandler.SyncServiceCounters)

			// Управление Камерами (Counters)
			r.Post("/api/counters", counterHandler.CreateCounter)
			r.Delete("/api/counters/{id}", counterHandler.DeleteCounter)
			r.Put("/api/counters/{id}", counterHandler.UpdateCounter)

			// Управление Пользователями
			r.Get("/api/users", usersHandler.List)
			r.Post("/api/users", usersHandler.Create)
			r.Put("/api/users/{id}", usersHandler.Update)
			r.Delete("/api/users/{id}", usersHandler.Delete)
			r.Get("/api/users/{id}/counters", usersHandler.GetCounters)
			r.Post("/api/users/{id}/counters", usersHandler.SetCounters)
		})
	})

	return r
}
