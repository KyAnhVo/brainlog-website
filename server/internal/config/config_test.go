package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// clearEnv unsets every variable Load reads, restoring them after the test.
func clearEnv(t *testing.T) {
	for _, k := range []string{"APP_ENV", "ADDR", "PORT", "DB_ADDR", "STORE", "COOKIE_SECURE", "SEED_MOCK_DATA", "LOG_LEVEL"} {
		t.Setenv(k, "") // registers restore
		os.Unsetenv(k)
	}
}

func TestProductionFile(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, ".env.production", "DB_ADDR=postgresql://u:p@db.example/app?sslmode=require\nADDR=:9090\n")
	writeFile(t, dir, ".env", "ADDR=:7070\nLOG_LEVEL=debug\n")
	t.Setenv("APP_ENV", "production")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":9090" {
		t.Errorf("ADDR = %q: .env.production should beat .env", c.Addr)
	}
	if c.LogLevel != "debug" {
		t.Errorf("LOG_LEVEL = %q: .env should still fill unset values", c.LogLevel)
	}
	if c.Store != StorePostgres || c.DBAddr == "" {
		t.Errorf("store = %q: DB_ADDR should select postgres", c.Store)
	}
	if *c.SeedMockData {
		t.Error("mock data must not be seeded into postgres by default")
	}
	if !c.CookieSecure {
		t.Error("COOKIE_SECURE should default to true")
	}
}

func TestRealEnvWinsAndDefaults(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, ".env.production", "ADDR=:9090\n")
	t.Setenv("APP_ENV", "production")
	t.Setenv("ADDR", ":1234")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":1234" {
		t.Errorf("ADDR = %q: real env should win", c.Addr)
	}
	if c.Store != StoreMemory || !*c.SeedMockData {
		t.Errorf("no DB_ADDR: store=%q seed=%v, want memory + seed", c.Store, *c.SeedMockData)
	}
}

func TestInvalid(t *testing.T) {
	for name, set := range map[string]map[string]string{
		"postgres without DB_ADDR": {"STORE": "postgres"},
		"unknown store":            {"STORE": "mysql"},
		"path in APP_ENV":          {"APP_ENV": "../secrets"},
	} {
		t.Run(name, func(t *testing.T) {
			clearEnv(t)
			t.Chdir(t.TempDir())
			for k, v := range set {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestPort(t *testing.T) {
	for _, tc := range []struct{ addr, port, want string }{
		{"", "", ":8080"},
		{"", "10000", ":10000"},                       // Render
		{"127.0.0.1:9000", "10000", "127.0.0.1:9000"}, // explicit ADDR wins
	} {
		clearEnv(t)
		t.Chdir(t.TempDir())
		if tc.addr != "" {
			t.Setenv("ADDR", tc.addr)
		}
		if tc.port != "" {
			t.Setenv("PORT", tc.port)
		}
		c, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if c.Addr != tc.want {
			t.Errorf("ADDR=%q PORT=%q -> %q, want %q", tc.addr, tc.port, c.Addr, tc.want)
		}
	}
}
