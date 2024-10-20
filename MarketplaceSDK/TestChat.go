// TestChat.go

package marketplacesdk

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
)

// MockApiGatewayClient is a mock implementation of the ApiGatewayClientInterface
type MockApiGatewayClient struct {
	mu        sync.Mutex
	allowance *big.Int
	sessionID string
}

func NewMockApiGatewayClient() *MockApiGatewayClient {
	return &MockApiGatewayClient{
		allowance: big.NewInt(0),
		sessionID: "test-session-id",
	}
}

// GetAllowance mocks the GetAllowance method
func (m *MockApiGatewayClient) GetAllowance(ctx context.Context, spender string) (*big.Int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return new(big.Int).Set(m.allowance), nil
}

// ApproveAllowance mocks the ApproveAllowance method
func (m *MockApiGatewayClient) ApproveAllowance(ctx context.Context, spender string, amount *big.Int) (*ApproveResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowance = new(big.Int).Set(amount)
	return &ApproveResponse{
		Success: true,
	}, nil
}

// OpenStakeSession mocks the OpenStakeSession method
func (m *MockApiGatewayClient) OpenStakeSession(ctx context.Context, request *SessionStakeRequest) (*SessionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &SessionResponse{
		SessionID: m.sessionID,
	}, nil
}

// PromptStream mocks the PromptStream method
func (m *MockApiGatewayClient) PromptStream(ctx context.Context, request *openai.ChatCompletionRequest, modelID string, sessionID string, callback CompletionCallback) error {
	// Simulate a streaming response from OpenAI
	response := openai.ChatCompletionStreamResponse{
		ID:      "test-response-id",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(), // Corrected to int64
		Model:   modelID,
		Choices: []openai.ChatCompletionStreamChoice{
			{
				Delta: openai.ChatCompletionStreamChoiceDelta{
					Content: "This is a test response.",
				},
				FinishReason: "stop",
				Index:        0,
			},
		},
	}
	callback(&response)
	return nil
}

// ensureAllowance ensures that the allowance is sufficient by approving if necessary.
func ensureAllowance(ctx context.Context, stakeAmount *big.Int, client ApiGatewayClientInterface) error {
	// In the mock, you can simulate approval.
	_, err := client.ApproveAllowance(ctx, "spender-address", stakeAmount)
	return err
}

// signData signs the given data using the provided private key.
func signData(data []byte, privateKey *ecdsa.PrivateKey) (string, error) {
	signature, err := crypto.Sign(crypto.Keccak256(data), privateKey)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(signature), nil
}

func TestChatIntegration(t *testing.T) {
	// Initialize Gin in test mode
	gin.SetMode(gin.TestMode)

	// Generate a test private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}
	agentWalletPKHex := crypto.FromECDSA(privateKey)
	agentWalletAddress := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	// Create a mock Morpheus-Lumerin-Node server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/allowance":
			// Mock response for GetAllowance
			response := AllowanceResponse{
				Allowance: "0",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		case "/approve":
			// Mock response for ApproveAllowance
			var approveReq ApproveRequest
			if err := json.NewDecoder(r.Body).Decode(&approveReq); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			// Simulate successful approval
			response := ApproveResponse{
				Success: true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		case "/sessions":
			// Mock response for OpenStakeSession
			var sessionReq SessionStakeRequest
			if err := json.NewDecoder(r.Body).Decode(&sessionReq); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			// Simulate successful session creation
			response := SessionResponse{
				SessionID: "mock-session-id",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		case "/v1/chat/completions":
			// Mock response for PromptStream
			var chatReq ChatRequest
			if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			// Simulate streaming response
			response := openai.ChatCompletionStreamResponse{
				ID:      "mock-response-id",
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(), // Corrected to int64
				Model:   chatReq.Model,
				Choices: []openai.ChatCompletionStreamChoice{
					{
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Content: "This is a mocked response from Morpheus-Lumerin-Node.",
						},
						FinishReason: "stop",
						Index:        0,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Set environment variables for the test
	os.Setenv("AGENT_ID", "test-agent-id")
	os.Setenv("AGENT_WALLET_PK", fmt.Sprintf("%x", agentWalletPKHex))
	os.Setenv("AGENT_WALLET_ADDRESS", agentWalletAddress)
	os.Setenv("API_BASE_URL", mockServer.URL)

	// Initialize the mock ApiGatewayClient
	mockClient := NewMockApiGatewayClient()
	var client ApiGatewayClientInterface = mockClient

	// Initialize the Gin router as per main.go
	router := gin.Default()

	router.POST("/v1/chat/completions", func(c *gin.Context) {
		// Parse the request body
		var chatRequest ChatRequest
		if err := c.ShouldBindJSON(&chatRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		modelID := chatRequest.Model

		// Define the stake amount (e.g., 1 MOR in wei)
		stakeAmount := big.NewInt(1e18) // 1 MOR

		// Check allowance and approve if necessary
		err := ensureAllowance(c.Request.Context(), stakeAmount, client)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to ensure allowance: %v", err)})
			return
		}

		// Create approval data
		approvalData := []byte("Approve session") // Placeholder data
		approvalSig, err := signData(approvalData, privateKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to sign approval: %v", err)})
			return
		}

		// Create OpenSessionRequest
		openSessionRequest := &SessionStakeRequest{
			Approval:    string(approvalData),
			ApprovalSig: approvalSig,
			Stake:       stakeAmount,
		}

		// Open session using the mock client
		sessionResponse, err := client.OpenStakeSession(c.Request.Context(), openSessionRequest)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to open session: %v", err)})
			return
		}

		currentSessionID := sessionResponse.SessionID

		// Prepare the chat completion request
		chatCompletionRequest := &openai.ChatCompletionRequest{
			Model:    chatRequest.Model,
			Messages: []openai.ChatCompletionMessage{},
			Stream:   chatRequest.Stream,
		}

		// Map messages from ChatRequest to ChatCompletionRequest
		for _, msg := range chatRequest.Messages {
			chatCompletionRequest.Messages = append(chatCompletionRequest.Messages, openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		// Set response headers for SSE
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// Flush the headers
		c.Writer.Flush()

		// Use the PromptStream method to get the streaming response
		err = client.PromptStream(c.Request.Context(), chatCompletionRequest, modelID, currentSessionID, func(response interface{}) {
			// Handle the streamed response
			if completion, ok := response.(*openai.ChatCompletionStreamResponse); ok {
				// Prepare SSE event
				data, err := json.Marshal(completion)
				if err != nil {
					// Handle error
					log.Printf("Error marshalling completion: %v", err)
					return
				}
				// Write data to the client
				fmt.Fprintf(c.Writer, "data: %s\n\n", data)
				c.Writer.Flush()
			}
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error during prompt stream: %v", err)})
			return
		}
	})

	// Start the Gin server in a goroutine
	server := httptest.NewServer(router)
	defer server.Close()

	// Prepare the chat request
	chatReq := ChatRequest{
		Model: "gpt-4",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
		Stream: false,
	}

	// Marshal the chat request to JSON
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		t.Fatalf("Failed to marshal chat request: %v", err)
	}

	// Send the POST request to /v1/chat/completions
	resp, err := http.Post(fmt.Sprintf("%s/v1/chat/completions", server.URL), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var openAIResp openai.ChatCompletionStreamResponse
	err = json.Unmarshal(body, &openAIResp)
	if err != nil {
		t.Fatalf("Failed to unmarshal OpenAI response: %v", err)
	}

	// Check the response content
	expectedContent := "This is a mocked response from Morpheus-Lumerin-Node."
	if openAIResp.Choices[0].Delta.Content != expectedContent {
		t.Fatalf("Expected response to be '%s', got '%s'", expectedContent, openAIResp.Choices[0].Delta.Content)
	}

	// Optionally, verify that the session was created and allowance was approved
	expectedAllowance := big.NewInt(1e18) // 1 MOR
	if mockClient.allowance.Cmp(expectedAllowance) != 0 {
		t.Fatalf("Expected allowance to be %s, got %s", expectedAllowance.String(), mockClient.allowance.String())
	}

	if mockClient.sessionID != "mock-session-id" {
		t.Fatalf("Expected session ID to be 'mock-session-id', got '%s'", mockClient.sessionID)
	}
}
