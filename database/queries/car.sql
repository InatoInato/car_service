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

-- name: FilterCars :many
SELECT id, brand, model, production_year, color, price, created_at, updated_at
FROM cars
WHERE (
    sqlc.arg('name')::text = ''
    OR STRPOS(LOWER(brand), LOWER(sqlc.arg('name')::text)) > 0
    OR STRPOS(LOWER(model), LOWER(sqlc.arg('name')::text)) > 0
    OR STRPOS(LOWER(CONCAT_WS(' ', brand, model)), LOWER(sqlc.arg('name')::text)) > 0
)
AND (sqlc.narg('year')::smallint IS NULL OR production_year = sqlc.narg('year')::smallint)
AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from')::timestamptz)
AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to')::timestamptz)
AND (sqlc.narg('min_price')::numeric IS NULL OR price >= sqlc.narg('min_price')::numeric)
AND (sqlc.narg('max_price')::numeric IS NULL OR price <= sqlc.narg('max_price')::numeric)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit_count') OFFSET sqlc.arg('offset_count');

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

-- name: CountFilteredCars :one
SELECT COUNT(*)
FROM cars
WHERE (
    sqlc.arg('name')::text = ''
    OR STRPOS(LOWER(brand), LOWER(sqlc.arg('name')::text)) > 0
    OR STRPOS(LOWER(model), LOWER(sqlc.arg('name')::text)) > 0
    OR STRPOS(LOWER(CONCAT_WS(' ', brand, model)), LOWER(sqlc.arg('name')::text)) > 0
)
AND (sqlc.narg('year')::smallint IS NULL OR production_year = sqlc.narg('year')::smallint)
AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from')::timestamptz)
AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to')::timestamptz)
AND (sqlc.narg('min_price')::numeric IS NULL OR price >= sqlc.narg('min_price')::numeric)
AND (sqlc.narg('max_price')::numeric IS NULL OR price <= sqlc.narg('max_price')::numeric);
