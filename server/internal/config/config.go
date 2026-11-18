// Package config reads settings from the environment and optional .env files.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const (
	StoreMemory   = "memory"
	StorePostgres = "postgres"
)

type Config struct {
	// Which .env.<AppEnv> file to load, e.g. "production" -> .env.production.
	AppEnv string `env:"APP_ENV" envDefault:"development"`
	// Address to listen on. If unset: ":$PORT" when PORT is set (Render and similar
	// platforms inject PORT), otherwise ":8080".
	Addr string `env:"ADDR"`
	Port string `env:"PORT"`
	// PostgreSQL connection URL, e.g. postgresql://user:pass@host/db?sslmode=require.
	// Never log this: it contains the password.
	DBAddr string `env:"DB_ADDR"`
	// "memory" or "postgres". Empty means postgres when DB_ADDR is set, memory otherwise.
	Store string `env:"STORE"`
	// Secure flag on the session cookie. Turn off only for plain-HTTP local development.
	CookieSecure bool `env:"COOKIE_SECURE" envDefault:"true"`
	// Load the mock dataset at startup. Unset means: yes for memory, NO for postgres
	// (mock accounts have published passwords and must not reach a real database by accident).
	SeedMockData *bool `env:"SEED_MOCK_DATA"`
	// slog level: debug, info, warn, error.
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

var appEnvShape = regexp.MustCompile(`^[a-z0-9_-]+$`)

// Load reads, in order of precedence: real environment variables, .env.<APP_ENV>, .env.
// godotenv never overrides a variable that is already set, so the first source wins.
func Load() (Config, error) {
	appEnv := "development"
	if v := os.Getenv("APP_ENV"); v != "" {
		appEnv = v
	}
	if !appEnvShape.MatchString(appEnv) {
		return Config{}, fmt.Errorf("APP_ENV=%q must match %s", appEnv, appEnvShape)
	}
	for _, file := range []string{".env." + appEnv, ".env"} {
		if err := godotenv.Load(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return Config{}, fmt.Errorf("reading %s: %w", file, err)
		}
	}

	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, err
	}

	if c.Addr == "" {
		c.Addr = ":8080"
		if c.Port != "" {
			c.Addr = ":" + c.Port
		}
	}

	switch c.Store {
	case "":
		c.Store = StoreMemory
		if c.DBAddr != "" {
			c.Store = StorePostgres
		}
	case StoreMemory:
	case StorePostgres:
		if c.DBAddr == "" {
			return Config{}, errors.New("STORE=postgres needs DB_ADDR")
		}
	default:
		return Config{}, fmt.Errorf("STORE=%q is not supported (memory or postgres)", c.Store)
	}

	if c.SeedMockData == nil {
		seed := c.Store == StoreMemory
		c.SeedMockData = &seed
	}
	return c, nil
}
