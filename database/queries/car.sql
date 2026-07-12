-- name: CreateCar :one
INSERT INTO cars (
    id,
    brand,
    model,
    production_year,
    color,
    price,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING *;

-- name: GetCarByID :one
SELECT
    id,
    brand,
    model,
    production_year,
    color,
    price,
    created_at,
    updated_at
FROM cars
WHERE id = $1;

-- name: ListCars :many
SELECT id, brand, model, production_year, color, price, created_at, updated_at
FROM cars
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateCar :one
UPDATE cars
SET
    brand = $2,
    model = $3,
    production_year = $4,
    color = $5,
    price = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCar :exec
DELETE FROM cars
WHERE id = $1;

-- name: CarExists :one
SELECT EXISTS (
    SELECT 1
    FROM cars
    WHERE id = $1
);

-- name: CountCars :one
SELECT COUNT(*)
FROM cars;