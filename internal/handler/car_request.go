package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/InatoInato/car_service.git/internal/handler/dto"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxCarRequestBytes = 128 << 10

func decodeCarRequest(w http.ResponseWriter, r *http.Request) (dto.CreateCarRequest, pgtype.Numeric, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCarRequestBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return dto.CreateCarRequest{}, pgtype.Numeric{}, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return dto.CreateCarRequest{}, pgtype.Numeric{}, errors.New("expected JSON object")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return dto.CreateCarRequest{}, pgtype.Numeric{}, err
	}
	var req dto.CreateCarRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return dto.CreateCarRequest{}, pgtype.Numeric{}, err
	}
	// Decode the price token separately. float64 cannot distinguish a missing
	// price from zero and can round away fractional cents before validation.
	priceDecoder := json.NewDecoder(bytes.NewReader(raw))
	opening, err := priceDecoder.Token()
	if err != nil || opening != json.Delim('{') {
		return dto.CreateCarRequest{}, pgtype.Numeric{}, errors.New("expected JSON object")
	}
	var priceRaw json.RawMessage
	for priceDecoder.More() {
		field, err := priceDecoder.Token()
		if err != nil {
			return dto.CreateCarRequest{}, pgtype.Numeric{}, err
		}
		name := field.(string) // A valid JSON object's key is always a string.
		var value json.RawMessage
		if err := priceDecoder.Decode(&value); err != nil {
			return dto.CreateCarRequest{}, pgtype.Numeric{}, err
		}
		if strings.EqualFold(name, "price") {
			if priceRaw != nil {
				return dto.CreateCarRequest{}, pgtype.Numeric{}, errors.New("duplicate price field")
			}
			priceRaw = value
		}
	}
	price, err := exactPrice(priceRaw)
	if err != nil {
		return dto.CreateCarRequest{}, pgtype.Numeric{}, err
	}
	return req, price, nil
}

func (h *CarHandler) writeRequestError(w http.ResponseWriter, err error) {
	var limit *http.MaxBytesError
	if errors.As(err, &limit) {
		h.writeError(w, http.StatusRequestEntityTooLarge, "car request must not exceed 128 KiB")
		return
	}
	if errors.Is(err, errInvalidPrice) {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeError(w, http.StatusBadRequest, "invalid request: expected one JSON object with correctly typed fields")
}
