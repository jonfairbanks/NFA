package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

type SessionManager struct {
    client          *http.Client
    marketplaceURL  string
    modelID         string
    sessionID       string
    sessionExpiry   time.Time
    sessionDuration time.Duration
    mu              sync.Mutex
}

func NewSessionManager(marketplaceURL, modelID string, sessionDuration time.Duration) *SessionManager {
    return &SessionManager{
        client:          &http.Client{Timeout: 30 * time.Second},
        marketplaceURL:  marketplaceURL,
        modelID:         modelID,
        sessionDuration: sessionDuration,
    }
}

func (sm *SessionManager) GetSessionID(ctx context.Context) (string, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    if sm.sessionID == "" || time.Now().After(sm.sessionExpiry) {
        err := sm.openSession(ctx)
        if err != nil {
            return "", err
        }
    }
    return sm.sessionID, nil
}

func (sm *SessionManager) openSession(ctx context.Context) error {
    url := sm.marketplaceURL + "/blockchain/models/" + sm.modelID + "/session"
    reqBody := map[string]interface{}{
        "sessionDuration": int(sm.sessionDuration.Seconds()),
    }
    jsonBody, _ := json.Marshal(reqBody)
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := sm.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to open session: %s", string(bodyBytes))
    }

    var result struct {
        SessionID string `json:"sessionID"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

    sm.sessionID = result.SessionID
    sm.sessionExpiry = time.Now().Add(sm.sessionDuration)
    log.Printf("Opened new session: %s", sm.sessionID)
    return nil
}

func (sm *SessionManager) CloseSession(ctx context.Context) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    if sm.sessionID == "" {
        return nil
    }

    url := sm.marketplaceURL + "/blockchain/sessions/" + sm.sessionID + "/close"
    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return err
    }

    resp, err := sm.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to close session: %s", string(bodyBytes))
    }

    sm.sessionID = ""
    sm.sessionExpiry = time.Time{}
    log.Printf("Closed session")
    return nil
}

func (sm *SessionManager) ProxyChatCompletion(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Read the request body
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
        return
    }
    r.Body.Close()

    // Check if the request is for streaming
    var requestBody map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
        http.Error(w, "Invalid JSON in request body: "+err.Error(), http.StatusBadRequest)
        return
    }
    isStreaming := false
    if streamVal, ok := requestBody["stream"]; ok {
        isStreaming, _ = streamVal.(bool)
    }

    // Obtain a valid session ID
    sessionID, err := sm.GetSessionID(ctx)
    if err != nil {
        http.Error(w, "Failed to obtain session ID: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Create a new request to the marketplace node
    url := sm.marketplaceURL + r.URL.Path
    proxyReq, err := http.NewRequestWithContext(ctx, r.Method, url, bytes.NewReader(bodyBytes))
    if err != nil {
        http.Error(w, "Failed to create proxy request: "+err.Error(), http.StatusInternalServerError)
        return
    }
    proxyReq.Header = r.Header.Clone()
    proxyReq.Header.Set("session_id", sessionID)

    // Send the request to the marketplace node
    resp, err := sm.client.Do(proxyReq)
    if err != nil {
        http.Error(w, "Failed to contact marketplace node: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    // Copy response headers and status code
    for k, v := range resp.Header {
        w.Header()[k] = v
    }
    w.WriteHeader(resp.StatusCode)

    if isStreaming {
        // Handle streaming responses
        flusher, ok := w.(http.Flusher)
        if !ok {
            http.Error(w, "Streaming not supported", http.StatusInternalServerError)
            return
        }

        buf := make([]byte, 4096)
        for {
            n, err := resp.Body.Read(buf)
            if n > 0 {
                if _, writeErr := w.Write(buf[:n]); writeErr != nil {
                    log.Printf("Failed to write to client: %v", writeErr)
                    return
                }
                flusher.Flush()
            }
            if err != nil {
                if err != io.EOF {
                    log.Printf("Error reading response body: %v", err)
                }
                break
            }
        }
    } else {
        // Handle non-streaming responses
        if _, err := io.Copy(w, resp.Body); err != nil {
            log.Printf("Failed to copy response body: %v", err)
        }
    }
}

func main() {
    marketplaceURL := os.Getenv("MARKETPLACE_URL")
    if marketplaceURL == "" {
        log.Fatal("MARKETPLACE_URL environment variable is required")
    }

    modelID := os.Getenv("MODEL_ID")
    if modelID == "" {
        log.Fatal("MODEL_ID environment variable is required")
    }

    sessionDurationStr := os.Getenv("SESSION_DURATION")
    if sessionDurationStr == "" {
        sessionDurationStr = "1h" // Default to 1 hour
    }
    sessionDuration, err := time.ParseDuration(sessionDurationStr)
    if err != nil {
        log.Fatalf("Invalid SESSION_DURATION: %v", err)
    }

    sm := NewSessionManager(marketplaceURL, modelID, sessionDuration)

    // Handle Ctrl+C and other interrupt signals to close the session gracefully
    go func() {
        c := make(chan os.Signal, 1)
        signal.Notify(c, os.Interrupt)
        <-c
        log.Println("Shutting down server...")
        if err := sm.CloseSession(context.Background()); err != nil {
            log.Printf("Error closing session: %v", err)
        }
        os.Exit(0)
    }()

    // Handle the OpenAI Chat Completions API endpoint
    http.HandleFunc("/v1/chat/completions", sm.ProxyChatCompletion)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Printf("Server starting on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}