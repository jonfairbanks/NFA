package marketplacesdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
)

func TestChat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, agentWalletPKHex, agentWalletAddress, mockServer := SetupTestEnvironment()
	defer mockServer.Close()

	os.Setenv("AGENT_ID", "test-agent-id")
	os.Setenv("AGENT_WALLET_PK", agentWalletPKHex)
	os.Setenv("AGENT_WALLET_ADDRESS", agentWalletAddress)
	os.Setenv("API_BASE_URL", mockServer.URL)

	mockClient := NewMockApiGatewayClient()
	router := SetupGinRouter(mockClient)

	server := httptest.NewServer(router)
	defer server.Close()

	chatReq := ChatRequest{
		Model: "gpt-4",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "user",
				Content: "Tell me a joke.",
			},
		},
		Stream: false,
	}

	reqBody, _ := json.Marshal(chatReq)
	resp, err := http.Post(fmt.Sprintf("%s/v1/chat/completions", server.URL), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp openai.ChatCompletionStreamResponse
	json.Unmarshal(body, &openAIResp)

	// Add specific assertions for TestChat...
}