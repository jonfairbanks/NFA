# Migration Guide: BaseImage to Morpheus Node Integration

## Breaking Changes

### 1. Authentication & Authorization
- New blockchain-based authorization flow required
- Contract approval needed for MOR token spending
- Session management now requires blockchain validation

### 2. Environment Variables
```diff
- MARKETPLACE_URL
- MODEL_ID
+ CHAIN_ID=421614  # Sepolia Arbitrum
+ DIAMOND_CONTRACT=0xb8C55cD613af947E73E262F0d3C54b7211Af16CF
+ MOR_TOKEN_ADDRESS=0x34a285a1b1c166420df5b6630132542923b5b27e
+ WALLET_PRIVATE_KEY=<private_key>
```

### 3. Session Management
- Sessions must be created via `/blockchain/models/{modelId}/session`
- Session validation requires blockchain state checks
- Token balance verification before session creation

## Migration Plan

### 1. Update Configuration Files
```bash
# 1. Update .env
cp example.env .env.new
cat << EOF >> .env.new
CHAIN_ID=421614
DIAMOND_CONTRACT=0xb8C55cD613af947E73E262F0d3C54b7211Af16CF
MOR_TOKEN_ADDRESS=0x34a285a1b1c166420df5b6630132542923b5b27e
EOF

# 2. Update docker-compose.yml
cp docker-compose.yml docker-compose.yml.new
```

### 2. Code Updates Required
```go
// 1. Add blockchain authorization
func (p *Proxy) authorizeContract(amount string) error {
    endpoint := fmt.Sprintf("/blockchain/approve?spender=%s&amount=%s", 
        os.Getenv("DIAMOND_CONTRACT"), amount)
    // Implementation
}

// 2. Update session creation
func (p *Proxy) createSession(modelID string) (*Session, error) {
    endpoint := fmt.Sprintf("/blockchain/models/%s/session", modelID)
    // Implementation
}

// 3. Add balance checking
func (p *Proxy) checkBalance() (*big.Int, error) {
    // Implementation
}
```

### 3. Devlepment Environment Setup Steps
```bash
# 1. Backup current deployment
cp -r . ../BaseImage-backup

# 2. Apply new configurations
mv .env.new .env
mv docker-compose.yml.new docker-compose.yml

# 3. Update proxy code with new blockchain integration

# 4. Test deployment
docker-compose up --build -d
```

### 4. Post-Migration Verification
```bash
# 1. Verify contract authorization
curl -X 'POST' 'http://localhost:8082/blockchain/approve?spender=0xb8C55cD613af947E73E262F0d3C54b7211Af16CF&amount=3'

# 2. Verify model listing
curl -X 'GET' 'http://localhost:8082/blockchain/models'

# 3. Test session creation
curl -X 'POST' 'http://localhost:8082/blockchain/models/{modelId}/session'

# 4. Test chat completion
curl -X 'POST' 'http://localhost:8082/v1/chat/completions' -H 'Content-Type: application/json' -d '{
  "messages": [{"role": "user", "content": "test"}],
  "stream": true
}'
```

The main focus is on integrating blockchain-based authorization and session management while maintaining the existing chat completion endpoint functionality. 