package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

// SessionManager manages session states
type SessionManager struct {
    SessionID string
    // Add more fields if necessary
}

// GetSessionID retrieves the current session ID
func (sm *SessionManager) GetSessionID() string {
    return sm.SessionID
}

// UpdateSessionID updates the session ID
func (sm *SessionManager) UpdateSessionID(newID string) {
    sm.SessionID = newID
}

// SessionManagerInstance is a global instance of SessionManager
var SessionManagerInstance = &SessionManager{}

// Add these new vars at the top of the file
var (
    defaultTimeout = 30 * time.Second
    circuitBreaker *gobreaker.CircuitBreaker
)

func init() {
    // Configure circuit breaker
    circuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        "marketplace",
        MaxRequests: 3,
        Interval:    10 * time.Second,
        Timeout:     60 * time.Second,
        OnStateChange: func(name string, from, to gobreaker.State) {
            log.Printf("Circuit breaker state changed from %v to %v", from, to)
        },
    })
}

type MorpheusSession struct {
    SessionID string
    LastUsed  time.Time
}

// Add session management
var (
    activeSession *MorpheusSession
    sessionMutex  sync.Mutex
)

// ensureSession makes sure we have an active session with Morpheus node
func ensureSession() error {
    sessionMutex.Lock()
    defer sessionMutex.Unlock()

    if activeSession != nil && time.Since(activeSession.LastUsed) < 30*time.Minute {
        return nil
    }

    reqBody := map[string]interface{}{
        "user": os.Getenv("WALLET_ADDRESS"),
    }

    reqBytes, err := json.Marshal(reqBody)
    if err != nil {
        return fmt.Errorf("failed to marshal session request: %v", err)
    }

    resp, err := http.Post("http://marketplace:9000/proxy/sessions/initiate", 
        "application/json", bytes.NewBuffer(reqBytes))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var result struct {
        SessionID string `json:"session_id"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

    activeSession = &MorpheusSession{
        SessionID: result.SessionID,
        LastUsed: time.Now(),
    }
    return nil
}

// ProxyChatCompletion handles incoming chat completion requests
func ProxyChatCompletion(w http.ResponseWriter, r *http.Request) {
    // Ensure we have active session
    if err := ensureSession(); err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to establish session")
        return
    }

    var requestBody map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Add session_id to forwarded request headers
    r.Header.Set("session_id", activeSession.SessionID)
    
    stream, ok := requestBody["stream"].(bool)
    if (!ok) {
        stream = false // Default to non-streaming if not specified
    }

    if stream {
        handleStreamingRequest(w, requestBody)
    } else {
        handleNonStreamingRequest(w, requestBody)
    }
}

// forwardRequest forwards the request to the marketplace node with necessary headers
func forwardRequest(requestBody map[string]interface{}) (*http.Response, error) {
    marketplaceURL := os.Getenv("MARKETPLACE_URL")
    if marketplaceURL == "" {
        return nil, fmt.Errorf("MARKETPLACE_URL environment variable is not set")
    }

    // Add debug logging for URL
    log.Printf("Attempting to forward request to: %s", marketplaceURL)

    // Test marketplace connection
    client := &http.Client{Timeout: 5 * time.Second}
    _, err := client.Get(strings.TrimSuffix(marketplaceURL, "/v1/chat/completions"))
    if err != nil {
        return nil, fmt.Errorf("marketplace is not accessible: %v", err)
    }

    reqBodyBytes, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request body: %v", err)
    }

    req, err := http.NewRequest("POST", marketplaceURL, bytes.NewBuffer(reqBodyBytes))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")
    if activeSession != nil {
        req.Header.Set("session_id", activeSession.SessionID)
    }

    // Add debug logging
    log.Printf("Request headers: %v", req.Header)
    log.Printf("Request body: %s", reqBodyBytes)

    client = &http.Client{
        Timeout: 30 * time.Second,
    }
    
    // Add detailed error logging
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Request failed: %v", err)
        return nil, fmt.Errorf("failed to forward request: %v", err)
    }

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        log.Printf("Marketplace returned error status %d: %s", resp.StatusCode, string(body))
        resp.Body = io.NopCloser(bytes.NewBuffer(body))
    }

    // Add response logging
    log.Printf("Response status: %d", resp.StatusCode)
    log.Printf("Response headers: %v", resp.Header)

    return resp, nil
}

// handleStreamingRequest processes streaming requests
func handleStreamingRequest(w http.ResponseWriter, requestBody map[string]interface{}) {
    resp, err := forwardRequest(requestBody)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to forward streaming request")
        return
    }
    defer resp.Body.Close()

    setStreamingHeaders(w)

    flusher, ok := w.(http.Flusher)
    if !ok {
        respondWithError(w, http.StatusInternalServerError, "Streaming unsupported")
        return
    }

    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        fmt.Fprintf(w, "%s\n", scanner.Text())
        flusher.Flush()
    }

    if err := scanner.Err(); err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error reading streaming response")
    }
}

// handleNonStreamingRequest processes non-streaming requests
func handleNonStreamingRequest(w http.ResponseWriter, requestBody map[string]interface{}) {
    resp, err := forwardRequest(requestBody)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to forward request")
        return
    }
    defer resp.Body.Close()

    copyHeaders(w, resp.Header)
    w.WriteHeader(resp.StatusCode)
    if _, err := io.Copy(w, resp.Body); err != nil {
        log.Printf("Error copying response body: %v", err)
    }
}

// setStreamingHeaders sets the necessary headers for streaming responses
func setStreamingHeaders(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
}

// copyHeaders copies headers from the marketplace response to the client response
func copyHeaders(w http.ResponseWriter, headers http.Header) {
    for key, values := range headers {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
}

// respondWithError sends an error response to the client
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// StartProxyServer starts the proxy server
func StartProxyServer() {
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    })
    http.HandleFunc("/v1/chat/completions", ProxyChatCompletion)
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Printf("Proxy server is running on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}