package marketplacesdk

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
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
func (m *MockApiGatewayClient) ApproveAllowance(ctx context.Context, spender string, amount *big.Int) (*TransactionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowance = new(big.Int).Set(amount)
	return &TransactionResponse{
		TxHash: "0x123mock",
	}, nil
}

// OpenStakeSession mocks the OpenStakeSession method
func (m *MockApiGatewayClient) OpenStakeSession(ctx context.Context, request *SessionStakeRequest) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &Session{
		SessionID: m.sessionID,
	}, nil
}

// PromptStream mocks the PromptStream method
func (m *MockApiGatewayClient) PromptStream(ctx context.Context, request *openai.ChatCompletionRequest, modelID string, sessionID string, callback CompletionCallback) error {
	// Simulate a streaming response
	response := &openai.ChatCompletionStreamResponse{
		ID:      "test-response-id",
		Object:  "chat.completion.chunk",
		Created: 1234567890,
		Model:   modelID,
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
	callback(response)
	return nil
}
// Helper functions

func SetupTestEnvironment() (*ecdsa.PrivateKey, string, string, *httptest.Server) {
	privateKey, _ := crypto.GenerateKey()
	agentWalletPKHex := crypto.FromECDSA(privateKey)
	agentWalletAddress := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Implement mock server logic...
	}))

	return privateKey, hex.EncodeToString(agentWalletPKHex), agentWalletAddress, mockServer
}

func SetupGinRouter(client ApiGatewayClientInterface) *gin.Engine {
	router := gin.Default()
	// Implement router setup...
	return router
}

// Add more shared utility functions as needed...