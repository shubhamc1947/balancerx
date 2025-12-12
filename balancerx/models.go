package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

// --- CONFIGURATION STRUCTS ---
type BackendConfig struct {
	URL    string `json:"url"`
	Weight int    `json:"weight"`
}

type Config struct {
	ServerPort          string          `json:"server_port"`
	HealthCheckInterval string          `json:"health_check_interval"`
	Strategy            string          `json:"lb_strategy"`
	Backends            []BackendConfig `json:"backends"`
}

// --- BACKEND STRUCT ---
type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
	ActiveConn   int64 // For Least Connection
	Weight       int   // For Weighted Round Robin
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

// --- SERVER POOL ---
type ServerPool struct {
	backends []*Backend
	current  uint64 // For Round Robin
	strategy string
}

func (s *ServerPool) AddBackend(backend *Backend) {
	s.backends = append(s.backends, backend)
}
