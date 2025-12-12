package main

import (
	"context"
	"encoding/json" // Change 1: Use standard JSON package
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// --- CONFIGURATION STRUCTS ---
type Config struct {
	ServerPort          string   `json:"server_port"` // Change 2: JSON tags
	HealthCheckInterval string   `json:"health_check_interval"`
	Backends            []string `json:"backends"`
}

// --- LOAD BALANCER STRUCTS (Unchanged) ---
type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
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
	current  uint64
}

func (s *ServerPool) AddBackend(backend *Backend) {
	s.backends = append(s.backends, backend)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, 1) % uint64(len(s.backends)))
}

func (s *ServerPool) GetNextPeer() *Backend {
	next := s.NextIndex()
	l := len(s.backends) + next
	for i := next; i < l; i++ {
		idx := i % len(s.backends)
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

func (s *ServerPool) HealthCheck() {
	for _, b := range s.backends {
		status := "up"
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]", b.URL, status)
	}
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

func lb(w http.ResponseWriter, r *http.Request) {
	peer := serverPool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

var serverPool ServerPool

// --- HELPER: Load Config from JSON ---
func loadConfig(file string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	// Change 3: Use JSON Unmarshal
	err = json.Unmarshal(data, &config)
	return &config, err
}

func main() {
	// 1. Load the Configuration
	config, err := loadConfig("balancerx/config.json") // Pointing to .json
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	log.Printf("Loaded config: Port %s, Interval %s, Backends %d", config.ServerPort, config.HealthCheckInterval, len(config.Backends))

	// 2. Parse Health Check Interval
	interval, err := time.ParseDuration(config.HealthCheckInterval)
	if err != nil {
		log.Fatalf("Invalid duration in config: %v", err)
	}

	// 3. Initialize Server Pool from Config
	for _, serverURL := range config.Backends {
		u, err := url.Parse(serverURL)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(u)
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
			log.Printf("[%s] %s\n", u.Host, e.Error())
			retries := GetRetryFromContext(request)
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(request.Context(), Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}
			serverPool.GetNextPeer().SetAlive(false)
			lb(writer, request)
		}

		serverPool.AddBackend(&Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: proxy,
		})
	}

	// 4. Start Health Check Loop
	go func() {
		for {
			serverPool.HealthCheck()
			time.Sleep(interval)
		}
	}()

	// 5. Start Server
	server := http.Server{
		Addr:    config.ServerPort,
		Handler: http.HandlerFunc(lb),
	}

	log.Printf("Load Balancer started at %s\n", config.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

const Retry int = 0

func GetRetryFromContext(r *http.Request) int {
	if v := r.Context().Value(Retry); v != nil {
		return v.(int)
	}
	return 0
}
