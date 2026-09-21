package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/InatoInato/car_service.git/internal/handler/dto"
)

const maxCarRequestBytes = 128 << 10

func decodeCarRequest(w http.ResponseWriter, r *http.Request) (dto.CreateCarRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCarRequestBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	var req *dto.CreateCarRequest
	if err := decoder.Decode(&req); err != nil {
		return dto.CreateCarRequest{}, err
	}
	if req == nil {
		return dto.CreateCarRequest{}, errors.New("expected JSON object")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return dto.CreateCarRequest{}, err
	}
	return *req, nil
}

func (h *CarHandler) writeRequestError(w http.ResponseWriter, err error) {
	var limit *http.MaxBytesError
	if errors.As(err, &limit) {
		h.writeError(w, http.StatusRequestEntityTooLarge, "car request must not exceed 128 KiB")
		return
	}
	h.writeError(w, http.StatusBadRequest, "invalid request: expected one JSON object with correctly typed fields")
}
