// Package generation contains suggestion types, independent of storage and HTTP.
package generation

import (
	"strings"
	"unicode"
)

type Query struct {
	Brand string
	Model string
	Year  int
}

type Candidate struct {
	Generation          string `json:"generation"`
	ProductionYearStart int    `json:"production_year_start"`
	ProductionYearEnd   int    `json:"production_year_end"`
	BodyStyle           string `json:"body_style"`
	SourceURL           string `json:"source_url"`
}

type Result struct {
	Candidates []Candidate `json:"candidates"`
	Notice     string      `json:"notice"`
}

// IdentityKey tolerates case, spacing and hyphens, not typos or model prefixes.
// E280 and E 280 match; E280 CDI and E280 do not.
func IdentityKey(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' {
			return -1
		}
		return unicode.ToLower(r)
	}, value)
}
