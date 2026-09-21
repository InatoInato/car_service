-- Destructive: rolling back removes the optional listing details, not the cars.
ALTER TABLE cars
    DROP COLUMN description,
    DROP COLUMN model_generation,
    DROP COLUMN image;
