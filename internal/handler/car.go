package handler

import (
	"encoding/json"
	"net/http"
)

type CarHandler struct{}

func NewCarHandler() *CarHandler {
	return &CarHandler{}
}

func (h *CarHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := map[string]any{
		"cars": []any{},
	}

	_ = json.NewEncoder(w).Encode(resp)
}