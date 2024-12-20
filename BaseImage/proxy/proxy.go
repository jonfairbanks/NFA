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

	"sort"

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
    modelCache     = make(map[string]string) // map[modelHandle]modelId
    modelCacheMux  sync.RWMutex
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

// Update MorpheusSession struct to include ModelID
type MorpheusSession struct {
    SessionID string
    ModelID   string
    LastUsed  time.Time
}

// Update activeSessions to manage sessions per model ID
var (
    activeSessions = make(map[string]*MorpheusSession)
    sessionMutex   sync.Mutex
)

// Remove getModelID function as modelID comes from the request

// Modify ensureSession to accept modelID as a parameter
func ensureSession(modelID string) error {
    sessionMutex.Lock()
    defer sessionMutex.Unlock()

    // Debug the session state
    log.Printf("Checking session state for model %s", modelID)

    session, exists := activeSessions[modelID]
    if exists && time.Since(session.LastUsed) < 30*time.Minute {
        log.Printf("Using existing session for model %s: %s", modelID, session.SessionID)
        session.LastUsed = time.Now()
        return nil
    }

    log.Printf("Establishing new session for model %s", modelID)

    reqBody := map[string]interface{}{
        "sessionDuration": 3600, // Send as number, not string
        "failover":        false,
    }

    reqBytes, err := json.Marshal(reqBody)
    if err != nil {
        return fmt.Errorf("failed to marshal session request: %v", err)
    }

    // Updated session endpoint with model ID
    sessionURL := fmt.Sprintf("http://marketplace:9000/blockchain/models/%s/session", modelID)
    resp, err := http.Post(sessionURL, "application/json", bytes.NewBuffer(reqBytes))
    if err != nil {
        log.Printf("Session establishment failed: %v", err)
        return fmt.Errorf("failed to establish session: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        log.Printf("Session response body: %s", string(bodyBytes))
        return fmt.Errorf("failed to establish session: %s", string(bodyBytes))
    }

    var result struct {
        Id string `json:"sessionID"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to decode session response: %v", err)
    }

    log.Printf("Session response: %+v", result)

    if result.Id == "" {
        return fmt.Errorf("failed to get valid session ID from response")
    }

    // Update the session for the model ID
    activeSessions[modelID] = &MorpheusSession{
        SessionID: result.Id,
        ModelID:   modelID,
        LastUsed:  time.Now(),
    }

    log.Printf("Successfully established new session for model %s: %s", modelID, result.Id)
    return nil
}

// Add model handle mapping
var modelHandleToID = map[string]string{
    "LMR-Collective Cognition Mistral 7B": os.Getenv("MODEL_ID"), // default model
    "gpt-3.5-turbo": os.Getenv("MODEL_ID"), // maps to same model
    "LMR-Hermes-2-Theta-Llama-3-8B": os.Getenv("MODEL_ID"), // Add new model handle
}

// ModelInfo represents the model information from the marketplace
type ModelInfo struct {
    Id   string `json:"Id"`
    Name string `json:"Name"`
}

// ModelSearchResponse represents the marketplace API response
type ModelSearchResponse struct {
    Models []ModelInfo `json:"models"`
}

// calculateSimilarity returns a similarity score between two strings
func calculateSimilarity(s1, s2 string) float64 {
    // Convert to lowercase for case-insensitive comparison
    s1 = strings.ToLower(s1)
    s2 = strings.ToLower(s2)

    // Simple Levenshtein-based similarity
    maxLen := len(s1)
    if len(s2) > maxLen {
        maxLen = len(s2)
    }
    if maxLen == 0 {
        return 1.0
    }

    distance := levenshteinDistance(s1, s2)
    return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
    if len(s1) == 0 {
        return len(s2)
    }
    if len(s2) == 0 {
        return len(s1)
    }

    matrix := make([][]int, len(s1)+1)
    for i := range matrix {
        matrix[i] = make([]int, len(s2)+1)
        matrix[i][0] = i
    }
    for j := range matrix[0] {
        matrix[0][j] = j
    }

    for i := 1; i <= len(s1); i++ {
        for j := 1; i <= len(s2); j++ {
            cost := 1
            if s1[i-1] == s2[j-1] {
                cost = 0
            }
            matrix[i][j] = min(
                matrix[i-1][j]+1,
                matrix[i][j-1]+1,
                matrix[i-1][j-1]+cost,
            )
        }
    }

    return matrix[len(s1)][len(s2)]
}

func min(nums ...int) int {
    result := nums[0]
    for _, num := range nums[1:] {
        if num < result {
            result = num
        }
    }
    return result
}

// findModelID searches for a model ID by exact or partial match
func findModelID(modelHandle string) (string, error) {
    // Check cache first
    modelCacheMux.RLock()
    if modelID, exists := modelCache[modelHandle]; exists {
        modelCacheMux.RUnlock()
        return modelID, nil
    }
    modelCacheMux.RUnlock()

    // Query the marketplace API
    resp, err := http.Get("http://marketplace:9000/blockchain/models?limit=100&order=desc")
    if err != nil {
        return "", fmt.Errorf("failed to fetch models: %v", err)
    }
    defer resp.Body.Close()

    var searchResp ModelSearchResponse
    if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
        return "", fmt.Errorf("failed to decode models response: %v", err)
    }

    // Normalize search handle
    searchHandle := strings.ToLower(modelHandle)
    log.Printf("Searching for model matching: '%s'", modelHandle)

    // Try finding a model that contains the search term
    var matches []struct {
        id    string
        name  string
        score float64
    }

    for _, model := range searchResp.Models {
        modelName := strings.ToLower(model.Name)
        
        // Check if model name contains search term or vice versa
        if strings.Contains(modelName, searchHandle) || strings.Contains(searchHandle, modelName) {
            score := calculateSimilarity(searchHandle, modelName)
            matches = append(matches, struct {
                id    string
                name  string
                score float64
            }{
                id:    model.Id,
                name:  model.Name,
                score: score,
            })
            log.Printf("Found matching model: '%s' with score %.2f", model.Name, score)
        }
    }

    // If we found matches, use the best one
    if len(matches) > 0 {
        // Sort matches by score (highest first)
        sort.Slice(matches, func(i, j int) bool {
            return matches[i].score > matches[j].score
        })

        bestMatch := matches[0]
        log.Printf("Selected best match: '%s' with score %.2f", bestMatch.name, bestMatch.score)

        // Cache the result
        modelCacheMux.Lock()
        modelCache[modelHandle] = bestMatch.id
        modelCacheMux.Unlock()

        return bestMatch.id, nil
    }

    // If no partial matches found, try fuzzy search as a last resort
    var bestMatch string
    var bestScore float64
    const similarityThreshold = 0.3

    for _, model := range searchResp.Models {
        score := calculateSimilarity(modelHandle, model.Name)
        if score > similarityThreshold && score > bestScore {
            bestScore = score
            bestMatch = model.Id
            log.Printf("Found fuzzy match: '%s' with score %.2f", model.Name, score)
        }
    }

    if bestMatch == "" {
        return "", fmt.Errorf("no matching model found for: %s", modelHandle)
    }

    // Cache the fuzzy match result
    modelCacheMux.Lock()
    modelCache[modelHandle] = bestMatch
    modelCacheMux.Unlock()

    return bestMatch, nil
}

// validateModelHandle checks if the model handle is valid and returns the corresponding ID
func validateModelHandle(handle string) (string, error) {
    // First check if it's in our static mapping
    if id, exists := modelHandleToID[handle]; exists {
        return id, nil
    }

    // If not in static mapping, try to find it dynamically
    return findModelID(handle)
}

// Update ProxyChatCompletion function
func ProxyChatCompletion(w http.ResponseWriter, r *http.Request) {
    // Read and log the request body
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to read request body")
        return
    }
    // Restore the request body for further processing
    r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

    fmt.Printf("Received chat request body: %s\n", string(bodyBytes))

    var requestBody map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Extract and validate model handle
    modelHandle, ok := requestBody["model"].(string)
    if !ok {
        respondWithError(w, http.StatusBadRequest, "model field is required")
        return
    }

    // Convert model handle to ID
    modelID, err := validateModelHandle(modelHandle)
    if err != nil {
        respondWithError(w, http.StatusBadRequest, err.Error())
        return
    }

    // Ensure we have an active session for this model ID
    if err := ensureSession(modelID); err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to establish session")
        return
    }

    // Update request with actual model ID
    requestBody["model"] = modelID

    // Add session_id to forwarded request headers
    r.Header.Set("session_id", activeSessions[modelID].SessionID)

    stream, ok := requestBody["stream"].(bool)
    if !ok {
        stream = false // Default to non-streaming if not specified
    }

    if stream {
        handleStreamingRequest(w, requestBody, modelID)
    } else {
        handleNonStreamingRequest(w, requestBody, modelID)
    }
}

// Modify forwardRequest to accept modelID and use the correct session
func forwardRequest(requestBody map[string]interface{}, modelID string) (*http.Response, error) {
    marketplaceURL := os.Getenv("MARKETPLACE_URL")
    if marketplaceURL == "" {
        return nil, fmt.Errorf("MARKETPLACE_URL environment variable is not set")
    }

    // Add debug logging for URL
    log.Printf("Attempting to forward request to: %s", marketplaceURL)

    reqBodyBytes, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request body: %v", err)
    }

    req, err := http.NewRequest("POST", marketplaceURL, bytes.NewBuffer(reqBodyBytes))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")

    log.Printf("Active session for model %s: %+v", modelID, activeSessions[modelID])
    if session, exists := activeSessions[modelID]; exists && session.SessionID != "" {
        // Add session ID to request headers
        req.Header.Set("Session_id", session.SessionID)
        log.Printf("Setting session ID in request headers: %s", session.SessionID)
    } else {
        log.Printf("Warning: No active session ID available for model %s", modelID)
        return nil, fmt.Errorf("no active session for model %s", modelID)
    }

    // Add debug logging for all headers
    log.Printf("Request headers: %v", req.Header)
    log.Printf("Request body: %s", reqBodyBytes)

    client := &http.Client{
        Timeout: 30 * time.Second,
    }

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

// Update handleStreamingRequest and handleNonStreamingRequest
func handleStreamingRequest(w http.ResponseWriter, requestBody map[string]interface{}, modelID string) {
    resp, err := forwardRequest(requestBody, modelID)
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

func handleNonStreamingRequest(w http.ResponseWriter, requestBody map[string]interface{}, modelID string) {
    resp, err := forwardRequest(requestBody, modelID)
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
    // Validate required environment variables
    walletAddress := os.Getenv("WALLET_ADDRESS")
    if walletAddress == "" || walletAddress == "0x0000000000000000000000000000000000000000" {
        log.Fatal("WALLET_ADDRESS environment variable must be set to a valid address")
    }
    
    modelID := os.Getenv("MODEL_ID") // Validate MODEL_ID exists
    log.Printf("Starting proxy server with Model ID: %s", modelID)

    // Validate that at least one model ID is configured
    if os.Getenv("MODEL_ID") == "" {
        log.Fatal("MODEL_ID environment variable must be set")
    }
    
    log.Printf("Starting proxy server with supported models: %v", getModelHandles())

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

// Helper function to get supported model handles
func getModelHandles() []string {
    handles := make([]string, 0, len(modelHandleToID))
    for handle := range modelHandleToID {
        handles = append(handles, handle)
    }
    return handles}