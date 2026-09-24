package router

import (
	"log/slog"

	"github.com/InatoInato/car_service.git/internal/handler"
	"github.com/InatoInato/car_service.git/internal/middleware"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// A nil generation service disables suggestions; CRUD remains available.
func New(logger *slog.Logger, carStore handler.CarStore, generationService *service.GenerationService, pinger handler.HealthPinger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logging(logger))

	carHandler := handler.NewCarHandler(carStore, logger)
	healthHandler := handler.NewHealthHandler(pinger, logger)
	generationHandler := handler.NewGenerationHandler(generationService, logger)

	r.Route("/cars", func(r chi.Router) {
		r.Get("/generations", generationHandler.Suggest)
		r.Post("/", carHandler.Create)
		r.Get("/", carHandler.List)
		r.Get("/{id}", carHandler.GetByID)
		r.Put("/{id}", carHandler.Update)
		r.Delete("/{id}", carHandler.Delete)
	})

	r.Get("/health", healthHandler.Health)

	r.Get("/ping", healthHandler.Ping)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	return r
}
