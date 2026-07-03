package config

import (
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Port             string `env:"PORT"              envDefault:"8080"`
	PostgresUser     string `env:"POSTGRES_USER"     envDefault:"postgres"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" envDefault:"postgres"`
	PostgresHost     string `env:"POSTGRES_HOST"     envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT"     envDefault:"5432"`
	PgDbName         string `env:"POSTGRES_DB_NAME"  envDefault:"filmmash_db"`

	TmdbApitoken string `env:"TMDB_API_TOKEN" envDefault:""`

	ZitadelClientId        string `env:"ZITADEL_CLIENT_ID"`
	ZitadelClientSecret    string `env:"ZITADEL_CLIENT_SECRET"`
	ZitadelBaseURL         string `env:"ZITADEL_BASE_URL"`
	ZitadelM2MClientId     string `env:"ZITADEL_M2M_CLIENT_ID"`
	ZitadelM2MClientSecret string `env:"ZITADEL_M2M_CLIENT_SECRET"`

	AppBaseURL string `env:"APP_BASE_URL" envDefault:"http://localhost:8080"`
}

func loadDotenvFile() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
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
