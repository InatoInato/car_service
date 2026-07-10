package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
}

type ServerConfig struct {
	Port string
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using environment variables")
	}

	cfg := Config{
		Server: ServerConfig{
			Port: getEnv("APP_PORT", "8080"),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "postgres"),
			Database: getEnv("POSTGRES_DB", "cars"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
	}

	validate(cfg)

	return cfg
}

func validate(cfg Config) {
	if cfg.Server.Port == "" {
		log.Fatal("APP_PORT is required")
	}

	if cfg.Postgres.Host == "" {
		log.Fatal("POSTGRES_HOST is required")
	}

	if cfg.Postgres.Port == "" {
		log.Fatal("POSTGRES_PORT is required")
	}

	if cfg.Postgres.User == "" {
		log.Fatal("POSTGRES_USER is required")
	}

	if cfg.Postgres.Password == "" {
		log.Fatal("POSTGRES_PASSWORD is required")
	}

	if cfg.Postgres.Database == "" {
		log.Fatal("POSTGRES_DB is required")
	}

	if cfg.Postgres.SSLMode == "" {
		log.Fatal("POSTGRES_SSLMODE is required")
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}