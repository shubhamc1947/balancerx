package main

import (
	"context"
	"encoding/json"
	"net"
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

func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Retry Context Helpers
const Retry int = 0

func GetRetryFromContext(ctx context.Context) int {
	if v := ctx.Value(Retry); v != nil {
		return v.(int)
	}
	return 0
}
