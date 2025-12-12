package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type BackendConfig struct {
	URL        string `json:"url"`
	Weight     int    `json:"weight"`
	HealthPath string `json:"health_path"` // NEW
}

type Config struct {
	ServerPort          string          `json:"server_port"`
	HealthCheckInterval string          `json:"health_check_interval"`
	Strategy            string          `json:"lb_strategy"`
	Backends            []BackendConfig `json:"backends"`
}

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
	ActiveConn   int64
	Weight       int
	HealthPath   string // NEW: Store specific health path
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.Alive
}

type ServerPool struct {
	backends []*Backend
	mux      sync.RWMutex // NEW: Protects the slice during Add/Remove
	current  uint64
	strategy string
}
