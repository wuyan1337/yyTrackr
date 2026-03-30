package config

import (
	"os"
)

type Config struct {
	DatabasePath string
	Port         string
	Environment  string
	DevPreview   bool
}

func Load() *Config {
	dbPath := firstEnv("./data/subtrackr.db", "DATABASE_PATH", "DB_PATH")

	return &Config{
		DatabasePath: dbPath,
		Port:         getEnv("PORT", "8080"),
		Environment:  getEnv("GIN_MODE", "debug"),
		DevPreview:   getEnv("DEV_PREVIEW", "false") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func firstEnv(defaultValue string, keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return defaultValue
}
