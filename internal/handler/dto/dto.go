package dto

type CreateCarRequest struct {
	Brand           string       `json:"brand" example:"BMW"`
	Model           string       `json:"model" example:"X5"`
	ProductionYear  int16        `json:"production_year" example:"2023"`
	Color           string       `json:"color" example:"Blue"`
	Price           float64      `json:"price" example:"49500"`
	Image           OptionalText `json:"image,omitzero" swaggertype:"string" extensions:"x-nullable" maxLength:"2048" example:"https://images.example.com/car.jpg"`
	ModelGeneration OptionalText `json:"model_generation,omitzero" swaggertype:"string" extensions:"x-nullable" maxLength:"100" example:"W124 facelift"`
	Description     OptionalText `json:"description,omitzero" swaggertype:"string" extensions:"x-nullable" maxLength:"10000" example:"Well maintained. Service records available."`
}

type CarResponse struct {
	ID              string  `json:"id" example:"23b12e8b-3995-4910-9a7b-55cda03ade9f"`
	Brand           string  `json:"brand" example:"BMW"`
	Model           string  `json:"model" example:"X5"`
	ProductionYear  int16   `json:"production_year" example:"2023"`
	Color           string  `json:"color" example:"Blue"`
	Price           float64 `json:"price" example:"49500"`
	CreatedAt       string  `json:"created_at" example:"2026-08-06T12:00:00Z"`
	UpdatedAt       string  `json:"updated_at" example:"2026-08-06T12:00:00Z"`
	Image           *string `json:"image" extensions:"x-nullable"`
	ModelGeneration *string `json:"model_generation" extensions:"x-nullable"`
	Description     *string `json:"description" extensions:"x-nullable"`
}

type ListCarsResponse struct {
	Data  []CarResponse `json:"data"`
	Page  int           `json:"page" example:"1"`
	Limit int           `json:"limit" example:"20"`
	Total int64         `json:"total" example:"42"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"car not found"`
}
