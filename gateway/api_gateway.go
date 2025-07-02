// FlexCore API Gateway - REAL Load Balancing & Service Discovery
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Gateway Configuration
type GatewayConfig struct {
	Port            string   `json:"port"`
	Backends        []Backend `json:"backends"`
	HealthCheckURL  string   `json:"health_check_url"`
	JWTSecret       string   `json:"jwt_secret"`
	RateLimitRPS    int      `json:"rate_limit_rps"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
}

type Backend struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Weight   int    `json:"weight"`
	Healthy  bool   `json:"healthy"`
	LastSeen time.Time `json:"last_seen"`
}

// Load Balancer with Round Robin + Weighted
type LoadBalancer struct {
	backends []Backend
	current  int
	mu       sync.RWMutex
}

func NewLoadBalancer(backends []Backend) *LoadBalancer {
	return &LoadBalancer{
		backends: backends,
		current:  0,
	}
}

func (lb *LoadBalancer) NextBackend() *Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Find next healthy backend with weighted round-robin
	attempts := 0
	for attempts < len(lb.backends)*2 {
		backend := &lb.backends[lb.current]
		lb.current = (lb.current + 1) % len(lb.backends)

		if backend.Healthy {
			// Apply weight (simple implementation)
			if attempts%backend.Weight == 0 {
				return backend
			}
		}
		attempts++
	}

	// Fallback to any healthy backend
	for i := range lb.backends {
		if lb.backends[i].Healthy {
			return &lb.backends[i]
		}
	}

	return nil
}

func (lb *LoadBalancer) UpdateBackendHealth(id string, healthy bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i := range lb.backends {
		if lb.backends[i].ID == id {
			lb.backends[i].Healthy = healthy
			lb.backends[i].LastSeen = time.Now()
			break
		}
	}
}

// Metrics
var (
	gatewayRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of gateway requests",
		},
		[]string{"backend", "method", "status"},
	)

	gatewayRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "gateway_request_duration_seconds",
			Help: "Gateway request duration in seconds",
		},
		[]string{"backend", "method"},
	)

	gatewayBackendsHealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_backends_healthy",
			Help: "Number of healthy backends",
		},
		[]string{"backend"},
	)
)

// API Gateway
type APIGateway struct {
	config       *GatewayConfig
	loadBalancer *LoadBalancer
	server       *http.Server
	jwtSecret    []byte
}

func NewAPIGateway(configPath string) (*APIGateway, error) {
	// Load configuration
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config GatewayConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Initialize load balancer
	lb := NewLoadBalancer(config.Backends)

	// Register metrics
	prometheus.MustRegister(gatewayRequestsTotal, gatewayRequestDuration, gatewayBackendsHealthy)

	return &APIGateway{
		config:       &config,
		loadBalancer: lb,
		jwtSecret:    []byte(config.JWTSecret),
	}, nil
}

func (gw *APIGateway) validateJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return gw.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

func (gw *APIGateway) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks and metrics
		if strings.HasPrefix(r.URL.Path, "/health") || strings.HasPrefix(r.URL.Path, "/metrics") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract JWT token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Allow some endpoints without auth for demo
			if strings.HasPrefix(r.URL.Path, "/api/public") {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := gw.validateJWT(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user info to context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			ctx := context.WithValue(r.Context(), "user", claims)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

func (gw *APIGateway) proxyHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Get next backend
	backend := gw.loadBalancer.NextBackend()
	if backend == nil {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		gatewayRequestsTotal.WithLabelValues("none", r.Method, "503").Inc()
		return
	}

	// Parse backend URL
	targetURL, err := url.Parse(backend.URL)
	if err != nil {
		http.Error(w, "Invalid backend URL", http.StatusInternalServerError)
		gatewayRequestsTotal.WithLabelValues(backend.ID, r.Method, "500").Inc()
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Custom transport with timeout
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialTimeout:     time.Duration(gw.config.TimeoutSeconds) * time.Second,
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Backend %s error: %v", backend.ID, err)
		gw.loadBalancer.UpdateBackendHealth(backend.ID, false)
		http.Error(w, "Backend unavailable", http.StatusBadGateway)
		gatewayRequestsTotal.WithLabelValues(backend.ID, r.Method, "502").Inc()
	}

	// Modify request
	r.URL.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme
	r.Header.Set("X-Forwarded-For", r.RemoteAddr)
	r.Header.Set("X-Gateway-Backend", backend.ID)

	// Serve the request
	proxy.ServeHTTP(w, r)

	// Record metrics
	duration := time.Since(startTime)
	gatewayRequestDuration.WithLabelValues(backend.ID, r.Method).Observe(duration.Seconds())
	gatewayRequestsTotal.WithLabelValues(backend.ID, r.Method, "200").Inc()
}

func (gw *APIGateway) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now(),
		"service": "flexcore-api-gateway",
		"version": "1.0.0",
		"backends": gw.loadBalancer.backends,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (gw *APIGateway) generateJWT(userID, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(gw.jwtSecret)
}

func (gw *APIGateway) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Simple auth check (in production, use real auth service)
	if loginReq.Username == "REDACTED_LDAP_BIND_PASSWORD" && loginReq.Password == "flexcore100" {
		token, err := gw.generateJWT(loginReq.Username, "REDACTED_LDAP_BIND_PASSWORD")
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
			"user":  loginReq.Username,
			"role":  "REDACTED_LDAP_BIND_PASSWORD",
		})
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

func (gw *APIGateway) healthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, backend := range gw.loadBalancer.backends {
			go func(b Backend) {
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get(b.URL + "/health")

				healthy := err == nil && resp != nil && resp.StatusCode == 200
				gw.loadBalancer.UpdateBackendHealth(b.ID, healthy)

				if healthy {
					gatewayBackendsHealthy.WithLabelValues(b.ID).Set(1)
				} else {
					gatewayBackendsHealthy.WithLabelValues(b.ID).Set(0)
				}

				if resp != nil {
					resp.Body.Close()
				}
			}(backend)
		}
	}
}

func (gw *APIGateway) setupRoutes() {
	mux := http.NewServeMux()

	// Auth endpoints
	mux.HandleFunc("/auth/login", gw.loginHandler)

	// Health and metrics
	mux.HandleFunc("/health", gw.healthCheckHandler)
	mux.Handle("/metrics", promhttp.Handler())

	// Protected proxy endpoints
	mux.Handle("/", gw.authMiddleware(http.HandlerFunc(gw.proxyHandler)))

	gw.server = &http.Server{
		Addr:    ":" + gw.config.Port,
		Handler: mux,
	}
}

func (gw *APIGateway) Start() error {
	gw.setupRoutes()

	// Start health checker
	go gw.healthChecker()

	log.Printf("🌐 FlexCore API Gateway starting on port %s", gw.config.Port)
	log.Println("🔐 Auth endpoint: /auth/login")
	log.Println("🩺 Health check: /health")
	log.Println("📊 Metrics: /metrics")
	log.Printf("🔄 Load balancing %d backends", len(gw.config.Backends))

	return gw.server.ListenAndServe()
}

func (gw *APIGateway) Stop(ctx context.Context) error {
	log.Println("🛑 Shutting down API Gateway...")
	return gw.server.Shutdown(ctx)
}

func main() {
	fmt.Println("🎆 FlexCore API Gateway - REAL Load Balancing")
	fmt.Println("=============================================")
	fmt.Println("🔄 Round-robin + Weighted load balancing")
	fmt.Println("🔐 JWT Authentication")
	fmt.Println("🩺 Health checking")
	fmt.Println("📊 Prometheus metrics")
	fmt.Println()

	gateway, err := NewAPIGateway("gateway_config.json")
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := gateway.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway failed: %v", err)
		}
	}()

	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := gateway.Stop(ctx); err != nil {
		log.Printf("Gateway shutdown error: %v", err)
	}

	fmt.Println("🏁 API Gateway shutdown complete")
}
