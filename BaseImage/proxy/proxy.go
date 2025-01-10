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
    baseURL := os.Getenv("MARKETPLACE_BASE_URL")
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
    return os.Getenv("MARKETPLACE_URL")
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
	modelCache     = make(map[string]string) // map[modelHandle]modelId
	modelCacheMux  sync.RWMutex
	sessionExpirationSeconds = getSessionExpirationSeconds()
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

	// Add periodic cleanup of expired sessions
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			cleanupExpiredSessions()
		}
	}()
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

// Modify ensureSession to be more robust
func ensureSession(modelID string) error {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	// Clean up expired sessions first
	cleanupExpiredSessions()

	session, exists := activeSessions[modelID]
	if exists && session.SessionID != "" {
		// Check if session is still valid using configurable expiration
		if time.Since(session.LastUsed) < time.Duration(sessionExpirationSeconds)*time.Second {
			session.LastUsed = time.Now() // Update last used time
			SessionManagerInstance.UpdateSession(session.SessionID, modelID)
			log.Printf("Using existing session for model %s: %s", modelID, session.SessionID)
			return nil
		} else {
			// Session expired, remove it
			delete(activeSessions, modelID)
			log.Printf("Removed expired session for model %s", modelID)
		}
	}

	// Create new session
	log.Printf("Creating new session for model %s", modelID)

	reqBody := map[string]interface{}{
		"sessionDuration": 3600,
		"failover":        false,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal session request: %v", err)
	}

	sessionURL := getMarketplaceSessionEndpoint(modelID)
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

	if result.Id == "" {
		return fmt.Errorf("failed to get valid session ID from response")
	}

	// Update the session for the model ID
	activeSessions[modelID] = &MorpheusSession{
		SessionID: result.Id,
		ModelID:   modelID,
		LastUsed:  time.Now(),
	}

	// Update the global session manager
	SessionManagerInstance.UpdateSession(result.Id, modelID)

	log.Printf("Successfully established new session for model %s: %s", modelID, result.Id)
	return nil
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
	modelCacheMux.RLock()
	if modelID, exists := modelCache[modelHandle]; exists {
		modelCacheMux.RUnlock()
		log.Printf("Found cached model ID for '%s': %s", modelHandle, modelID)
		return modelID, nil
	}
	modelCacheMux.RUnlock()

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
    fmt.Println("\n---\n\n")

	// Update SessionManager with the new or existing session ID
	SessionManagerInstance.UpdateSession(activeSessions[modelID].SessionID, modelID)

	// Update request with actual model ID
	requestBody["model"] = modelID

	// Add session_id to forwarded request headers
	r.Header.Set("session_id", sessionId)

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
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	http.HandleFunc("/v1/chat/completions", ProxyChatCompletion)

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("DEFAULT_PORT")
		if port == "" {
			port = "8080"
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
		if time.Since(session.LastUsed) > time.Duration(sessionExpirationSeconds)*time.Second {
			delete(activeSessions, modelID)
			log.Printf("Cleaned up expired session for model %s", modelID)
		}
	}
}
