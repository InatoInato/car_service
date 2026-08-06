package handler

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
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
// @Description  Reports whether the service is running.
// @Tags         System
// @Produce      json
// @Success      200 {object} map[string]string
// @Router       /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := map[string]string{
		"status": "ok",
	}

	_ = json.NewEncoder(w).Encode(resp)
}
