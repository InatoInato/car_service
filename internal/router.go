package router

import (
	"log/slog"

	"github.com/InatoInato/car_service.git/internal/handler"
	"github.com/InatoInato/car_service.git/internal/middleware"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Main wires both services explicitly. A nil generation service disables only
// suggestions; CRUD does not depend on the suggestion provider.
func New(logger *slog.Logger, carService *service.CarService, generationService *service.GenerationService) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logging(logger))

	// Fixed: Passed the service dependency
	carHandler := handler.NewCarHandler(carService)
	healthHandler := handler.NewHealthHandler()
	generationHandler := handler.NewGenerationHandler(generationService, logger)

	r.Route("/cars", func(r chi.Router) {
		r.Get("/generations", generationHandler.Suggest)
		r.Post("/", carHandler.Create)       // Missing POST
		r.Get("/", carHandler.List)          // Missing GET all cars
		r.Get("/{id}", carHandler.GetByID)   // Missing GET by ID
		r.Put("/{id}", carHandler.Update)    // Missing PUT
		r.Delete("/{id}", carHandler.Delete) // Missing DELETE
	})

	r.Get("/health", healthHandler.Health) // Missing health check endpoint

	r.Get("/ping", healthHandler.Ping)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	return r
}
