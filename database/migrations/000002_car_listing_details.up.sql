ALTER TABLE cars
    ADD COLUMN image TEXT CHECK (char_length(image) <= 2048),
    ADD COLUMN model_generation TEXT CHECK (char_length(model_generation) <= 100),
    ADD COLUMN description TEXT CHECK (char_length(description) <= 10000);
