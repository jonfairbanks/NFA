package marketplacesdk

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestMarketplaceE2E(t *testing.T) {
	// Initialize test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Initialize the marketplace client
	client := NewApiGatewayClient("http://localhost:8082", httpClient)

	// Set up test wallet with private key
	testPrivateKey := "6f3453bc036158833281b9e0204975bad0f2a4407c4ab32132bb0f9e6e692cbb"
	walletResp, err := client.CreateWallet(ctx, testPrivateKey)
	assert.NoError(t, err)
	assert.NotNil(t, walletResp)
	assert.NotEmpty(t, walletResp.Address)

	// Verify wallet balance
	ethBalance, morBalance, err := client.GetBalance(ctx)
	assert.NoError(t, err)
	assert.True(t, ethBalance.Cmp(big.NewInt(0)) >= 0)
	assert.True(t, morBalance.Cmp(big.NewInt(0)) >= 0)

	// Get available models
	models, err := client.GetAllModels(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, models)

	// Select first available model
	selectedModel := models[0]
	assert.NotEmpty(t, selectedModel.ID)

	// Get model's minimum stake requirement
	minStake, err := client.ModelMinStake(ctx)
	
	fmt.Printf("Error from ModelMinStake: %v\n", err)
	fmt.Printf("Minimum stake required: %v\n", minStake)

	assert.NoError(t, err, "ModelMinStake should not return an error")
	assert.NotNil(t, minStake, "minStake should not be nil")
	assert.True(t, minStake.Cmp(big.NewInt(0)) > 0, "minStake should be greater than zero")

	assert.NoError(t, err)
	assert.True(t, minStake.Cmp(big.NewInt(0)) > 0)

	// Approve allowance for session stake
	approveResp, err := client.ApproveAllowance(ctx, selectedModel.ID, minStake)
	assert.NoError(t, err)
	assert.NotNil(t, approveResp)
	assert.NotEmpty(t, approveResp.TxHash)

	// Verify allowance was set
	allowance, err := client.GetAllowance(ctx, selectedModel.ID)
	assert.NoError(t, err)
	assert.True(t, allowance.Cmp(minStake) >= 0)

	// Open session with model
	sessionReq := &OpenSessionWithDurationRequest{
		SessionDuration: big.NewInt(3600), // 1 hour session
	}
	session, err := client.OpenSession(ctx, sessionReq, selectedModel.ID)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.SessionID)

	// Prepare chat request
	chatReq := &openai.ChatCompletionRequest{
		Model: selectedModel.ID,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Hello! Can you help me test the marketplace integration?",
			},
		},
		Stream: true,
	}

	// Set up channel to collect streamed responses
	responseChan := make(chan *openai.ChatCompletionStreamResponse)
	var responseCallback CompletionCallback = func(resp interface{}) {
		if chatResp, ok := resp.(*openai.ChatCompletionStreamResponse); ok {
			responseChan <- chatResp
		}
	}

	// Start streaming chat completion
	streamErr := make(chan error, 1)
	go func() {
		streamErr <- client.PromptStream(ctx, chatReq, selectedModel.ID, session.SessionID, responseCallback)
	}()

	// Collect and validate responses
	var collectedContent string
	responseTimeout := time.After(10 * time.Second)

	for {
		select {
		case resp := <-responseChan:
			assert.NotNil(t, resp)
			assert.NotEmpty(t, resp.ID)
			assert.Equal(t, selectedModel.ID, resp.Model)
			if len(resp.Choices) > 0 {
				collectedContent += resp.Choices[0].Delta.Content
			}
			if len(resp.Choices) > 0 && resp.Choices[0].FinishReason == "stop" {
				goto StreamComplete
			}
		case err := <-streamErr:
			assert.NoError(t, err)
			goto StreamComplete
		case <-responseTimeout:
			t.Fatal("Timeout waiting for stream response")
		case <-ctx.Done():
			t.Fatal("Context cancelled")
		}
	}

StreamComplete:
	assert.NotEmpty(t, collectedContent)
	fmt.Printf("Received response: %s\n", collectedContent)

	// Close session
	closedSession, err := client.CloseSession(ctx, session.SessionID)
	assert.NoError(t, err)
	assert.NotNil(t, closedSession)

	// Verify session is closed by trying to list it
	userSessions, err := client.ListUserSessions(ctx, walletResp.Address)
	assert.NoError(t, err)

	var foundSession bool
	for _, s := range userSessions {
		if s.ID == session.SessionID {
			assert.NotZero(t, s.ClosedAt)
			foundSession = true
			break
		}
	}
	assert.True(t, foundSession, "Closed session should be found in user's session list")
}
