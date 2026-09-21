package handler

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/InatoInato/car_service.git/internal/handler/dto"
	"github.com/jackc/pgx/v5/pgtype"
)

type listingDetails struct{ image, generation, description pgtype.Text }

func parseListingDetails(req dto.CreateCarRequest) (listingDetails, error) {
	var details listingDetails
	for _, field := range []struct {
		name   string
		value  *string
		max    int
		target *pgtype.Text
	}{
		{"image", req.Image.Value, 2048, &details.image},
		{"model_generation", req.ModelGeneration.Value, 100, &details.generation},
		{"description", req.Description.Value, 10000, &details.description},
	} {
		if field.value == nil {
			continue
		}
		value := strings.TrimSpace(*field.value)
		if value == "" {
			continue
		}
		if !utf8.ValidString(value) || strings.ContainsRune(value, 0) || utf8.RuneCountInString(value) > field.max {
			return details, fmt.Errorf("%s must be valid text of at most %d characters without NUL", field.name, field.max)
		}
		*field.target = pgtype.Text{String: value, Valid: true}
	}
	if details.image.Valid {
		value := details.image.String
		u, err := url.Parse(value)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" || strings.ContainsFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
			return details, fmt.Errorf("image must be an absolute HTTP(S) URL without credentials, spaces or a fragment")
		}
	}
	return details, nil
}
