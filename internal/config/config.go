package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

const HTTPTimeout = 10 * time.Second

type Config struct {
	DatabaseURL   string
	APIAddr       string
	LogLevel      string
	CORSOrigins   []string
	AdminAPIToken string
	GCSBucket     string
	ImageDir      string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8083"
	}

	var origins []string
	if raw := os.Getenv("CORS_ORIGINS"); raw != "" {
		for _, o := range strings.Split(raw, ",") {
			if s := strings.TrimSpace(o); s != "" {
				origins = append(origins, s)
			}
		}
	}

	return &Config{
		DatabaseURL:   dbURL,
		APIAddr:       addr,
		LogLevel:      os.Getenv("LOG_LEVEL"),
		CORSOrigins:   origins,
		AdminAPIToken: os.Getenv("ADMIN_API_TOKEN"),
		GCSBucket:     os.Getenv("GCS_BUCKET"),
		ImageDir:      os.Getenv("IMAGE_DIR"),
	}, nil
}
