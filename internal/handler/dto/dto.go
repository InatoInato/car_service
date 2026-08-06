package dto

type CreateCarRequest struct {
	Brand          string  `json:"brand" example:"BMW"`
	Model          string  `json:"model" example:"X5"`
	ProductionYear int16   `json:"production_year" example:"2023"`
	Color          string  `json:"color" example:"Blue"`
	Price          float64 `json:"price" example:"49500"`
}

type CarResponse struct {
	ID             string  `json:"id" example:"23b12e8b-3995-4910-9a7b-55cda03ade9f"`
	Brand          string  `json:"brand" example:"BMW"`
	Model          string  `json:"model" example:"X5"`
	ProductionYear int16   `json:"production_year" example:"2023"`
	Color          string  `json:"color" example:"Blue"`
	Price          float64 `json:"price" example:"49500"`
	CreatedAt      string  `json:"created_at" example:"2026-08-06T12:00:00Z"`
	UpdatedAt      string  `json:"updated_at" example:"2026-08-06T12:00:00Z"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"car not found"`
}