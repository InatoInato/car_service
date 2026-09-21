package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/InatoInato/car_service.git/internal/generation"
)

var ErrGenerationQuery = errors.New("brand and model must be nonblank text of at most 100 characters; year must be between 1886 and 2100")

// GenerationSource is the only replaceable boundary. It returns the known
// generations for a make/model; the service applies inclusive year matching.
type GenerationSource interface {
	Generations(context.Context, string, string) ([]generation.Candidate, error)
}

type GenerationService struct{ source GenerationSource }

func NewGenerationService(source GenerationSource) *GenerationService {
	return &GenerationService{source: source}
}

func (s *GenerationService) Suggest(ctx context.Context, q generation.Query) (generation.Result, error) {
	result := generation.Result{Candidates: []generation.Candidate{}, Notice: "Suggestions only. Catalogue coverage is limited; no match does not mean an invalid car. Years are inclusive calendar years, not exact build dates or market-specific model years. Choose the generation yourself."}
	for _, value := range []string{q.Brand, q.Model} {
		if !utf8.ValidString(value) || generation.IdentityKey(strings.TrimSpace(value)) == "" || utf8.RuneCountInString(value) > 100 || strings.ContainsFunc(value, unicode.IsControl) {
			return result, ErrGenerationQuery
		}
	}
	if q.Year < 1886 || q.Year > 2100 {
		return result, ErrGenerationQuery
	}
	if s.source == nil {
		return result, errors.New("generation source is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	candidates, err := s.source.Generations(ctx, strings.TrimSpace(q.Brand), strings.TrimSpace(q.Model))
	if err != nil {
		return result, fmt.Errorf("generation source: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	for _, candidate := range candidates {
		if candidate.ProductionYearStart <= q.Year && q.Year <= candidate.ProductionYearEnd {
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	sort.Slice(result.Candidates, func(i, j int) bool {
		a, b := result.Candidates[i], result.Candidates[j]
		if a.ProductionYearStart != b.ProductionYearStart {
			return a.ProductionYearStart < b.ProductionYearStart
		}
		return a.Generation < b.Generation
	})
	return result, nil
}
