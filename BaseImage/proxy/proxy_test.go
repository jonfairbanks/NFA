package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sony/gobreaker"
)

// MockMarketplaceHandler simulates the marketplace node's response
func MockMarketplaceHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "data: {\"message\": \"Hello\"}")
    fmt.Fprintln(w, "data: {\"message\": \"World\"}")
}

func TestProxyChatCompletionStreaming(t *testing.T) {
    // Set up mock marketplace server
    marketplaceServer := httptest.NewServer(http.HandlerFunc(MockMarketplaceHandler))
    defer marketplaceServer.Close()

    // Override the MARKETPLACE_URL environment variable
    os.Setenv("MARKETPLACE_URL", marketplaceServer.URL)
    defer os.Unsetenv("MARKETPLACE_URL")

    // Initialize session manager
    sessionManager.UpdateSessionID("test_session_id")

    // Set up proxy server
    proxy := httptest.NewServer(http.HandlerFunc(ProxyChatCompletion))
    defer proxy.Close()

    // Create streaming request
    requestBody := map[string]interface{}{
        "model": "gpt-3.5-turbo",
        "messages": []map[string]string{
            {"role": "user", "content": "Hello"},
        },
        "stream": true,
    }
    requestBytes, _ := json.Marshal(requestBody)

    resp, err := http.Post(proxy.URL+"/v1/chat/completions", "application/json", bytes.NewReader(requestBytes))
    if err != nil {
        t.Fatalf("Failed to send request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("Expected status 200, got %d", resp.StatusCode)
    }

    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        t.Fatalf("Failed to read response body: %v", err)
    }

    // Simple check to ensure data was streamed
    bodyString := string(bodyBytes)
    if !strings.Contains(bodyString, "Hello") || !strings.Contains(bodyString, "World") {
        t.Errorf("Streaming response does not contain expected messages")
    }
}

func TestProxyChatCompletionWithTimeout(t *testing.T) {
    // Create a server that delays response
    slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(defaultTimeout + time.Second)
        w.WriteHeader(http.StatusOK)
    }))
    defer slowServer.Close()

    os.Setenv("MARKETPLACE_URL", slowServer.URL)
    defer os.Unsetenv("MARKETPLACE_URL")

    req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"stream":false}`))
    w := httptest.NewRecorder()

    ProxyChatCompletion(w, req)

    if w.Code != http.StatusGatewayTimeout {
        t.Errorf("Expected timeout status 504, got %d", w.Code)
    }
}

func TestCircuitBreaker(t *testing.T) {
    // Create a server that always fails
    failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
    }))
    defer failingServer.Close()

    os.Setenv("MARKETPLACE_URL", failingServer.URL)
    defer os.Unsetenv("MARKETPLACE_URL")

    // Make multiple requests to trigger circuit breaker
    for i := 0; i < 5; i++ {
        req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"stream":false}`))
        w := httptest.NewRecorder()
        ProxyChatCompletion(w, req)
    }

    // Verify circuit breaker is open
    state := circuitBreaker.State()
    if state != gobreaker.StateOpen {
        t.Errorf("Expected circuit breaker to be open, got %v", state)
    }
}
