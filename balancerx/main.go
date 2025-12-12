package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

var serverPool ServerPool

// --- MANAGEMENT LOGIC ---

func (s *ServerPool) AddBackend(b *Backend) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.backends = append(s.backends, b)
}

func (s *ServerPool) RemoveBackend(targetURL string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	var newBackends []*Backend
	for _, b := range s.backends {
		if b.URL.String() != targetURL {
			newBackends = append(newBackends, b)
		}
	}
	s.backends = newBackends
}

func (s *ServerPool) GetBackendsInfo() []map[string]interface{} {
	s.mux.RLock()
	defer s.mux.RUnlock()

	var stats []map[string]interface{}
	for _, b := range s.backends {
		stats = append(stats, map[string]interface{}{
			"url":         b.URL.String(),
			"alive":       b.Alive,
			"active_conn": atomic.LoadInt64(&b.ActiveConn),
			"weight":      b.Weight,
		})
	}
	return stats
}

// --- API HANDLERS ---

func apiBackendsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(serverPool.GetBackendsInfo())

	case "POST":
		var req BackendConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		u, _ := url.Parse(req.URL)
		proxy := httputil.NewSingleHostReverseProxy(u)
		// Add error handler (same as in main)

		serverPool.AddBackend(&Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: proxy,
			Weight:       req.Weight,
			HealthPath:   req.HealthPath,
		})
		json.NewEncoder(w).Encode(map[string]string{"status": "added"})

	case "DELETE":
		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		serverPool.RemoveBackend(req.URL)
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
	}
}

// --- DASHBOARD UI ---
// Simple HTML template embedded in code for portability
const dashboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Load Balancer Dashboard</title>
    <style>
        body { font-family: sans-serif; padding: 20px; }
        .server { border: 1px solid #ccc; padding: 10px; margin: 5px; border-radius: 5px; }
        .up { background-color: #d4edda; }
        .down { background-color: #f8d7da; }
        .form-group { margin-bottom: 10px; }
    </style>
</head>
<body>
    <h1>BalancerX Dashboard</h1>
    <div id="list"></div>

    <h3>Add Backend</h3>
    <div class="form-group"><input id="newUrl" placeholder="http://localhost:8084"></div>
    <div class="form-group"><input id="newWeight" placeholder="Weight (e.g. 1)" type="number"></div>
    <div class="form-group"><input id="newPath" placeholder="/health" value="/"></div>
    <button onclick="addBackend()">Add Server</button>

    <script>
        async function fetchStats() {
            const res = await fetch('/api/backends');
            const data = await res.json();
            const container = document.getElementById('list');
            container.innerHTML = '';
            data.forEach(s => {
                const div = document.createElement('div');
                div.className = 'server ' + (s.alive ? 'up' : 'down');
                div.innerHTML = '<strong>' + s.url + '</strong> | Alive: ' + s.alive + ' | Conns: ' + s.active_conn + 
                                ' <button onclick="removeBackend(\''+s.url+'\')" style="float:right;color:red">Remove</button>';
                container.appendChild(div);
            });
        }

        async function addBackend() {
            const url = document.getElementById('newUrl').value;
            const weight = parseInt(document.getElementById('newWeight').value);
            const path = document.getElementById('newPath').value;
            await fetch('/api/backends', {
                method: 'POST',
                body: JSON.stringify({url: url, weight: weight, health_path: path})
            });
            fetchStats();
        }

        async function removeBackend(url) {
            await fetch('/api/backends', {
                method: 'DELETE',
                body: JSON.stringify({url: url})
            });
            fetchStats();
        }

        setInterval(fetchStats, 2000); // Auto-refresh every 2s
        fetchStats();
    </script>
</body>
</html>
`

func uiHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))
	tmpl.Execute(w, nil)
}

// --- MAIN SERVER LOGIC ---

func lbHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Intercept UI requests
	if strings.HasPrefix(r.URL.Path, "/ui") {
		uiHandler(w, r)
		return
	}
	// 2. Intercept API requests
	if strings.HasPrefix(r.URL.Path, "/api/backends") {
		apiBackendsHandler(w, r)
		return
	}

	// 3. Normal Load Balancer Logic
	peer := serverPool.GetNextPeer(r) // Make sure Strategies.go is updated to take 'r'
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

	// Init Backends from Config
	for _, bConfig := range config.Backends {
		u, err := url.Parse(bConfig.URL)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(u)
		// ... (Add Proxy Error Handler Here as before) ...

		serverPool.AddBackend(&Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: proxy,
			Weight:       bConfig.Weight,
			HealthPath:   bConfig.HealthPath,
		})
	}

	// Health Check Loop
	go func() {
		interval, _ := time.ParseDuration(config.HealthCheckInterval)
		for {
			serverPool.mux.RLock() // Lock list while iterating
			// We iterate a copy or index to avoid holding lock too long during HTTP calls?
			// Better: iterate list, collect work, release lock, do checks, relock to update.
			// For simplicity in this demo: we just iterate directly.
			for _, b := range serverPool.backends {
				alive := isBackendAlive(b.URL, b.HealthPath) // Pass custom path
				b.SetAlive(alive)
			}
			serverPool.mux.RUnlock()
			time.Sleep(interval)
		}
	}()

	server := http.Server{
		Addr:    config.ServerPort,
		Handler: http.HandlerFunc(lbHandler),
	}

	log.Printf("Load Balancer (%s) started on %s", config.Strategy, config.ServerPort)
	log.Printf("Dashboard available at http://localhost%s/ui", config.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
