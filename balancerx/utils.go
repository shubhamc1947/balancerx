package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"
)

func loadConfig(file string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &config)
	return &config, err
}

// NEW: Check Health via HTTP GET
func isBackendAlive(u *url.URL, healthPath string) bool {
	timeout := 2 * time.Second
	client := http.Client{
		Timeout: timeout,
	}
	// Construct full health URL (e.g., http://localhost:8081/health)
	fullURL := u.String() + healthPath

	resp, err := client.Get(fullURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Consider it healthy if status is 200 OK
	return resp.StatusCode == http.StatusOK
}

const Retry int = 0

func GetRetryFromContext(ctx context.Context) int {
	if v := ctx.Value(Retry); v != nil {
		return v.(int)
	}
	return 0
}
