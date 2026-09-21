package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/InatoInato/car_service.git/internal/generation"
	"github.com/InatoInato/car_service.git/internal/handler/dto"
	"github.com/InatoInato/car_service.git/internal/service"
)

type GenerationHandler struct {
	service *service.GenerationService
	logger  *slog.Logger
}

func NewGenerationHandler(s *service.GenerationService, logger *slog.Logger) *GenerationHandler {
	return &GenerationHandler{service: s, logger: logger}
}

// Suggest godoc
// @Summary Suggest model generations (advisory, limited coverage)
// @Tags Cars
// @Produce json
// @Param brand query string true "Manufacturer, as in car CRUD"
// @Param model query string true "Model"
// @Param year query int true "Calendar production year" minimum(1886) maximum(2100)
// @Success 200 {object} generation.Result
// @Failure 400 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /cars/generations [get]
func (h *GenerationHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	fail := func(status int, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(dto.ErrorResponse{Error: message})
	}
	query := r.URL.Query()
	year, err := strconv.Atoi(query.Get("year"))
	for _, key := range []string{"brand", "model", "year"} {
		if len(query[key]) != 1 {
			fail(400, "provide brand, model and year exactly once")
			return
		}
	}
	if err != nil {
		fail(400, service.ErrGenerationQuery.Error())
		return
	}
	if h.service == nil {
		fail(503, "generation suggestions are unavailable")
		return
	}
	result, err := h.service.Suggest(r.Context(), generation.Query{Brand: query.Get("brand"), Model: query.Get("model"), Year: year})
	if err != nil {
		if errors.Is(err, service.ErrGenerationQuery) {
			fail(400, err.Error())
			return
		}
		h.logger.ErrorContext(r.Context(), "generation suggestions failed", "error", err)
		fail(503, "generation suggestions are unavailable; you can still save the car")
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}
