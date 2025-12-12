# **BalancerX**

> 🚀 *A lightweight, dynamic, and customizable Load Balancer built in Go (Golang) with multiple routing strategies, active health checks, real-time dashboard, and live backend management.*

---

## ✨ **Features**

### 🔄 **Multi-Strategy Routing**

BalancerX supports four load-balancing algorithms (configurable via JSON):

* **Round Robin** — simple cyclical distribution
* **Weighted Round Robin** — distributes traffic relative to server weight
* **Least Connections** — sends requests to the server with the fewest active connections
* **IP Hash** — sticky sessions based on client IP

---

### ⚙️ **Dynamic Configuration**

* Add/remove backend servers **without restarting** the load balancer
* Fully JSON-driven configuration

---

### ❤️ **Active Health Checks**

* Periodically probes each backend’s `/health` (or custom) endpoint
* Automatically removes unhealthy backends from rotation
* Restores them once they recover

---

### 🖥️ **Real-Time Dashboard**

UI available at:

```
http://localhost:8080/ui
```

View:

* Server health (green/red)
* Active connections
* Traffic routing
* Add/remove servers live

---

### 🧵 **Concurrency-Safe Architecture**

* `sync.RWMutex` for read-optimized backend pool
* `sync/atomic` for counters (active requests, RR index)
* Goroutines for parallel health checks & proxying

---

### 🔁 **Smart Retry Mechanism**

If a backend fails during a request:

1. Proxy detects failure
2. Tracks retry attempts
3. Automatically routes the request to next healthy server

---

## 📁 **Project Structure**

```
/balancerx
  ├── config.json       # Dynamic configuration file
  ├── main.go           # Entry point, server, API, dashboard
  ├── models.go         # Backend, ServerPool structures
  ├── strategies.go     # Round Robin, WRR, LC, IP-Hash algorithms
  └── utils.go          # Health checks, config loader

/backend
  └── main.go           # Simple dummy backend for testing
```

---

## 🛠️ **Getting Started**

### **Prerequisites**

* Go **1.18+**

---

## 🔧 **1. Installation**

```bash
git clone https://github.com/shubhamc1947/balancerx.git
cd balancerx
```

---

## ⚙️ **2. Configuration**

Edit `balancerx/config.json`:

```json
{
  "server_port": ":8080",
  "health_check_interval": "10s",
  "lb_strategy": "least-connection",
  "backends": [
    {
      "url": "http://localhost:8081",
      "weight": 1,
      "health_path": "/"
    }
  ]
}
```

**Supported Strategies:**

| Strategy             | Value                    |
| -------------------- | ------------------------ |
| Round Robin          | `"round-robin"`          |
| Weighted Round Robin | `"weighted-round-robin"` |
| Least Connections    | `"least-connection"`     |
| IP Hash              | `"ip-hash"`              |

---

## ▶️ **3. Running the System**

### **Step 1 — Start Dummy Backends**

This starts 3 test backends on ports `8081`, `8082`, `8083`:

```bash
go run backend/main.go
```

---

### **Step 2 — Start BalancerX**

```bash
go run ./balancerx
```

---

## 🖥️ **Dashboard**

Open:

```
http://localhost:8080/ui
```

From the control panel you can:

* ✔ See backend status
* ✔ Monitor live connections
* ✔ Add a new server
* ✔ Remove a server

---

# 🔌 **API Reference**

### **List Backends**

```
GET /api/backends
```

---

### **Add Backend**

```
POST /api/backends
```

**Example:**

```bash
curl -X POST http://localhost:8080/api/backends \
  -d '{"url": "http://localhost:8084", "weight": 2, "health_path": "/"}'
```

---

### **Remove Backend**

```
DELETE /api/backends
```

---

# 🧠 **Architecture Highlights**

### **Concurrency Model**

* 🧵 **Goroutines** — non-blocking request forwarding & health checks
* 🔐 **Atomic Counters** — RR index, active connections
* 🔒 **RWMutex** — safe concurrent backend management

---

### **Resilience**

* Custom `ReverseProxy` error handler
* Automatic retries to next healthy backend
* Context-aware retry limit

---

# 🔮 **Future Improvements**

* [ ] Weighted Least Connections
* [ ] Prometheus `/metrics` exporter
* [ ] HTTPS/TLS termination
* [ ] Rate limiting

---

# 📝 **License**

This project is licensed under the **MIT License**.
