package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

var serverPool ServerPool

func lb(w http.ResponseWriter, r *http.Request) {
	// Pass 'r' so Strategies like IP Hash can access IP
	peer := serverPool.GetNextPeer(r)

	if peer != nil {
		atomic.AddInt64(&peer.ActiveConn, 1)
		defer atomic.AddInt64(&peer.ActiveConn, -1)

		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func main() {
	config, err := loadConfig("balancerx/config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	serverPool.strategy = config.Strategy

	// Initialize Backends
	for _, bConfig := range config.Backends {
		u, err := url.Parse(bConfig.URL)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(u)

		// Custom Error Handler for Retries
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
			log.Printf("[%s] %s\n", u.Host, e.Error())
			retries := GetRetryFromContext(request.Context())
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(request.Context(), Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}
			// Mark as down if retries fail
			// Ideally, we'd find the specific backend in the pool to mark it down
			// For simplicity here, we rely on the HealthCheck loop to catch it shortly
		}

		serverPool.AddBackend(&Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: proxy,
			Weight:       bConfig.Weight,
		})
	}

	// Start Health Check
	go func() {
		interval, _ := time.ParseDuration(config.HealthCheckInterval)
		for {
			for _, b := range serverPool.backends {
				alive := isBackendAlive(b.URL)
				b.SetAlive(alive)
				status := "up"
				if !alive {
					status = "down"
				}
				// Log active connections for debugging
				log.Printf("[%s] %s | Active: %d", b.URL, status, atomic.LoadInt64(&b.ActiveConn))
			}
			time.Sleep(interval)
		}
	}()

	server := http.Server{
		Addr:    config.ServerPort,
		Handler: http.HandlerFunc(lb),
	}

	log.Printf("Load Balancer (%s) started on %s", config.Strategy, config.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
