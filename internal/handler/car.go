package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/InatoInato/car_service.git/internal/handler/dto"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CarHandler struct {
	service *service.CarService
}

func NewCarHandler(service *service.CarService) *CarHandler {
	return &CarHandler{service: service}
}

// Create creates a new car.
//
// @Summary      Create car
// @Description  Creates a new car in the database.
// @Tags         Cars
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateCarRequest true "Car data"
// @Success      201 {object} dto.CarResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /cars [post]
func (h *CarHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateCarRequest

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	params, err := createCarParams(req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid price")
		return
	}

	car, err := h.service.CreateCar(r.Context(), params)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, car)
}

// GetByID returns a car by ID.
//
// @Summary      Get car
// @Tags         Cars
// @Produce      json
// @Param        id path string true "Car UUID"
// @Success      200 {object} dto.CarResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Router       /cars/{id} [get]
func (h *CarHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid car id")
		return
	}

	car, err := h.service.GetCarByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "car not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, car)
}

// List returns all cars.
//
// @Summary      List cars
// @Tags         Cars
// @Produce      json
// @Success      200 {array} dto.CarResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /cars [get]
func (h *CarHandler) List(w http.ResponseWriter, r *http.Request) {
	cars, err := h.service.ListCars(r.Context(), 20, 0)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to fetch cars")
		return
	}
	h.writeJSON(w, http.StatusOK, cars)
}

// Update updates a car.
//
// @Summary      Update car
// @Tags         Cars
// @Accept       json
// @Produce      json
// @Param        id path string true "Car UUID"
// @Param        request body dto.CreateCarRequest true "Updated car"
// @Success      200 {object} dto.CarResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Router       /cars/{id} [put]
func (h *CarHandler) Update(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid car id")
		return
	}

	var payload dto.CreateCarRequest
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	price, err := numericPrice(payload.Price)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid price")
		return
	}

	params := db.UpdateCarParams{
		ID:             id,
		Brand:          payload.Brand,
		Model:          payload.Model,
		ProductionYear: payload.ProductionYear,
		Color:          payload.Color,
		Price:          price,
	}

	car, err := h.service.UpdateCar(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "car not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to update car")
		return
	}

	h.writeJSON(w, http.StatusOK, car)
}

func createCarParams(req dto.CreateCarRequest) (db.CreateCarParams, error) {
	price, err := numericPrice(req.Price)
	if err != nil {
		return db.CreateCarParams{}, err
	}

	now := time.Now().UTC()
	return db.CreateCarParams{
		ID:             uuid.New(),
		Brand:          req.Brand,
		Model:          req.Model,
		ProductionYear: req.ProductionYear,
		Color:          req.Color,
		Price:          price,
		CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
	}, nil
}

func numericPrice(price float64) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	if err := numeric.Scan(strconv.FormatFloat(price, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return numeric, nil
}

// Delete removes a car.
//
// @Summary      Delete car
// @Tags         Cars
// @Param        id path string true "Car UUID"
// @Success      204
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /cars/{id} [delete]
func (h *CarHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid car id")
		return
	}

	err = h.service.DeleteCar(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to delete car")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CarHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *CarHandler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}
