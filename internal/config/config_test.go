package config

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDSNPreservesCredentials(t *testing.T) {
	for _, host := range []string{"localhost", "::1"} {
		for _, password := range []string{"normal", "a/b?c#d@e:% +$'\\"} {
			p := PostgresConfig{Host: host, Port: "5432", User: "user@name", Password: password, Database: "cars test", SSLMode: "disable"}
			parsed, err := pgxpool.ParseConfig(p.DSN())
			if err != nil {
				t.Fatal("pgx could not parse encoded connection settings")
			}
			c := parsed.ConnConfig
			if c.User != p.User || c.Password != p.Password || c.Database != p.Database || c.Host != p.Host || c.Port != 5432 {
				t.Fatal("connection settings changed while encoding")
			}
		}
	}
}
