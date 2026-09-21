package provider

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/InatoInato/car_service.git/internal/generation"
)

//go:embed generations.json
var catalogueJSON string

type catalogueEntry struct {
	Brand string `json:"brand"`
	Model string `json:"model"`
	generation.Candidate
}

// GenerationCatalog is immutable after construction. Each lookup returns a
// fresh slice so concurrent callers cannot modify the shared catalogue.
type GenerationCatalog struct{ entries []catalogueEntry }

func NewGenerationCatalog() (*GenerationCatalog, error) { return parseGenerationCatalog(catalogueJSON) }

func parseGenerationCatalog(data string) (*GenerationCatalog, error) {
	var entries []catalogueEntry
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		return nil, fmt.Errorf("decode generation catalogue: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("generation catalogue is empty")
	}
	seen := make(map[string]bool)
	for i, entry := range entries {
		u, err := url.Parse(entry.SourceURL)
		if generation.IdentityKey(entry.Brand) == "" || generation.IdentityKey(entry.Model) == "" || strings.TrimSpace(entry.Generation) == "" || entry.BodyStyle == "" || entry.ProductionYearStart < 1886 || entry.ProductionYearEnd > 2100 || entry.ProductionYearStart > entry.ProductionYearEnd || err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return nil, fmt.Errorf("invalid generation catalogue entry %d", i)
		}
		key := fmt.Sprintf("%s|%s|%s|%s|%d|%d", generation.IdentityKey(entry.Brand), generation.IdentityKey(entry.Model), entry.Generation, entry.BodyStyle, entry.ProductionYearStart, entry.ProductionYearEnd)
		if seen[key] {
			return nil, fmt.Errorf("duplicate generation catalogue entry %d", i)
		}
		seen[key] = true
	}
	return &GenerationCatalog{entries: entries}, nil
}

func (c *GenerationCatalog) Generations(ctx context.Context, brand, model string) ([]generation.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	brand, model = generation.IdentityKey(brand), generation.IdentityKey(model)
	result := []generation.Candidate{}
	for _, entry := range c.entries {
		if generation.IdentityKey(entry.Brand) == brand && generation.IdentityKey(entry.Model) == model {
			result = append(result, entry.Candidate)
		}
	}
	return result, nil
}
