package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	PostgresUser     string `env:"POSTGRES_USER"     envDefault:"postgres"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" envDefault:"postgres"`
	PostgresHost     string `env:"POSTGRES_HOST"     envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT"     envDefault:"5432"`
	PgDbName         string `env:"POSTGRES_DB_NAME"  envDefault:"filmmash_db"`

	TmdbApitoken string `env:"TMDB_API_TOKEN" envDefault:""`
}

func loadDotenvFile() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
}

func Load() *Config {
	loadDotenvFile()
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		panic(err)
	}
	return &cfg
}
