package marketplacesdk

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sashabaranov/go-openai"
)

// WalletResponse represents a response containing the wallet address information.
type WalletResponse struct {
    Address string `json:"address"`
}

// SessionRequest represents a request to start a new session.
type SessionRequest struct {
    ModelID         string   `json:"modelId"`
    SessionDuration *big.Int `json:"sessionDuration"`
}

// SessionStakeRequest represents a request to stake for a session.
type SessionStakeRequest struct {
    Approval    string   `json:"approval"`
    ApprovalSig string   `json:"approvalSig"`
    Stake       *big.Int `json:"stake"`
}

// CloseSessionRequest represents a request to close an ongoing session.
type CloseSessionRequest struct {
    SessionID string `json:"id"`
}

// WalletRequest represents a request to create or set up a wallet.
type WalletRequest struct {
    PrivateKey string `json:"privateKey"`
}

// ProdiaGenerationRequest represents a request to generate data from the Prodia model.
type ProdiaGenerationRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    ApiURL string `json:"apiUrl"`
    ApiKey string `json:"apiKey"`
}

// OpenSessionRequest represents a request to open a session.
type OpenSessionRequest struct {
    Approval    string   `json:"approval"`
    ApprovalSig string   `json:"approvalSig"`
    Stake       *big.Int `json:"stake"`
}

// SendRequest represents a request to send a specific amount to a recipient.
type SendRequest struct {
    To     common.Address `json:"to"`
    Amount *big.Int       `json:"amount"`
}

// OpenSessionWithDurationRequest represents a request to open a session with a specific duration.
type OpenSessionWithDurationRequest struct {
    SessionDuration *big.Int `json:"sessionDuration"`
}

// CreateBidRequest represents a request to create a bid with a model ID and price per second.
type CreateBidRequest struct {
    ModelID        string   `json:"modelID"`
    PricePerSecond *big.Int `json:"pricePerSecond"`
}

// CreateProviderRequest represents a request to register a provider.
type CreateProviderRequest struct {
    Stake    *big.Int `json:"stake"`
    Endpoint string   `json:"endpoint"`
}

// CreateModelRequest represents a request to create a model.
type CreateModelRequest struct {
    Name   string   `json:"name"`
    IpfsCID string   `json:"ipfsCID"`
    Fee    *big.Int `json:"fee"`
    Stake  *big.Int `json:"stake"`
    Tags   []string `json:"tags"`
}

// Session represents a session in the system.
type Session struct {
    SessionID string `json:"sessionID"`
    // Add other necessary fields
}

// SessionListItem represents an item in the session list.
type SessionListItem struct {
    ID              string `json:"id"`
    ClosedAt        int64  `json:"closedAt"`
    CloseoutReceipt string `json:"closeoutReceipt"`
    // Add other necessary fields
}

// StatusResponse represents a generic status response.
type StatusResponse struct {
    Status string `json:"status"`
}

// BalanceResponse represents a response containing balance information.
type BalanceResponse struct {
    Balance *big.Int `json:"balance"`
}

// BidResponse represents a response containing bid information.
type BidResponse struct {
    Bid *Bid `json:"bid"`
}

// SessionResponse represents a response containing session information.
type SessionResponse struct {
    SessionID string `json:"sessionID"`
}

// ProviderResponse represents a response containing provider information.
type ProviderResponse struct {
    Provider *Provider `json:"provider"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
    Error string `json:"error"`
}

// TransactionResponse represents a response containing a transaction hash.
type TransactionResponse struct {
    TxHash string `json:"tx"`
}

// Bid represents a bid in the system.
type Bid struct {
    ID             string   `json:"id"`
    Provider       string   `json:"provider"`
    ModelAgentID   string   `json:"modelAgentId"`
    PricePerSecond *big.Int `json:"pricePerSecond"`
}

// Provider represents a provider in the system.
type Provider struct {
    Address  string   `json:"address"`
    Endpoint string   `json:"endpoint"`
    Stake    *big.Int `json:"stake"`
}

// Model represents a model in the system.
type Model struct {
    ID     string   `json:"id"`
    Name   string   `json:"name"`
    IpfsCID string  `json:"ipfsCID"`
    Fee    *big.Int `json:"fee"`
    Stake  *big.Int `json:"stake"`
    Tags   []string `json:"tags"`
}

// ApproveResponse represents the response from an approval request.
type ApproveResponse struct {
    Success bool `json:"success"`
}

// ApproveRequest represents the request payload for approving an allowance.
type ApproveRequest struct {
    Spender string `json:"spender"`
    Amount  string `json:"amount"`
}

// AllowanceResponse represents the response when querying an allowance.
type AllowanceResponse struct {
    Allowance string `json:"allowance"`
}

// ChatRequest represents the request payload for initiating a chat completion.
type ChatRequest struct {
    Model    string `json:"model"`
    Messages []struct {
        Role    string `json:"role"`
        Content string `json:"content"`
    } `json:"messages"`
    Stream bool `json:"stream"`
}

// ModelStats represents the statistics of a model.
type ModelStats struct {
    TotalSessions uint64   `json:"totalSessions"`
    TotalStake    *big.Int `json:"totalStake"`
}

// ApiGatewayClientInterface defines the methods that ApiGatewayClient and MockApiGatewayClient must implement.
type ApiGatewayClientInterface interface {
    GetAllowance(ctx context.Context, spender string) (*big.Int, error)
    ApproveAllowance(ctx context.Context, spender string, amount *big.Int) (*TransactionResponse, error)
    OpenStakeSession(ctx context.Context, request *SessionStakeRequest) (*Session, error)
    PromptStream(ctx context.Context, request *openai.ChatCompletionRequest, modelID string, sessionID string, callback CompletionCallback) error
}