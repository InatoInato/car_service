package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/InatoInato/car_service.git/internal/handler/dto"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CarStore interface {
	CreateCar(context.Context, db.CreateCarParams) (db.Car, error)
	GetCarByID(context.Context, uuid.UUID) (db.Car, error)
	FilterCars(context.Context, db.FilterCarsParams) ([]db.Car, error)
	CountFilteredCars(context.Context, db.CountFilteredCarsParams) (int64, error)
	UpdateCar(context.Context, db.UpdateCarParams) (db.Car, error)
	DeleteCar(context.Context, uuid.UUID) error
}

type CarHandler struct {
	store  CarStore
	logger *slog.Logger
}

func NewCarHandler(store CarStore, logger *slog.Logger) *CarHandler {
	return &CarHandler{store: store, logger: logger}
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
// @Failure      413 {object} dto.ErrorResponse
// @Router       /cars [post]
func (h *CarHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, price, err := decodeCarRequest(w, r)
	if err != nil {
		h.writeRequestError(w, err)
		return
	}
	if err := validateCarCore(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	params, err := createCarParams(req, price)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	car, err := h.store.CreateCar(r.Context(), params)
	if err != nil {
		h.writeStoreError(w, r, "create car", "failed to create car", err)
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

	car, err := h.store.GetCarByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "car not found")
			return
		}
		h.writeStoreError(w, r, "get car", "failed to fetch car", err)
		return
	}

	h.writeJSON(w, http.StatusOK, car)
}

// List returns all cars.
//
// @Summary      List cars
// @Tags         Cars
// @Produce      json
// @Param        page query int false "Page number" minimum(1)
// @Param        limit query int false "Items per page" minimum(1) maximum(100)
// @Param        name query string false "Case-insensitive brand or model search"
// @Param        year query int false "Exact production year" minimum(1886) maximum(2100)
// @Param        production_year query int false "Alias for year" minimum(1886) maximum(2100)
// @Param        created_from query string false "Created at or after (RFC3339)"
// @Param        created_to query string false "Created at or before (RFC3339)"
// @Param        created_at query string false "Exact creation time (RFC3339)"
// @Param        min_price query number false "Minimum price" minimum(0)
// @Param        max_price query number false "Maximum price" minimum(0)
// @Param        price query number false "Exact price" minimum(0)
// @Success      200 {object} dto.ListCarsResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /cars [get]
func (h *CarHandler) List(w http.ResponseWriter, r *http.Request) {
	params, page, limit, err := listCarsParams(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cars, err := h.store.FilterCars(r.Context(), params)
	if err != nil {
		h.writeStoreError(w, r, "list cars", "failed to fetch cars", err)
		return
	}
	total, err := h.store.CountFilteredCars(r.Context(), db.CountFilteredCarsParams{
		Name:        params.Name,
		Year:        params.Year,
		CreatedFrom: params.CreatedFrom,
		CreatedTo:   params.CreatedTo,
		MinPrice:    params.MinPrice,
		MaxPrice:    params.MaxPrice,
	})
	if err != nil {
		h.writeStoreError(w, r, "count cars", "failed to fetch cars", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"data":  cars,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

func listCarsParams(r *http.Request) (db.FilterCarsParams, int, int, error) {
	const maxInt32 = int64(1<<31 - 1)

	query := r.URL.Query()
	page := 1
	limit := 20

	if value := query.Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return db.FilterCarsParams{}, 0, 0, errors.New("invalid page")
		}
		page = parsed
	}

	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return db.FilterCarsParams{}, 0, 0, errors.New("limit must be between 1 and 100")
		}
		limit = parsed
	}

	if int64(page-1) > maxInt32/int64(limit) {
		return db.FilterCarsParams{}, 0, 0, errors.New("page is too large")
	}

	params := db.FilterCarsParams{
		Name:        strings.TrimSpace(query.Get("name")),
		LimitCount:  int32(limit),
		OffsetCount: int32((page - 1) * limit),
	}

	yearValue := firstQueryValue(r, "year", "production_year")
	if yearValue != "" {
		year, err := strconv.ParseInt(yearValue, 10, 16)
		if err != nil || year < 1886 || year > 2100 {
			return db.FilterCarsParams{}, 0, 0, errors.New("year must be between 1886 and 2100")
		}
		params.Year = pgtype.Int2{Int16: int16(year), Valid: true}
	}

	createdFromValue := query.Get("created_from")
	createdToValue := query.Get("created_to")
	exactCreatedValue := firstQueryValue(r, "created_at", "created_time")
	if exactCreatedValue != "" {
		if createdFromValue != "" || createdToValue != "" {
			return db.FilterCarsParams{}, 0, 0, errors.New("created_at cannot be combined with created_from or created_to")
		}
		createdFromValue = exactCreatedValue
		createdToValue = exactCreatedValue
	}

	createdFrom, err := timestampFilter(createdFromValue)
	if err != nil {
		return db.FilterCarsParams{}, 0, 0, errors.New("created_from must be an RFC3339 timestamp")
	}
	createdTo, err := timestampFilter(createdToValue)
	if err != nil {
		return db.FilterCarsParams{}, 0, 0, errors.New("created_to must be an RFC3339 timestamp")
	}
	if createdFrom.Valid && createdTo.Valid && createdFrom.Time.After(createdTo.Time) {
		return db.FilterCarsParams{}, 0, 0, errors.New("created_from must not be after created_to")
	}
	params.CreatedFrom = createdFrom
	params.CreatedTo = createdTo

	minPriceValue := query.Get("min_price")
	maxPriceValue := query.Get("max_price")
	exactPriceValue := query.Get("price")
	if exactPriceValue != "" {
		if minPriceValue != "" || maxPriceValue != "" {
			return db.FilterCarsParams{}, 0, 0, errors.New("price cannot be combined with min_price or max_price")
		}
		minPriceValue = exactPriceValue
		maxPriceValue = exactPriceValue
	}

	minPrice, minPriceNumber, err := priceFilter(minPriceValue)
	if err != nil {
		return db.FilterCarsParams{}, 0, 0, errors.New("min_price must be a non-negative number")
	}
	maxPrice, maxPriceNumber, err := priceFilter(maxPriceValue)
	if err != nil {
		return db.FilterCarsParams{}, 0, 0, errors.New("max_price must be a non-negative number")
	}
	if minPrice.Valid && maxPrice.Valid && minPriceNumber > maxPriceNumber {
		return db.FilterCarsParams{}, 0, 0, errors.New("min_price must not be greater than max_price")
	}
	params.MinPrice = minPrice
	params.MaxPrice = maxPrice

	return params, page, limit, nil
}

func firstQueryValue(r *http.Request, keys ...string) string {
	for _, key := range keys {
		if value := r.URL.Query().Get(key); value != "" {
			return value
		}
	}
	return ""
}

func timestampFilter(value string) (pgtype.Timestamptz, error) {
	if value == "" {
		return pgtype.Timestamptz{}, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: parsed, Valid: true}, nil
}

func priceFilter(value string) (pgtype.Numeric, float64, error) {
	if value == "" {
		return pgtype.Numeric{}, 0, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return pgtype.Numeric{}, 0, errors.New("invalid price")
	}

	var numeric pgtype.Numeric
	if err := numeric.Scan(value); err != nil {
		return pgtype.Numeric{}, 0, err
	}
	return numeric, parsed, nil
}

// Update updates a car.
//
// @Summary      Update car
// @Description  Replaces core fields. Omitted image/model_generation/description stay unchanged; null or blank clears them.
// @Tags         Cars
// @Accept       json
// @Produce      json
// @Param        id path string true "Car UUID"
// @Param        request body dto.CreateCarRequest true "Updated car"
// @Success      200 {object} dto.CarResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      413 {object} dto.ErrorResponse
// @Router       /cars/{id} [put]
func (h *CarHandler) Update(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid car id")
		return
	}

	payload, price, err := decodeCarRequest(w, r)
	if err != nil {
		h.writeRequestError(w, err)
		return
	}
	if err := validateCarCore(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	details, err := parseListingDetails(payload)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	params := db.UpdateCarParams{
		ID:             id,
		Brand:          payload.Brand,
		Model:          payload.Model,
		ProductionYear: payload.ProductionYear,
		Color:          payload.Color,
		Price:          price,
		Image:          details.image, SetImage: payload.Image.Present,
		ModelGeneration: details.generation, SetModelGeneration: payload.ModelGeneration.Present,
		Description: details.description, SetDescription: payload.Description.Present,
	}

	car, err := h.store.UpdateCar(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "car not found")
			return
		}
		h.writeStoreError(w, r, "update car", "failed to update car", err)
		return
	}

	h.writeJSON(w, http.StatusOK, car)
}

func createCarParams(req dto.CreateCarRequest, price pgtype.Numeric) (db.CreateCarParams, error) {
	details, err := parseListingDetails(req)
	if err != nil {
		return db.CreateCarParams{}, err
	}

	now := time.Now().UTC()
	return db.CreateCarParams{
		ID:              uuid.New(),
		Brand:           req.Brand,
		Model:           req.Model,
		ProductionYear:  req.ProductionYear,
		Color:           req.Color,
		Price:           price,
		Image:           details.image,
		ModelGeneration: details.generation,
		Description:     details.description,
		CreatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
	}, nil
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

	err = h.store.DeleteCar(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, r, "delete car", "failed to delete car", err)
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

func (h *CarHandler) writeStoreError(w http.ResponseWriter, r *http.Request, operation, message string, err error) {
	h.logger.ErrorContext(r.Context(), "car store request failed", "operation", operation, "error", err)
	h.writeError(w, http.StatusInternalServerError, message)
}
