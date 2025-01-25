package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func init() {
	// Disable cleanup goroutine during tests
	enableCleanupGoroutine = false
}

func TestGetMarketplaceBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "with environment variable set",
			envValue: "http://custom-marketplace:8080",
			want:     "http://custom-marketplace:8080",
		},
		{
			name:     "without environment variable",
			envValue: "",
			want:     "http://marketplace:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("MARKETPLACE_URL", tt.envValue)
				defer os.Unsetenv("MARKETPLACE_URL")
			} else {
				os.Unsetenv("MARKETPLACE_URL")
			}

			got := getMarketplaceBaseURL()
			if got != tt.want {
				t.Errorf("getMarketplaceBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSessionManager(t *testing.T) {
	sm := &SessionManager{}

	// Test initial state
	sessionID, modelID := sm.GetSessionInfo()
	if sessionID != "" || modelID != "" {
		t.Errorf("Initial state should be empty, got sessionID=%s, modelID=%s", sessionID, modelID)
	}

	// Test updating session
	sm.UpdateSession("test-session", "test-model")
	sessionID, modelID = sm.GetSessionInfo()
	if sessionID != "test-session" || modelID != "test-model" {
		t.Errorf("Session not updated correctly, got sessionID=%s, modelID=%s", sessionID, modelID)
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected float64
	}{
		{"hello", "hello", 1.0},
		{"hello", "world", 0.2},
		{"", "", 1.0},
		{"test", "TEST", 1.0},
		{"gpt4", "gpt-4", 0.8},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s", tt.s1, tt.s2), func(t *testing.T) {
			got := calculateSimilarity(tt.s1, tt.s2)
			if got != tt.expected {
				t.Errorf("calculateSimilarity(%s, %s) = %v, want %v", tt.s1, tt.s2, got, tt.expected)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"hello", "hello", 0},
		{"hello", "world", 4},
		{"", "", 0},
		{"test", "TEST", 0},
		{"gpt4", "gpt-4", 1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s", tt.s1, tt.s2), func(t *testing.T) {
			got := levenshteinDistance(tt.s1, tt.s2)
			if got != tt.expected {
				t.Errorf("levenshteinDistance(%s, %s) = %v, want %v", tt.s1, tt.s2, got, tt.expected)
			}
		})
	}
}

func TestEnsureSession(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/blockchain/models/test-model/session" {
			json.NewEncoder(w).Encode(map[string]string{
				"sessionID": "test-session-id",
			})
		}
	}))
	defer server.Close()

	// Set environment variable for test
	os.Setenv("MARKETPLACE_URL", server.URL)
	defer os.Unsetenv("MARKETPLACE_URL")

	// Test creating new session
	err := ensureSession("test-model")
	if err != nil {
		t.Errorf("ensureSession() error = %v", err)
	}

	// Verify session was created
	session, exists := activeSessions["test-model"]
	if !exists {
		t.Error("Session was not created")
	}
	if session.SessionID != "test-session-id" {
		t.Errorf("Session ID mismatch, got %s, want test-session-id", session.SessionID)
	}
}

func TestFindModelID(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string][]ModelInfo{
			"models": {
				{Id: "model1", Name: "GPT-4"},
				{Id: "model2", Name: "GPT-3.5"},
			},
		})
	}))
	defer server.Close()

	// Set environment variable for test
	os.Setenv("MARKETPLACE_URL", server.URL)
	defer os.Unsetenv("MARKETPLACE_URL")

	tests := []struct {
		name        string
		modelHandle string
		wantID      string
		wantErr     bool
	}{
		{
			name:        "exact match",
			modelHandle: "GPT-4",
			wantID:      "model1",
			wantErr:     false,
		},
		{
			name:        "no match",
			modelHandle: "nonexistent-model",
			wantID:      "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := findModelID(tt.modelHandle)
			if (err != nil) != tt.wantErr {
				t.Errorf("findModelID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("findModelID() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestProxyChatCompletion(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/blockchain/models":
			json.NewEncoder(w).Encode(map[string][]ModelInfo{
				"models": {{Id: "test-model", Name: "Test Model"}},
			})
		case "/blockchain/models/test-model/session":
			json.NewEncoder(w).Encode(map[string]string{
				"sessionID": "test-session",
			})
		case "/chat/completions":
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"choices\": [{\"text\": \"test response\"}]}\n\n")
		}
	}))
	defer server.Close()

	// Set environment variable for test
	os.Setenv("MARKETPLACE_URL", server.URL)
	defer os.Unsetenv("MARKETPLACE_URL")

	// Create test request
	reqBody := map[string]interface{}{
		"model":    "Test Model",
		"messages": []map[string]string{{"role": "user", "content": "Hello"}},
	}
	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Call the handler
	ProxyChatCompletion(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.StatusCode)
	}

	// Verify response contains streaming data
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "test response") {
		t.Errorf("Response does not contain expected content")
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	// Setup test sessions
	activeSessions = make(map[string]*MorpheusSession)
	activeSessions["model1"] = &MorpheusSession{
		SessionID: "session1",
		ModelID:   "model1",
		Created:   time.Now().Add(-2 * time.Hour),
	}
	activeSessions["model2"] = &MorpheusSession{
		SessionID: "session2",
		ModelID:   "model2",
		Created:   time.Now(),
	}

	// Run cleanup
	cleanupExpiredSessions()

	// Verify results
	if _, exists := activeSessions["model1"]; exists {
		t.Error("Expired session was not cleaned up")
	}
	if _, exists := activeSessions["model2"]; !exists {
		t.Error("Valid session was incorrectly cleaned up")
	}
}

func TestValidateModelHandle(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string][]ModelInfo{
			"models": {
				{Id: "valid-id", Name: "Valid Model"},
			},
		})
	}))
	defer server.Close()

	// Set environment variable for test
	os.Setenv("MARKETPLACE_URL", server.URL)
	defer os.Unsetenv("MARKETPLACE_URL")

	tests := []struct {
		name    string
		handle  string
		wantID  string
		wantErr bool
	}{
		{
			name:    "valid model",
			handle:  "Valid Model",
			wantID:  "valid-id",
			wantErr: false,
		},
		{
			name:    "invalid model",
			handle:  "Invalid Model",
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := validateModelHandle(tt.handle)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateModelHandle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("validateModelHandle() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestGetSessionExpirationSeconds(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		expected  int
		shouldSet bool
	}{
		{
			name:      "default value",
			envValue:  "",
			expected:  1800,
			shouldSet: false,
		},
		{
			name:      "custom value",
			envValue:  "3600",
			expected:  3600,
			shouldSet: true,
		},
		{
			name:      "invalid value",
			envValue:  "invalid",
			expected:  1800,
			shouldSet: true,
		},
		{
			name:      "too small value",
			envValue:  "30",
			expected:  1800,
			shouldSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv("SESSION_EXPIRATION_SECONDS", tt.envValue)
				defer os.Unsetenv("SESSION_EXPIRATION_SECONDS")
			} else {
				os.Unsetenv("SESSION_EXPIRATION_SECONDS")
			}

			got := getSessionExpirationSeconds()
			if got != tt.expected {
				t.Errorf("getSessionExpirationSeconds() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMin3(t *testing.T) {
	tests := []struct {
		a, b, c  int
		expected int
	}{
		{1, 2, 3, 1},
		{3, 2, 1, 1},
		{2, 1, 3, 1},
		{0, 0, 0, 0},
		{-1, -2, -3, -3},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d,%d", tt.a, tt.b, tt.c), func(t *testing.T) {
			got := min3(tt.a, tt.b, tt.c)
			if got != tt.expected {
				t.Errorf("min3(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, got, tt.expected)
			}
		})
	}
}
