package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr                     string
	Token                    string
	ReadHeaderTimeout        time.Duration
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
	IdleTimeout              time.Duration
	MaxBodyBytes             int64
	DefaultMaxTemplatePasses int
}

func Load() (Config, error) {
	cfg := Config{
		Addr:                     getenv("ADDR", ":4355"),
		ReadHeaderTimeout:        getenvDuration("READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:              getenvDuration("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:             getenvDuration("WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:              getenvDuration("IDLE_TIMEOUT", 60*time.Second),
		MaxBodyBytes:             getenvInt64("MAX_BODY_BYTES", 1<<20),
		DefaultMaxTemplatePasses: getenvInt("MAX_TEMPLATE_PASSES", 4),
	}

	token := strings.TrimSpace(os.Getenv("PLUGIN_TOKEN"))
	if token == "" {
		tokenFile := getenv("TOKEN_FILE", "/var/run/argo/token")
		b, err := os.ReadFile(tokenFile)
		if err != nil {
			return Config{}, fmt.Errorf("read token file %q: %w", tokenFile, err)
		}
		token = strings.TrimSpace(string(b))
	}
	if token == "" {
		return Config{}, fmt.Errorf("plugin token is empty")
	}
	cfg.Token = token

	if cfg.DefaultMaxTemplatePasses <= 0 {
		return Config{}, fmt.Errorf("MAX_TEMPLATE_PASSES must be > 0")
	}
	if cfg.MaxBodyBytes <= 0 {
		return Config{}, fmt.Errorf("MAX_BODY_BYTES must be > 0")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}

func getenvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}
