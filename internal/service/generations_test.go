package service

import (
	"context"
	"errors"
	"testing"

	"github.com/InatoInato/car_service.git/internal/generation"
)

type generationSourceFunc func(context.Context, string, string) ([]generation.Candidate, error)

func (f generationSourceFunc) Generations(ctx context.Context, b, m string) ([]generation.Candidate, error) {
	return f(ctx, b, m)
}

func TestGenerationServiceContextAndFailure(t *testing.T) {
	want := errors.New("source unavailable")
	s := NewGenerationService(generationSourceFunc(func(ctx context.Context, b, m string) ([]generation.Candidate, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("missing bounded context")
		}
		if b != "Brand" || m != "Model" {
			t.Error("query was not trimmed")
		}
		return nil, want
	}))
	result, err := s.Suggest(context.Background(), generation.Query{Brand: " Brand ", Model: " Model ", Year: 1995})
	if !errors.Is(err, want) || result.Candidates == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s = NewGenerationService(generationSourceFunc(func(ctx context.Context, _, _ string) ([]generation.Candidate, error) { return nil, ctx.Err() }))
	if _, err = s.Suggest(ctx, generation.Query{Brand: "B", Model: "M", Year: 1995}); !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cancellation: %v", err)
	}
}
