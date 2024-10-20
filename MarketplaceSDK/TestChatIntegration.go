// File: integration_test/integration_test.go

package marketplacesdk

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
    apiBaseURL = "http://localhost:8080" // Ensure your service is running here
    client     *ApiGatewayClient
    createdModelID string
    createdSessionID string
)

func TestMain(m *testing.M) {
    // Initialize the client with the running API base URL
    client = NewApiGatewayClient(apiBaseURL, nil)

    // Run tests
    exitCode := m.Run()

    // Exit with code from test result
    os.Exit(exitCode)
}

// TestCreateAndVerifyModel tests creating a new model and verifying its persistence
func TestCreateAndVerifyModel(t *testing.T) {
    ctx := context.Background()

    // Example parameters for creating a new model
    ipfsID := "QmExampleIpfsHash"           // Replace with a valid IPFS ID
    modelName := "Integration Test AI Model" // Name of the model
    modelFee := big.NewInt(1e18)             // 1 MOR in Wei
    modelStake := big.NewInt(5e18)           // 5 MOR in Wei

    // Define the new model's parameters
    createModelReq := &CreateModelRequest{
        Name:   modelName,
        IpfsID: ipfsID,
        Fee:    modelFee,
        Stake:  modelStake,
        Tags:   []string{"AI", "Integration Test"},
    }

    // Call the client to create a new model
    createdModel, err := client.CreateNewModel(ctx, createModelReq)

    // Assertions to validate the creation
    assert.NoError(t, err, "Expected no error when creating a new model")
    assert.NotNil(t, createdModel, "Model creation should return a valid response")
    assert.Equal(t, createdModel.Name, modelName, "The returned model name should match the input name")
    assert.Equal(t, createdModel.IpfsCID, ipfsID, "The returned IPFS CID should match the input IPFS ID")

    // Capture the created model ID for use in subsequent tests
    createdModelID = createdModel.ID
    assert.NotEmpty(t, createdModelID, "Created Model ID should not be empty")

    // Verify persistence by retrieving all models and checking for the created model
    allModels, err := client.GetAllModels(ctx)
    assert.NoError(t, err, "Expected no error when retrieving all models")
    assert.NotNil(t, allModels, "Models list should not be nil")
    assert.GreaterOrEqual(t, len(allModels), 1, "There should be at least one model available in the system")

    // Find the created model in the list
    var foundModel *Model
    for _, model := range allModels {
        if model.ID == createdModel.ID {
            foundModel = &model
            break
        }
    }
    assert.NotNil(t, foundModel, "Created model should be present in the retrieved models list")
    assert.Equal(t, foundModel.Name, modelName, "Retrieved model name should match the created model name")
    assert.Equal(t, foundModel.IpfsCID, ipfsID, "Retrieved model IPFS CID should match the created model IPFS CID")
}

// TestCreateAndVerifyBid tests creating a new bid and verifying its persistence
func TestCreateAndVerifyBid(t *testing.T) {
    ctx := context.Background()

    // Use the dynamically captured model ID
    modelID := createdModelID
    assert.NotEmpty(t, modelID, "Model ID should be set by TestCreateAndVerifyModel")

    pricePerSecond := big.NewInt(1000000000) // Example price per second

    // Call the client to create a new bid
    createdBid, err := client.CreateNewProviderBid(ctx, modelID, pricePerSecond)

    // Assertions to validate the creation
    assert.NoError(t, err, "Expected no error when creating a new bid")
    assert.NotNil(t, createdBid, "Bid creation should return a valid response")
    assert.Equal(t, createdBid.ModelAgentID, modelID, "The bid's ModelAgentID should match the input model ID")
    assert.Equal(t, createdBid.PricePerSecond, pricePerSecond, "The bid's price per second should match the input value")

    // Verify persistence by retrieving bids for the model agent
    bids, err := client.GetBidsByModelAgent(ctx, modelID, "0", "10")
    assert.NoError(t, err, "Expected no error when retrieving bids by model agent")
    assert.NotNil(t, bids, "Bids list should not be nil")
    assert.GreaterOrEqual(t, len(bids), 1, "There should be at least one bid for the model agent")

    // Find the created bid in the list
    var foundBid *Bid
    for _, bid := range bids {
        if bid.ID == createdBid.ID {
            foundBid = &bid
            break
        }
    }
    assert.NotNil(t, foundBid, "Created bid should be present in the retrieved bids list")
    assert.Equal(t, foundBid.PricePerSecond, pricePerSecond, "Retrieved bid price per second should match the created bid price")
}

// TestOpenAndVerifySession tests opening a session and verifying its persistence
func TestOpenAndVerifySession(t *testing.T) {
    ctx := context.Background()

    // Use the dynamically captured model ID
    modelID := createdModelID
    assert.NotEmpty(t, modelID, "Model ID should be set by TestCreateAndVerifyModel")

    // Define the session's parameters
    openSessionReq := &OpenSessionWithDurationRequest{
        SessionDuration: big.NewInt(3600), // 1-hour session
    }

    // Call the client to open a new session
    openedSession, err := client.OpenSession(ctx, openSessionReq, modelID)

    // Assertions to validate the opening
    assert.NoError(t, err, "Expected no error when opening a session for the model")
    assert.NotNil(t, openedSession, "Session should be successfully opened")
    assert.NotEmpty(t, openedSession.SessionID, "SessionID should not be empty")

    // Capture the opened session ID for use in subsequent tests
    createdSessionID = openedSession.SessionID
    assert.NotEmpty(t, createdSessionID, "Created Session ID should not be empty")

    // Verify persistence by retrieving sessions for the provider
    providerAddress := "provider-address-12345" // Replace with the actual provider address associated with the model
    sessions, err := client.ListProviderSessions(ctx, providerAddress)
    assert.NoError(t, err, "Expected no error when retrieving provider sessions")
    assert.NotNil(t, sessions, "Sessions list should not be nil")
    assert.GreaterOrEqual(t, len(sessions), 1, "There should be at least one session for the provider")

    // Find the opened session in the list
    var foundSession *SessionListItem
    for _, session := range sessions {
        if session.ID == openedSession.SessionID {
            foundSession = &session
            break
        }
    }
    assert.NotNil(t, foundSession, "Opened session should be present in the retrieved sessions list")
    assert.Equal(t, foundSession.ID, openedSession.SessionID, "Retrieved session ID should match the opened session ID")
}

// TestCloseAndVerifySession tests closing a session and verifying its closure
func TestCloseAndVerifySession(t *testing.T) {
    ctx := context.Background()

    // Use the dynamically captured session ID
    sessionID := createdSessionID
    assert.NotEmpty(t, sessionID, "Session ID should be set by TestOpenAndVerifySession")

    // Call the client to close the session and capture both response and error
    closedSession, err := client.CloseSession(ctx, sessionID)

    // Assertions to validate session closure
    assert.NoError(t, err, "Expected no error when closing the session")
    assert.NotNil(t, closedSession, "Session response should not be nil")
    assert.Equal(t, closedSession.SessionID, sessionID, "Closed session ID should match the original session ID")

    // Verify closure by retrieving the session and checking its status
    providerAddress := "provider-address-12345" // Replace with the actual provider address associated with the model
    sessions, err := client.ListProviderSessions(ctx, providerAddress)
    assert.NoError(t, err, "Expected no error when retrieving provider sessions")
    assert.NotNil(t, sessions, "Sessions list should not be nil")

    // Find the closed session in the list and verify its closure
    var foundSession *SessionListItem
    for _, session := range sessions {
        if session.ID == sessionID {
            foundSession = &session
            break
        }
    }
    assert.NotNil(t, foundSession, "Closed session should be present in the retrieved sessions list")
    assert.NotZero(t, foundSession.ClosedAt, "ClosedAt timestamp should be set for the closed session")
    assert.NotEmpty(t, foundSession.CloseoutReceipt, "CloseoutReceipt should be present for the closed session")
}

// TestGetAndVerifyBalance tests retrieving ETH and MOR balances
func TestGetAndVerifyBalance(t *testing.T) {
    ctx := context.Background()

    // Call the client to get ETH and MOR balances
    ethBalance, morBalance, err := client.GetBalance(ctx)

    // Assertions to validate the outcome
    assert.NoError(t, err, "Expected no error when retrieving balances")
    assert.NotNil(t, ethBalance, "ETH balance should not be nil")
    assert.NotNil(t, morBalance, "MOR balance should not be nil")
    // Example assertions based on expected balance ranges
    // Adjust the values according to your test environment
    assert.GreaterOrEqual(t, ethBalance.Int64(), int64(0), "ETH balance should be non-negative")
    assert.GreaterOrEqual(t, morBalance.Int64(), int64(0), "MOR balance should be non-negative")
}

// TestApproveAndVerifyAllowance tests approving a spender and verifying the allowance
func TestApproveAndVerifyAllowance(t *testing.T) {
    ctx := context.Background()

    // Replace with a valid spender address and amount
    spender := "0xSpenderAddress"             // Replace with actual spender address
    amount := big.NewInt(1e18)                // 1 MOR in Wei

    // Call the client to approve allowance
    txResponse, err := client.ApproveAllowance(ctx, spender, amount)

    // Assertions to validate the approval
    assert.NoError(t, err, "Expected no error when approving allowance")
    assert.NotNil(t, txResponse, "Transaction response should not be nil")
    assert.NotEmpty(t, txResponse.TxHash, "Transaction hash should not be empty")

    // Verify the allowance by retrieving it
    allowance, err := client.GetAllowance(ctx, spender)
    assert.NoError(t, err, "Expected no error when retrieving allowance")
    assert.NotNil(t, allowance, "Allowance should not be nil")
    assert.Equal(t, allowance.Cmp(amount), 0, "Allowance should match the approved amount")
}
