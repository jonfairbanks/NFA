package mocks

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// MockMarketplaceHandler simulates the marketplace node's Chat Completions API
func MockMarketplaceHandler(w http.ResponseWriter, r *http.Request) {
    // Check for session_id header
    sessionID := r.Header.Get("session_id")
    if sessionID == "" {
        http.Error(w, `{"error": "Missing session_id"}`, http.StatusUnauthorized)
        return
    }

    // Decode the request body
    var requestBody map[string]interface{}
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&requestBody)
    if err != nil {
        http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
        return
    }

    // Check if "stream" parameter is true
    stream, ok := requestBody["stream"].(bool)
    if !ok {
        stream = false // Default to non-streaming if not specified
    }

    if stream {
        handleStreamingResponse(w)
    } else {
        handleNonStreamingResponse(w)
    }
}

func handleStreamingResponse(w http.ResponseWriter) {
    // Set headers for streaming
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.WriteHeader(http.StatusOK)

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, `{"error": "Streaming unsupported"}`, http.StatusInternalServerError)
        return
    }

    // Simulate streaming responses
    messages := []string{
        `data: {"choices":[{"delta":{"content":"Hello"}}]}`,
        `data: {"choices":[{"delta":{"content":" world!"}}]}`,
        `data: [DONE]`,
    }

    for _, msg := range messages {
        fmt.Fprintf(w, "%s\n\n", msg)
        flusher.Flush()
    }
}

func handleNonStreamingResponse(w http.ResponseWriter) {
    response := map[string]interface{}{
        "id":      "cmpl-123",
        "object":  "text_completion",
        "created": 1610078131,
        "model":   "gpt-3.5-turbo",
        "choices": []map[string]interface{}{
            {
                "text":          "Hello world!",
                "index":         0,
                "logprobs":      nil,
                "finish_reason": "length",
            },
        },
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}