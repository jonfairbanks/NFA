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
	"strconv"
	"strings"
	"sync"
	"time"

	"sort"

	"github.com/sony/gobreaker"
)

// Add these new environment variable getters at the top of the file
func getMarketplaceBaseURL() string {
    baseURL := os.Getenv("MARKETPLACE_URL")
    if baseURL == "" {
        baseURL = "http://marketplace:9000"
    }
    return baseURL
}

func getMarketplaceModelsEndpoint() string {
    return fmt.Sprintf("%s/blockchain/models", getMarketplaceBaseURL())
}

func getMarketplaceSessionEndpoint(modelID string) string {
    return fmt.Sprintf("%s/blockchain/models/%s/session", getMarketplaceBaseURL(), modelID)
}

func getMarketplaceChatEndpoint() string {
    marketplaceURL := os.Getenv("MARKETPLACE_URL")
    if marketplaceURL == "" {
        return ""
    }
    return fmt.Sprintf("%s/chat/completions", marketplaceURL)
}

func getSessionExpirationSeconds() int {
	expirationStr := os.Getenv("SESSION_EXPIRATION_SECONDS")
	if expirationStr == "" {
		return 1800 // Default to 30 minutes
	}
	expiration, err := strconv.Atoi(expirationStr)
	if err != nil || expiration < 60 { // Minimum 1 minute
		log.Printf("Invalid SESSION_EXPIRATION_SECONDS value: %s, using default of 1800", expirationStr)
		return 1800
	}
	return expiration
}

// Update SessionManager to track model ID
type SessionManager struct {
	SessionID string
	ModelID   string
}

// Update GetSessionID to include model context
func (sm *SessionManager) GetSessionInfo() (string, string) {
	return sm.SessionID, sm.ModelID
}

// Update UpdateSessionID to track model
func (sm *SessionManager) UpdateSession(sessionID, modelID string) {
	sm.SessionID = sessionID
	sm.ModelID = modelID
}

// SessionManagerInstance is a global instance of SessionManager
var SessionManagerInstance = &SessionManager{}

// Add these new vars at the top of the file
var (
	defaultTimeout = 30 * time.Second
	circuitBreaker *gobreaker.CircuitBreaker
	sessionExpirationSeconds = getSessionExpirationSeconds()

	// Session and model caches with mutex protection
	sessionCache = struct {
		sync.RWMutex
		m map[string]CachedSession
	}{m: make(map[string]CachedSession)}

	modelCache = struct {
		sync.RWMutex
		m map[string]CachedModel
	}{m: make(map[string]CachedModel)}

	consumerNodeURL = getEnvOrDefault("CONSUMER_NODE_URL", "https://consumer-node-yalzemm5uq-uc.a.run.app")

	// Add a flag to control cleanup goroutine
	enableCleanupGoroutine = true
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

	// Add periodic cleanup of expired sessions only if enabled
	if enableCleanupGoroutine {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			for range ticker.C {
				cleanupExpiredSessions()
			}
		}()
	}
}

// MorpheusSession represents a session with the Morpheus consumer node
type MorpheusSession struct {
	SessionID string
	ModelID   string
	ModelName string
	Created   time.Time
}

// Update activeSessions to manage sessions per model ID
var (
	activeSessions = make(map[string]*MorpheusSession)
	sessionMutex   sync.Mutex
)

// Remove getModelID function as modelID comes from the request

// Add retry configuration constants
const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
)

// Modify ensureSession to be more robust with retry logic
func ensureSession(modelID string) error {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	// Clean up expired sessions first
	cleanupExpiredSessions()

	// Get current session ID from SessionManagerInstance
	currentSessionID, currentModelID := SessionManagerInstance.GetSessionInfo()
	
	// If we have a current session but it's for a different model, we need a new session
	if currentSessionID != "" && currentModelID != modelID {
		log.Printf("Current session is for different model (current: %s, requested: %s). Creating new session.", currentModelID, modelID)
		delete(activeSessions, currentModelID)
		SessionManagerInstance.UpdateSession("", "") // Clear the current session
	}

	session, exists := activeSessions[modelID]
	if exists && session.SessionID != "" {
		// Check if session is still valid using configurable expiration
		if time.Since(session.Created) < time.Duration(sessionExpirationSeconds)*time.Second {
			session.Created = time.Now() // Update last used time
			SessionManagerInstance.UpdateSession(session.SessionID, modelID)
			log.Printf("Using existing session for model %s: %s", modelID, session.SessionID)
			return nil
		} else {
			// Session expired, remove it
			delete(activeSessions, modelID)
			log.Printf("Removed expired session for model %s", modelID)
		}
	}

	// Create new session with retry logic
	log.Printf("Creating new session for model %s", modelID)

	// Get model name from available models
	modelName := "" // Default empty
	if models, err := getModels(); err == nil {
		for _, model := range models {
			if model.Id == modelID {
				modelName = model.Name
				break
			}
		}
	}

	reqBody := map[string]interface{}{
		"sessionDuration": 3600,
		"failover":        false,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal session request: %v", err)
	}

	var result struct {
		Id string `json:"sessionID"`
	}

	// Implement retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			log.Printf("Retrying session creation (attempt %d/%d) after %v delay", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		sessionURL := getMarketplaceSessionEndpoint(modelID)
		resp, err := http.Post(sessionURL, "application/json", bytes.NewBuffer(reqBytes))
		if err != nil {
			lastErr = fmt.Errorf("failed to establish session: %v", err)
			log.Printf("Session establishment failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Check for nonce error in response
			var errorResp struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(bodyBytes, &errorResp); err == nil && strings.Contains(strings.ToLower(errorResp.Error), "nonce") {
				lastErr = fmt.Errorf("nonce error: %s", errorResp.Error)
				log.Printf("Nonce error detected (attempt %d/%d): %s", attempt+1, maxRetries, errorResp.Error)
				continue
			}

			lastErr = fmt.Errorf("failed to establish session: %s", string(bodyBytes))
			log.Printf("Session establishment failed with status %d (attempt %d/%d): %s", resp.StatusCode, attempt+1, maxRetries, string(bodyBytes))
			continue
		}

		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			lastErr = fmt.Errorf("failed to decode session response: %v", err)
			log.Printf("Failed to decode session response (attempt %d/%d): %v", attempt+1, maxRetries, err)
			continue
		}

		if result.Id == "" {
			lastErr = fmt.Errorf("failed to get valid session ID from response")
			log.Printf("Empty session ID received (attempt %d/%d)", attempt+1, maxRetries)
			continue
		}

		// Success! Update the session and return
		activeSessions[modelID] = &MorpheusSession{
			SessionID: result.Id,
			ModelID:   modelID,
			ModelName: modelName,
			Created:   time.Now(),
		}

		// Update the global session manager
		SessionManagerInstance.UpdateSession(result.Id, modelID)

		log.Printf("Successfully established new session for model %s: %s (attempt %d)", modelID, result.Id, attempt+1)
		return nil
	}

	// If we get here, all retries failed
	return fmt.Errorf("failed to establish session after %d attempts: %v", maxRetries, lastErr)
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

	// If either string is empty
	if len(s1) == 0 || len(s2) == 0 {
		if len(s1) == len(s2) {
			return 1.0 // both empty
		}
		return 0.0 // one is empty
	}

	// If strings are identical
	if s1 == s2 {
		return 1.0
	}

	// Calculate Levenshtein distance
	distance := levenshteinDistance(s1, s2)
	maxLen := float64(max(len(s1), len(s2)))

	// Calculate similarity score and round to nearest 0.1
	similarity := 1.0 - float64(distance)/maxLen
	return float64(int(similarity*10+0.5)) / 10.0
}

// Helper function to find maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Fix levenshteinDistance function
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Convert strings to runes for proper UTF-8 handling
	r1 := []rune(strings.ToLower(s1))
	r2 := []rune(strings.ToLower(s2))

	// Create matrix
	rows := len(r1) + 1
	cols := len(r2) + 1
	matrix := make([][]int, rows)
	for i := range matrix {
		matrix[i] = make([]int, cols)
		matrix[i][0] = i // Initialize first column
	}

	// Initialize first row
	for j := 0; j < cols; j++ {
		matrix[0][j] = j
	}

	// Fill rest of the matrix
	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[rows-1][cols-1]
}

// Add helper function for min of 3 numbers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// Update findModelID to be more robust
func findModelID(modelHandle string) (string, error) {
	// Add debug logging
	log.Printf("Attempting to find model ID for handle: '%s'", modelHandle)

	// Normalize input
	modelHandle = strings.TrimSpace(modelHandle)
	if modelHandle == "" {
		return "", fmt.Errorf("model handle cannot be empty")
	}

	// Check cache first
	modelCache.RLock()
	if cached, exists := modelCache.m[modelHandle]; exists && time.Since(cached.Created) < time.Hour {
		modelCache.RUnlock()
		log.Printf("Found cached model ID for '%s': %s", modelHandle, cached.ModelID)
		return cached.ModelID, nil
	}
	modelCache.RUnlock()

	endpoint := getMarketplaceModelsEndpoint()
	log.Printf("Fetching models from: %s", endpoint)

	// Query the marketplace API
	resp, err := http.Get(fmt.Sprintf("%s?limit=100&order=desc", endpoint))
	if err != nil {
		return "", fmt.Errorf("failed to fetch models: %v", err)
	}
	defer resp.Body.Close()

	// Read and log the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	log.Printf("Marketplace response: %s", string(bodyBytes))

	var searchResp ModelSearchResponse
	if err := json.Unmarshal(bodyBytes, &searchResp); err != nil {
		return "", fmt.Errorf("failed to decode models response: %v", err)
	}

	if len(searchResp.Models) == 0 {
		return "", fmt.Errorf("No Supported Model Has Been Registered")
	}

	// Log available models
	log.Printf("Available models:")
	for _, model := range searchResp.Models {
		log.Printf("- %s (ID: %s)", model.Name, model.Id)
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
		modelCache.Lock()
		modelCache.m[modelHandle] = CachedModel{
			ModelID:   bestMatch.id,
			ModelName: bestMatch.name,
			Created:   time.Now(),
		}
		modelCache.Unlock()

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
	modelCache.Lock()
	modelCache.m[modelHandle] = CachedModel{
		ModelID:   bestMatch,
		ModelName: modelHandle,
		Created:   time.Now(),
	}
	modelCache.Unlock()

	return bestMatch, nil
}

// validateModelHandle checks if the model handle is valid and returns the corresponding ID
func validateModelHandle(handle string) (string, error) {
	modelID, err := findModelID(handle)
	if err != nil {
		if err.Error() == "No Supported Model Has Been Registered" {
			return "", err
		}
		// For any other error, return the standard message
		return "", fmt.Errorf("No Supported Model Has Been Registered")
	}
	return modelID, nil
}

// Update ProxyChatCompletion to ensure proper model and session handling
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

	sessionId := activeSessions[modelID].SessionID

	fmt.Printf("--- Update Session with Model ID: %s\n ---", modelID)
	fmt.Printf("------ Session ID: %s\n ------", sessionId)
	fmt.Printf("------ Active Sessions: %+v\n ------", activeSessions)
	fmt.Print("\n---\n\n")

	// Update SessionManager with the new or existing session ID
	SessionManagerInstance.UpdateSession(activeSessions[modelID].SessionID, modelID)

	// Create a new request body with the model ID
	newRequestBody := make(map[string]interface{})
	for k, v := range requestBody {
		if k == "model" {
			// Get the model name from the matched model
			if models, err := getModels(); err == nil {
				for _, model := range models {
					if model.Id == modelID {
						newRequestBody[k] = model.Name
						break
					}
				}
			}
			if _, exists := newRequestBody[k]; !exists {
				newRequestBody[k] = modelHandle // Fallback to original handle
			}
		} else {
			newRequestBody[k] = v
		}
	}
	newRequestBody["model"] = modelID

	stream, ok := newRequestBody["stream"].(bool)
	if !ok {
		stream = false // Default to non-streaming if not specified
	}

	if stream {
		handleStreamingRequest(w, newRequestBody, modelID)
	} else {
		handleNonStreamingRequest(w, newRequestBody, modelID)
	}
}

// Modify forwardRequest to accept modelID and use the correct session
func forwardRequest(requestBody map[string]interface{}, modelID string) (*http.Response, error) {
	marketplaceURL := getMarketplaceChatEndpoint()
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
		req.Header.Set("session_id", session.SessionID)
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
	proxy := NewProxy()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Add handlers for blockchain/models endpoints
	http.HandleFunc("/blockchain/models", proxy.handleGetModels)
	http.HandleFunc("/blockchain/models/", proxy.handleModelOperations)
	http.HandleFunc("/v1/chat/completions", proxy.handleChatCompletions)

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("DEFAULT_PORT")
		if port == "" {
			port = "8081"
		}
	}
	log.Printf("Proxy server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Add a cleanup function for expired sessions
func cleanupExpiredSessions() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	for modelID, session := range activeSessions {
		if time.Since(session.Created) > time.Duration(sessionExpirationSeconds)*time.Second {
			delete(activeSessions, modelID)
			log.Printf("Cleaned up expired session for model %s", modelID)
		}
	}
}

func (p *Proxy) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
    log.Printf("Received chat completions request from %s", r.RemoteAddr)
    
    // Read and parse request body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("Error reading request body: %v", err)
        http.Error(w, "Error reading request body", http.StatusBadRequest)
        return
    }
    log.Printf("Raw request body: %s", string(body))

    var chatRequest ChatCompletionRequest
    if err := json.Unmarshal(body, &chatRequest); err != nil {
        log.Printf("Error parsing chat request: %v", err)
        http.Error(w, "Error parsing request body", http.StatusBadRequest)
        return
    }

    // Ensure stream is set to true
    chatRequest.Stream = true
    
    // Check for existing session ID in header using consistent header name
    sessionID := r.Header.Get("session_id")
    log.Printf("Session ID from header: %s", sessionID)
    
    if sessionID != "" {
        sessionCache.RLock()
        if session, exists := sessionCache.m[sessionID]; exists && time.Now().Before(session.ExpiresAt) {
            sessionCache.RUnlock()
            log.Printf("Using existing session: %s for model %s", sessionID, session.ModelID)
            if err := p.forwardChatRequest(w, r, session.ModelID, chatRequest, sessionID); err != nil {
                log.Printf("Error forwarding chat request: %v", err)
                http.Error(w, fmt.Sprintf("Error forwarding request: %v", err), http.StatusInternalServerError)
            }
            return
        }
        sessionCache.RUnlock()
        log.Printf("Session %s not found or expired", sessionID)
    }

    // Get model ID from request
    modelID, err := validateModelHandle(chatRequest.Model)
    if err != nil {
        log.Printf("Error validating model handle: %v", err)
        http.Error(w, fmt.Sprintf("Error finding model ID: %v", err), http.StatusBadRequest)
        return
    }
    log.Printf("Validated model ID: %s", modelID)

    // Create new session
    sessionID, err = p.createSession(modelID)
    if err != nil {
        log.Printf("Error creating session: %v", err)
        http.Error(w, fmt.Sprintf("Error creating session: %v", err), http.StatusInternalServerError)
        return
    }
    log.Printf("Created new session: %s", sessionID)

    if err := p.forwardChatRequest(w, r, modelID, chatRequest, sessionID); err != nil {
        log.Printf("Error forwarding chat request: %v", err)
        http.Error(w, fmt.Sprintf("Error forwarding request: %v", err), http.StatusInternalServerError)
    }
}

func (p *Proxy) findModelID(modelHandle string) (string, error) {
    // Check model cache first
    modelCache.RLock()
    if model, exists := modelCache.m[modelHandle]; exists && time.Since(model.Created) < time.Hour {
        modelCache.RUnlock()
        return model.ModelID, nil
    }
    modelCache.RUnlock()

    // Fetch models from consumer node
    models, err := p.getMarketplaceModels()
    if err != nil {
        return "", fmt.Errorf("error getting marketplace models: %v", err)
    }

    // Find best matching model
    var bestMatch string
    var bestScore float64
    log.Printf("Looking for model match: %s", modelHandle)
    
    for _, model := range models {
        score := calculateSimilarity(model.Name, modelHandle)
        log.Printf("Comparing '%s' with '%s', score: %.2f", model.Name, modelHandle, score)
        
        if score > bestScore {
            bestScore = score
            bestMatch = model.Id
            
            // Cache the match
            modelCache.Lock()
            modelCache.m[modelHandle] = CachedModel{
                ModelID:   model.Id,
                ModelName: model.Name,
                Created:   time.Now(),
            }
            modelCache.Unlock()
            
            log.Printf("New best match: '%s' (ID: %s) with score %.2f", model.Name, model.Id, score)
        }
    }

    if bestScore >= 0.5 {
        return bestMatch, nil
    }

    return "", fmt.Errorf("no supported model has been registered")
}

func (p *Proxy) createSession(modelID string) (string, error) {
    log.Printf("Creating new session for model ID: %s", modelID)
    
    endpoint := fmt.Sprintf("%s/blockchain/models/%s/session", p.getMarketplaceBaseURL(), modelID)
    log.Printf("Session creation endpoint: %s", endpoint)
    
    reqBody := map[string]interface{}{
        "sessionDuration": sessionExpirationSeconds,
        "failover": false,
    }
    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        log.Printf("Error marshaling session request: %v", err)
        return "", err
    }

    req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
    if err != nil {
        log.Printf("Error creating session request: %v", err)
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    log.Printf("Sending session creation request with body: %s", string(jsonBody))
    
    resp, err := p.client.Do(req)
    if err != nil {
        log.Printf("Error sending session request: %v", err)
        return "", err
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Error reading response body: %v", err)
        return "", err
    }
    log.Printf("Received response with status %d: %s", resp.StatusCode, string(respBody))

    if resp.StatusCode != http.StatusOK {
        log.Printf("Received non-200 status code: %d", resp.StatusCode)
        return "", fmt.Errorf("failed to create session, status: %d, body: %s", resp.StatusCode, string(respBody))
    }

    var result struct {
        SessionID string `json:"sessionId"`
    }
    if err := json.Unmarshal(respBody, &result); err != nil {
        log.Printf("Error decoding session response: %v", err)
        return "", err
    }

    if result.SessionID == "" {
        log.Printf("No sessionId in response. Full response: %s", string(respBody))
        return "", fmt.Errorf("no sessionId in response")
    }
    
    // Cache the session
    sessionCache.Lock()
    sessionCache.m[result.SessionID] = CachedSession{
        SessionID:  result.SessionID,
        ModelID:    modelID,
        ExpiresAt:  time.Now().Add(time.Duration(sessionExpirationSeconds) * time.Second),
    }
    sessionCache.Unlock()
    
    log.Printf("Successfully created and cached session with ID: %s", result.SessionID)
    return result.SessionID, nil
}

// Add cleanup function for sessions
func (p *Proxy) cleanupSession(sessionID string, modelID string) error {
    if sessionID == "" || modelID == "" {
        return fmt.Errorf("invalid session ID or model ID")
    }

    endpoint := fmt.Sprintf("%s/blockchain/models/%s/session/%s", p.getMarketplaceBaseURL(), modelID, sessionID)
    req, err := http.NewRequest("DELETE", endpoint, nil)
    if err != nil {
        return fmt.Errorf("error creating cleanup request: %v", err)
    }

    resp, err := p.client.Do(req)
    if err != nil {
        return fmt.Errorf("error sending cleanup request: %v", err)
    }
    defer resp.Body.Close()

    // Log cleanup attempt
    log.Printf("Session cleanup for ID %s completed with status: %d", sessionID, resp.StatusCode)
    return nil
}

type ChatCompletionRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
    Stream   bool      `json:"stream"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type MarketplaceModel struct {
    Id   string `json:"id"`
    Name string `json:"name"`
}

// getMatchScore returns a similarity score between 0 and 1 for two strings
func getMatchScore(a, b string) float32 {
    a = strings.ToLower(a)
    b = strings.ToLower(b)
    if a == b {
        return 1.0
    }
    // Simple contains check
    if strings.Contains(a, b) || strings.Contains(b, a) {
        return 0.8
    }
    return 0.0
}

type Proxy struct {
	client *http.Client
}

func NewProxy() *Proxy {
	return &Proxy{
		client: &http.Client{},
	}
}

func (p *Proxy) getMarketplaceBaseURL() string {
    if url := os.Getenv("MARKETPLACE_URL"); url != "" {
        return url
    }
    return "https://consumer-node-yalzemm5uq-uc.a.run.app"
}

func (p *Proxy) getMarketplaceModels() ([]MarketplaceModel, error) {
    endpoint := fmt.Sprintf("%s/blockchain/models", p.getMarketplaceBaseURL())
    log.Printf("Fetching models from: %s", endpoint)
    
    req, err := http.NewRequest("GET", endpoint, nil)
    if err != nil {
        log.Printf("Error creating request: %v", err)
        return nil, err
    }

    resp, err := p.client.Do(req)
    if err != nil {
        log.Printf("Error sending request: %v", err)
        return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Error reading response body: %v", err)
        return nil, err
    }
    log.Printf("Raw response body: %s", string(body))

    if resp.StatusCode != http.StatusOK {
        log.Printf("Received non-200 status code: %d", resp.StatusCode)
        return nil, fmt.Errorf("failed to fetch models, status: %d", resp.StatusCode)
    }

    var result struct {
        Models []MarketplaceModel `json:"models"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Error parsing response: %v", err)
        return nil, err
    }

    log.Printf("Successfully parsed models: %+v", result.Models)
    return result.Models, nil
}

func (p *Proxy) forwardChatRequest(w http.ResponseWriter, r *http.Request, modelID string, req ChatCompletionRequest, sessionID string) error {
    // Create request body with original model name (not ID) to match consumer node expectation
    reqBody := map[string]interface{}{
        "model":    req.Model, // Use original model name
        "messages": req.Messages,
        "stream":   true, // Always set to true
    }

    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return fmt.Errorf("error marshaling request body: %v", err)
    }

    // Create the request
    endpoint := fmt.Sprintf("%s/v1/chat/completions", p.getMarketplaceBaseURL())
    proxyReq, err := http.NewRequest(r.Method, endpoint, bytes.NewBuffer(jsonBody))
    if err != nil {
        return fmt.Errorf("error creating request: %v", err)
    }

    // Set required headers
    proxyReq.Header.Set("Content-Type", "application/json")
    proxyReq.Header.Set("Accept", "application/json")
    proxyReq.Header.Set("session_id", sessionID) // Use consistent session_id header

    // Log request details
    log.Printf("Forwarding request to: %s", endpoint)
    log.Printf("Request headers: %v", proxyReq.Header)
    log.Printf("Request body: %s", string(jsonBody))

    // Send the request with increased timeout
    client := &http.Client{
        Timeout: time.Minute * 5,
    }
    resp, err := client.Do(proxyReq)
    if err != nil {
        return fmt.Errorf("error sending request: %v", err)
    }
    defer resp.Body.Close()

    // Check response status
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("marketplace request failed, status: %d, response: %s", resp.StatusCode, string(body))
    }

    // Set streaming headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.WriteHeader(http.StatusOK)

    // Stream the response
    reader := bufio.NewReader(resp.Body)
    for {
        line, err := reader.ReadBytes('\n')
        if err != nil {
            if err == io.EOF {
                break
            }
            return fmt.Errorf("error reading stream: %v", err)
        }
        if _, err := w.Write(line); err != nil {
            return fmt.Errorf("error writing stream: %v", err)
        }
        if f, ok := w.(http.Flusher); ok {
            f.Flush()
        }
    }

    return nil
}

// Add these new types for better model and session management
type CachedSession struct {
    SessionID  string
    ModelID    string
    ExpiresAt  time.Time
}

type CachedModel struct {
    ModelID   string
    ModelName string
    Created   time.Time
}

// Model represents a model from the consumer node
type Model struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}

// getModels fetches the list of available models from the consumer node
func getModels() ([]Model, error) {
	modelsURL := fmt.Sprintf("%s/blockchain/models", consumerNodeURL)
	resp, err := http.Get(modelsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch models: %s", string(bodyBytes))
	}

	var result struct {
		Models []Model `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %v", err)
	}

	return result.Models, nil
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Add handler for getting models
func (p *Proxy) handleGetModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	marketplaceURL := getMarketplaceModelsEndpoint()
	req, err := http.NewRequest(http.MethodGet, marketplaceURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch models", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w, resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// Add handler for model operations (session creation/deletion)
func (p *Proxy) handleModelOperations(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling model operation: %s %s", r.Method, r.URL.Path)
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/blockchain/models/"), "/")
	if len(pathParts) < 1 {
		log.Printf("Invalid path: %s", r.URL.Path)
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	marketplaceURL := fmt.Sprintf("%s/%s", getMarketplaceModelsEndpoint(), strings.Join(pathParts, "/"))
	log.Printf("Forwarding to marketplace URL: %s", marketplaceURL)

	// Forward the request to the marketplace
	req, err := http.NewRequest(r.Method, marketplaceURL, r.Body)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	log.Printf("Request headers: %v", req.Header)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to forward request: %v", err)
		http.Error(w, "Failed to forward request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Log response details
	body, _ := io.ReadAll(resp.Body)
	log.Printf("Response status: %d", resp.StatusCode)
	log.Printf("Response body: %s", string(body))

	copyHeaders(w, resp.Header)
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
