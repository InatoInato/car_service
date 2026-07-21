package router

import (
	"log/slog"

	"github.com/InatoInato/car_service.git/internal/handler"
	"github.com/InatoInato/car_service.git/internal/middleware"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/go-chi/chi/v5"
)

func New(logger *slog.Logger, carService *service.CarService) *chi.Mux {
    r := chi.NewRouter()

    r.Use(middleware.Logging(logger))

    // Fixed: Passed the service dependency
    carHandler := handler.NewCarHandler(carService)
    healthHandler := handler.NewHealthHandler()

    r.Route("/cars", func(r chi.Router) {
        r.Post("/", carHandler.Create)       // Missing POST
        r.Get("/", carHandler.List)          // Missing GET all cars
        r.Get("/{id}", carHandler.GetByID)   // Missing GET by ID
        r.Put("/{id}", carHandler.Update)    // Missing PUT
        r.Delete("/{id}", carHandler.Delete) // Missing DELETE
    })

    r.Get("/health", healthHandler.Health) // Missing health check endpoint

    return r
}