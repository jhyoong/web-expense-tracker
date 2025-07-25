package middleware

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "net/http"
    "strings"
    "sync"
    "time"
)

type CSRFTokenStore struct {
    tokens map[string]time.Time
    mutex  sync.RWMutex
}

func NewCSRFTokenStore() *CSRFTokenStore {
    store := &CSRFTokenStore{
        tokens: make(map[string]time.Time),
    }
    
    // Start cleanup goroutine
    go store.cleanup()
    
    return store
}

func (store *CSRFTokenStore) cleanup() {
    ticker := time.NewTicker(10 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        store.mutex.Lock()
        now := time.Now()
        for token, expiry := range store.tokens {
            if now.After(expiry) {
                delete(store.tokens, token)
            }
        }
        store.mutex.Unlock()
    }
}

func (store *CSRFTokenStore) GenerateToken() string {
    bytes := make([]byte, 32)
    rand.Read(bytes)
    token := base64.URLEncoding.EncodeToString(bytes)
    
    store.mutex.Lock()
    store.tokens[token] = time.Now().Add(30 * time.Minute) // 30 minute expiry
    store.mutex.Unlock()
    
    return token
}

func (store *CSRFTokenStore) ValidateToken(token string) bool {
    if token == "" {
        return false
    }
    
    store.mutex.RLock()
    expiry, exists := store.tokens[token]
    store.mutex.RUnlock()
    
    if !exists {
        return false
    }
    
    if time.Now().After(expiry) {
        // Clean up expired token
        store.mutex.Lock()
        delete(store.tokens, token)
        store.mutex.Unlock()
        return false
    }
    
    return true
}

func (store *CSRFTokenStore) ConsumeToken(token string) bool {
    if token == "" {
        return false
    }
    
    store.mutex.Lock()
    defer store.mutex.Unlock()
    
    expiry, exists := store.tokens[token]
    if !exists {
        return false
    }
    
    if time.Now().After(expiry) {
        delete(store.tokens, token)
        return false
    }
    
    // Remove token after use (one-time use)
    delete(store.tokens, token)
    return true
}

func CSRFMiddleware(store *CSRFTokenStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip CSRF for GET, HEAD, OPTIONS, and TRACE methods
            if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" || r.Method == "TRACE" {
                next.ServeHTTP(w, r)
                return
            }
            
            // Skip CSRF for some endpoints that don't need it (like file uploads)
            if strings.HasPrefix(r.URL.Path, "/static/") {
                next.ServeHTTP(w, r)
                return
            }
            
            // Get token from header or form
            token := r.Header.Get("X-CSRF-Token")
            if token == "" {
                token = r.FormValue("csrf_token")
            }
            
            if !store.ConsumeToken(token) {
                http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

func CSRFTokenHandler(store *CSRFTokenStore) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := store.GenerateToken()
        
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"csrf_token": "%s"}`, token)
    }
}