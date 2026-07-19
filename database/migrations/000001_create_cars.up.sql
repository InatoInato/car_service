CREATE TABLE IF NOT EXISTS cars (
    id UUID PRIMARY KEY,

    brand VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,

    production_year SMALLINT NOT NULL
        CHECK (production_year BETWEEN 1886 AND 2100),

    color VARCHAR(50) NOT NULL,

    price NUMERIC(12,2) NOT NULL
        CHECK (price >= 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cars_production_year
    ON cars (production_year);
