package router

import (
	"log/slog"

	"github.com/InatoInato/car_service.git/internal/handler"
	"github.com/go-chi/chi/v5"
)

func New(logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	healthHandler := handler.NewHealthHandler()
	carHandler := handler.NewCarHandler()

	r.Get("/health", healthHandler.Health)

	r.Route("/cars", func(r chi.Router) {
		r.Get("/", carHandler.List)
	})

	return r
}