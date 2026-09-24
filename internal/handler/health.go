package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type HealthPinger interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	pinger HealthPinger
	logger *slog.Logger
}

func NewHealthHandler(pinger HealthPinger, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{pinger: pinger, logger: logger}
}

// Ping godoc
//
//	@Summary		Ping
//	@Description	Simple ping endpoint
//	@Tags			System
//	@Produce		plain
//	@Success		200	{string}	string	"pong"
//	@Router			/ping [get]
func (h *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

// Health godoc
//
// @Summary      Health check
// @Description  Reports whether PostgreSQL is available.
// @Tags         System
// @Produce      json
// @Success      200 {object} map[string]string
// @Failure      503 {object} map[string]string
// @Router       /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.pinger == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := h.pinger.Ping(ctx); err != nil {
		h.logger.WarnContext(r.Context(), "health check failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
