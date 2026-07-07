package model

import (
	"time"

	"github.com/google/uuid"
)

type Car struct {
    ID        uuid.UUID
    Brand     string
    Model     string
    Year      int
    Color     string
    Price     float64
    CreatedAt time.Time
    UpdatedAt time.Time
}