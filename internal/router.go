package router

import (
	"log/slog"

	"github.com/InatoInato/car_service.git/internal/handler"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/go-chi/chi/v5"
)

func New(logger *slog.Logger, carService *service.CarService) *chi.Mux {
    r := chi.NewRouter()

    // Fixed: Passed the service dependency
    carHandler := handler.NewCarHandler(carService) 

    r.Route("/cars", func(r chi.Router) {
        r.Post("/", carHandler.Create)       // Missing POST
        r.Get("/", carHandler.List)
        r.Get("/{id}", carHandler.GetByID)   // Missing GET by ID
        r.Put("/{id}", carHandler.Update)    // Missing PUT
        r.Delete("/{id}", carHandler.Delete) // Missing DELETE
    })

    return r
}