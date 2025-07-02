// FlexCore Multi-Tenant Authentication - REAL JWT Implementation
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Domain      string                 `json:"domain"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Active      bool                   `json:"active"`
	Subscription string                `json:"subscription"` // "free", "pro", "enterprise"
	Limits      TenantLimits           `json:"limits"`
}

type TenantLimits struct {
	MaxUsers       int `json:"max_users"`
	MaxAPIRequests int `json:"max_api_requests"`
	MaxStorage     int `json:"max_storage_gb"`
}

// User represents a user within a tenant
type User struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	PasswordHash string                `json:"password_hash"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Active      bool                   `json:"active"`
	LastLogin   *time.Time             `json:"last_login,omitempty"`
}

// Role defines a role with permissions
type Role struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// JWT Claims for multi-tenant
type FlexCoreClaims struct {
	UserID      string                 `json:"user_id"`
	TenantID    string                 `json:"tenant_id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	TenantLimits TenantLimits          `json:"tenant_limits"`
	Metadata    map[string]interface{} `json:"metadata"`
	jwt.RegisteredClaims
}

// Auth Service for multi-tenant authentication
type MultiTenantAuthService struct {
	tenants     map[string]*Tenant
	users       map[string]*User
	roles       map[string]*Role
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	mu          sync.RWMutex
	server      *http.Server
	sessions    map[string]*Session
	blacklist   map[string]time.Time
}

type Session struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	TenantID  string            `json:"tenant_id"`
	Token     string            `json:"token"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	Metadata  map[string]string `json:"metadata"`
}

func NewMultiTenantAuthService() (*MultiTenantAuthService, error) {
	// Generate RSA key pair for JWT signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	publicKey := &privateKey.PublicKey

	auth := &MultiTenantAuthService{
		tenants:    make(map[string]*Tenant),
		users:      make(map[string]*User),
		roles:      make(map[string]*Role),
		privateKey: privateKey,
		publicKey:  publicKey,
		sessions:   make(map[string]*Session),
		blacklist:  make(map[string]time.Time),
	}

	// Initialize demo data
	auth.initializeDemoData()

	return auth, nil
}

func (auth *MultiTenantAuthService) initializeDemoData() {
	// Create demo tenants
	tenants := []*Tenant{
		{
			ID:     "tenant-1",
			Name:   "FlexCore Enterprise",
			Domain: "enterprise.flexcore.com",
			Settings: map[string]interface{}{
				"theme": "dark",
				"timezone": "UTC",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
			Subscription: "enterprise",
			Limits: TenantLimits{
				MaxUsers:       1000,
				MaxAPIRequests: 100000,
				MaxStorage:     1000,
			},
		},
		{
			ID:     "tenant-2",
			Name:   "FlexCore Startup",
			Domain: "startup.flexcore.com",
			Settings: map[string]interface{}{
				"theme": "light",
				"timezone": "America/New_York",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
			Subscription: "pro",
			Limits: TenantLimits{
				MaxUsers:       50,
				MaxAPIRequests: 10000,
				MaxStorage:     100,
			},
		},
	}

	for _, tenant := range tenants {
		auth.tenants[tenant.ID] = tenant
	}

	// Create demo roles
	roles := []*Role{
		{
			ID:       "REDACTED_LDAP_BIND_PASSWORD-role",
			TenantID: "tenant-1",
			Name:     "Administrator",
			Description: "Full system access",
			Permissions: []string{"*"},
			CreatedAt: time.Now(),
		},
		{
			ID:       "user-role",
			TenantID: "tenant-1",
			Name:     "User",
			Description: "Standard user access",
			Permissions: []string{"read:pipelines", "create:pipelines", "read:workflows"},
			CreatedAt: time.Now(),
		},
	}

	for _, role := range roles {
		auth.roles[role.ID] = role
	}

	// Create demo users
	REDACTED_LDAP_BIND_PASSWORDHash, _ := bcrypt.GenerateFromPassword([]byte("flexcore100"), bcrypt.DefaultCost)
	userHash, _ := bcrypt.GenerateFromPassword([]byte("user123"), bcrypt.DefaultCost)

	users := []*User{
		{
			ID:          "user-1",
			TenantID:    "tenant-1",
			Username:    "REDACTED_LDAP_BIND_PASSWORD",
			Email:       "REDACTED_LDAP_BIND_PASSWORD@enterprise.flexcore.com",
			PasswordHash: string(REDACTED_LDAP_BIND_PASSWORDHash),
			Roles:       []string{"REDACTED_LDAP_BIND_PASSWORD-role"},
			Permissions: []string{"*"},
			Metadata: map[string]interface{}{
				"department": "IT",
				"location": "HQ",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		},
		{
			ID:          "user-2",
			TenantID:    "tenant-2",
			Username:    "user",
			Email:       "user@startup.flexcore.com",
			PasswordHash: string(userHash),
			Roles:       []string{"user-role"},
			Permissions: []string{"read:pipelines", "create:pipelines"},
			Metadata: map[string]interface{}{
				"department": "Engineering",
				"location": "Remote",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		},
	}

	for _, user := range users {
		auth.users[user.ID] = user
	}
}

func (auth *MultiTenantAuthService) generateJWT(user *User, tenant *Tenant) (string, error) {
	now := time.Now()
	expirationTime := now.Add(24 * time.Hour)

	claims := &FlexCoreClaims{
		UserID:      user.ID,
		TenantID:    user.TenantID,
		Username:    user.Username,
		Email:       user.Email,
		Roles:       user.Roles,
		Permissions: user.Permissions,
		TenantLimits: tenant.Limits,
		Metadata:    user.Metadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "flexcore-auth",
			Subject:   user.ID,
			Audience:  []string{tenant.ID},
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(auth.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	// Store session
	session := &Session{
		ID:        claims.ID,
		UserID:    user.ID,
		TenantID:  user.TenantID,
		Token:     tokenString,
		CreatedAt: now,
		ExpiresAt: expirationTime,
		Metadata: map[string]string{
			"user_agent": "flexcore-auth",
			"ip_address": "127.0.0.1",
		},
	}

	auth.mu.Lock()
	auth.sessions[session.ID] = session
	auth.mu.Unlock()

	return tokenString, nil
}

func (auth *MultiTenantAuthService) validateJWT(tokenString string) (*FlexCoreClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &FlexCoreClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return auth.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*FlexCoreClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	// Check blacklist
	auth.mu.RLock()
	if _, blacklisted := auth.blacklist[claims.ID]; blacklisted {
		auth.mu.RUnlock()
		return nil, fmt.Errorf("token is blacklisted")
	}
	auth.mu.RUnlock()

	return claims, nil
}

func (auth *MultiTenantAuthService) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
		TenantID string `json:"tenant_id"`
		Domain   string `json:"domain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Find tenant
	auth.mu.RLock()
	var tenant *Tenant
	if loginReq.TenantID != "" {
		tenant = auth.tenants[loginReq.TenantID]
	} else if loginReq.Domain != "" {
		for _, t := range auth.tenants {
			if t.Domain == loginReq.Domain {
				tenant = t
				break
			}
		}
	}
	auth.mu.RUnlock()

	if tenant == nil || !tenant.Active {
		http.Error(w, "Tenant not found or inactive", http.StatusUnauthorized)
		return
	}

	// Find user
	auth.mu.RLock()
	var user *User
	for _, u := range auth.users {
		if u.TenantID == tenant.ID && (u.Username == loginReq.Username || u.Email == loginReq.Username) {
			user = u
			break
		}
	}
	auth.mu.RUnlock()

	if user == nil || !user.Active {
		http.Error(w, "User not found or inactive", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginReq.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	token, err := auth.generateJWT(user, tenant)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now

	// Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"user":    user,
		"tenant":  tenant,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
}

func (auth *MultiTenantAuthService) validateHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.validateJWT(tokenString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   true,
		"claims":  claims,
		"user_id": claims.UserID,
		"tenant_id": claims.TenantID,
		"permissions": claims.Permissions,
	})
}

func (auth *MultiTenantAuthService) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.validateJWT(tokenString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Add to blacklist
	auth.mu.Lock()
	auth.blacklist[claims.ID] = time.Now()
	delete(auth.sessions, claims.ID)
	auth.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

func (auth *MultiTenantAuthService) tenantsHandler(w http.ResponseWriter, r *http.Request) {
	auth.mu.RLock()
	tenants := make([]*Tenant, 0, len(auth.tenants))
	for _, tenant := range auth.tenants {
		if tenant.Active {
			tenants = append(tenants, tenant)
		}
	}
	auth.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenants": tenants,
		"count":   len(tenants),
	})
}

func (auth *MultiTenantAuthService) healthHandler(w http.ResponseWriter, r *http.Request) {
	auth.mu.RLock()
	stats := map[string]interface{}{
		"status":        "healthy",
		"service":       "flexcore-multi-tenant-auth",
		"timestamp":     time.Now(),
		"tenants":       len(auth.tenants),
		"users":         len(auth.users),
		"active_sessions": len(auth.sessions),
		"blacklisted_tokens": len(auth.blacklist),
	}
	auth.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (auth *MultiTenantAuthService) publicKeyHandler(w http.ResponseWriter, r *http.Request) {
	// Export public key for JWT validation by other services
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(auth.publicKey)
	if err != nil {
		http.Error(w, "Failed to marshal public key", http.StatusInternalServerError)
		return
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"public_key": string(pubKeyPEM),
		"algorithm": "RS256",
		"key_id":    generateKeyID(auth.publicKey),
	})
}

func generateKeyID(publicKey *rsa.PublicKey) string {
	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(publicKey)
	hash := sha256.Sum256(pubKeyBytes)
	return base64.URLEncoding.EncodeToString(hash[:8])
}

func (auth *MultiTenantAuthService) setupRoutes() {
	mux := http.NewServeMux()

	// Auth endpoints
	mux.HandleFunc("/auth/login", auth.loginHandler)
	mux.HandleFunc("/auth/validate", auth.validateHandler)
	mux.HandleFunc("/auth/logout", auth.logoutHandler)
	mux.HandleFunc("/auth/public-key", auth.publicKeyHandler)

	// Tenant management
	mux.HandleFunc("/tenants", auth.tenantsHandler)

	// Health check
	mux.HandleFunc("/health", auth.healthHandler)

	auth.server = &http.Server{
		Addr:    ":8998",
		Handler: mux,
	}
}

func (auth *MultiTenantAuthService) Start() error {
	auth.setupRoutes()

	log.Println("🔐 FlexCore Multi-Tenant Auth Server starting on :8998")
	log.Println("🏢 Login endpoint: /auth/login")
	log.Println("✅ Validate endpoint: /auth/validate")
	log.Println("🚪 Logout endpoint: /auth/logout")
	log.Println("🔑 Public key endpoint: /auth/public-key")
	log.Println("🏢 Tenants endpoint: /tenants")
	log.Println("🩺 Health check: /health")

	return auth.server.ListenAndServe()
}

func (auth *MultiTenantAuthService) Stop(ctx context.Context) error {
	log.Println("🛑 Shutting down auth server...")
	return auth.server.Shutdown(ctx)
}

func main() {
	fmt.Println("🔐 FlexCore Multi-Tenant Authentication System")
	fmt.Println("=============================================")
	fmt.Println("🏢 Real multi-tenant JWT authentication")
	fmt.Println("🔑 RSA-256 signed tokens")
	fmt.Println("📊 Role-based access control (RBAC)")
	fmt.Println("⏱️ Session management and blacklisting")
	fmt.Println()

	auth, err := NewMultiTenantAuthService()
	if err != nil {
		log.Fatalf("Failed to create auth service: %v", err)
	}

	if err := auth.Start(); err != nil {
		log.Fatalf("Auth server failed: %v", err)
	}
}
