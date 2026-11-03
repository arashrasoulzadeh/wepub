package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Addr        string
	DataDir     string
	Token       string
	MaxUploadMB int
}

func FromEnv() (Config, error) {
	c := Config{
		Addr:        envOr("WEPUB_ADDR", ":8080"),
		DataDir:     envOr("WEPUB_DATA_DIR", "./data"),
		Token:       os.Getenv("WEPUB_TOKEN"),
		MaxUploadMB: 32,
	}
	if v := os.Getenv("WEPUB_MAX_UPLOAD_MB"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("bad WEPUB_MAX_UPLOAD_MB: %w", err)
		}
		c.MaxUploadMB = n
	}
	if c.Token == "" {
		return Config{}, errors.New("WEPUB_TOKEN required")
	}
	return c, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
