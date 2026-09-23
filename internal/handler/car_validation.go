package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/InatoInato/car_service.git/internal/handler/dto"
	"github.com/jackc/pgx/v5/pgtype"
)

var errInvalidPrice = errors.New("price must be present and between 0 and 9999999999.99 in whole cents")

func validateCarCore(req *dto.CreateCarRequest) error {
	for _, field := range []struct {
		name  string
		value *string
		limit int
	}{
		{"brand", &req.Brand, 100},
		{"model", &req.Model, 100},
		{"color", &req.Color, 50},
	} {
		value := strings.TrimSpace(*field.value)
		if value == "" || !utf8.ValidString(value) || strings.ContainsRune(value, 0) || utf8.RuneCountInString(value) > field.limit {
			return errors.New(field.name + " must contain 1 to " + strconv.Itoa(field.limit) + " characters without NUL")
		}
		*field.value = value
	}
	if req.ProductionYear < 1886 || req.ProductionYear > 2100 {
		return errors.New("production_year must be between 1886 and 2100")
	}
	return nil
}

func exactPrice(raw json.RawMessage) (pgtype.Numeric, error) {
	if len(raw) == 0 || len(raw) > 64 || raw[0] == '"' || string(raw) == "null" {
		return pgtype.Numeric{}, errInvalidPrice
	}
	// Bound exponent size before using big.Rat, which otherwise could allocate
	// a huge integer for a tiny JSON body such as 1e100000000.
	if i := strings.IndexAny(string(raw), "eE"); i >= 0 {
		exponent, err := strconv.Atoi(string(raw[i+1:]))
		if err != nil || exponent < -20 || exponent > 20 {
			return pgtype.Numeric{}, errInvalidPrice
		}
	}
	value, ok := new(big.Rat).SetString(string(raw))
	if !ok || value.Sign() < 0 {
		return pgtype.Numeric{}, errInvalidPrice
	}
	cents := new(big.Rat).Mul(value, big.NewRat(100, 1))
	if !cents.IsInt() || cents.Num().Cmp(big.NewInt(999999999999)) > 0 {
		return pgtype.Numeric{}, errInvalidPrice
	}
	amount := cents.Num().Int64()
	var numeric pgtype.Numeric
	if err := numeric.Scan(fmt.Sprintf("%d.%02d", amount/100, amount%100)); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("parse price: %w", err)
	}
	return numeric, nil
}
