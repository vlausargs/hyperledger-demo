#!/bin/bash

# Script 009: Start Client Application
# This script builds and starts the Go client application with Gin framework

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Load environment variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

if [ ! -f "${PROJECT_ROOT}/.env" ]; then
    print_status $RED "Error: .env file not found at ${PROJECT_ROOT}/.env"
    exit 1
fi

source "${PROJECT_ROOT}/.env"

print_status $GREEN "=== Starting Fabric Client Application ==="
print_status $YELLOW "Client Configuration:"
echo "  Channel ID: ${CHANNEL_NAME}"
echo "  Chaincode ID: ${CHAINCODE_NAME}"
echo "  Server Port: ${CLIENT_PORT}"
echo "  Mode: ${CLIENT_MODE}"
echo ""

# Verify prerequisites
print_status $YELLOW "Verifying prerequisites..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_status $RED "Error: Go is not installed. Please install Go 1.19 or higher."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_status $GREEN "✓ Go version: $GO_VERSION"

# Check if chaincode is deployed
if [ "$DEPLOY_ORG1" = true ]; then
    if ! docker ps | grep -q "peer0.${ORG1_DOMAIN}"; then
        print_status $RED "Error: Org1 peer is not running. Chaincode might not be deployed."
        exit 1
    fi
elif [ "$DEPLOY_ORG2" = true ]; then
    if ! docker ps | grep -q "peer0.${ORG2_DOMAIN}"; then
        print_status $RED "Error: Org2 peer is not running. Chaincode might not be deployed."
        exit 1
    fi
fi

print_status $GREEN "✓ Prerequisites verified"

# Function to setup crypto files for client
setup_crypto() {
    print_status $YELLOW "Setting up crypto files for client..."

    # Check if fabric-ca-client is available
    if ! command -v fabric-ca-client &> /dev/null; then
        print_status $RED "Error: fabric-ca-client not found. Please install Fabric CA client."
        exit 1
    fi

    # Create crypto directory structure
    local client_crypto_dir="${PROJECT_ROOT}/fabric-network/client/crypto"
    mkdir -p "${client_crypto_dir}/signcerts"
    mkdir -p "${client_crypto_dir}/keystore"

    # Copy TLS CA certificate (from peer's TLS CA)
    if [ "$DEPLOY_ORG1" = true ]; then
        # Copy peer TLS CA certificate
        cp "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt" \
           "${client_crypto_dir}/ca.crt"
        print_status $GREEN "✓ Copied TLS CA certificate"

        # Copy user certificate and private key from user1 (client identity)
        local user1_msp_dir="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/user1.${ORG1_DOMAIN}/msp"
        if [ -f "${user1_msp_dir}/signcerts/cert.pem" ]; then
            cp "${user1_msp_dir}/signcerts/cert.pem" "${client_crypto_dir}/signcerts/cert.pem"
            print_status $GREEN "✓ Copied ${ORG1_NAME} user certificate (user1)"
        else
            print_status $RED "✗ User certificate not found at ${user1_msp_dir}/signcerts/cert.pem"
            exit 1
        fi

        # Copy private key
        if [ -d "${user1_msp_dir}/keystore" ]; then
            local key_file=$(find "${user1_msp_dir}/keystore" -name "*_sk" | head -n 1)
            if [ -n "$key_file" ]; then
                cp "$key_file" "${client_crypto_dir}/keystore/priv_sk"
                print_status $GREEN "✓ Copied ${ORG1_NAME} user private key (user1)"
            else
                print_status $RED "✗ Private key not found in ${user1_msp_dir}/keystore"
                exit 1
            fi
        else
            print_status $RED "✗ Keystore directory not found at ${user1_msp_dir}/keystore"
            exit 1
        fi

        # Copy MSP config file
        if [ -f "${user1_msp_dir}/config.yaml" ]; then
            cp "${user1_msp_dir}/config.yaml" "${client_crypto_dir}/config.yaml"
            print_status $GREEN "✓ Copied ${ORG1_NAME} MSP config (user1)"
        else
            print_status $RED "✗ MSP config not found at ${user1_msp_dir}/config.yaml"
            exit 1
        fi

        # Create connection profile (simplified version)
        cat > "${client_crypto_dir}/connection-profile.yaml" << EOF
name: "test-network"
version: "1.0.0"
client:
  organization: Org1
  connection:
    timeout:
      peer:
        endorser: '300'
organizations:
  Org1:
    mspid: ${ORG1_NAME}
    peers:
    - peer0.${ORG1_DOMAIN}
    certificateAuthorities:
    - ca.${ORG1_DOMAIN}
peers:
  peer0.${ORG1_DOMAIN}:
    url: grpcs://peer0.${ORG1_DOMAIN}:${PEER0_ORG1_PORT}
    tlsCACerts:
      pem: |
$(cat "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt" | sed 's/^/        /')
    grpcOptions:
      ssl-target-name-override: peer0.${ORG1_DOMAIN}
certificateAuthorities:
  ca.${ORG1_DOMAIN}:
    url: https://localhost:${CA_ORG1_PORT}
    caName: ca.${ORG1_DOMAIN}
    tlsCACerts:
      pem: |
$(cat "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/ca/msp/cacerts/localhost-${CA_ORG1_PORT}.pem" | sed 's/^/        /')
    httpOptions:
      verify: false
EOF
    fi

    print_status $GREEN "✓ Connection profile created"
}

# Setup crypto files
setup_crypto

# Build the client application
print_status $YELLOW "Building client application..."

cd "${PROJECT_ROOT}/fabric-network/client"

# Download dependencies
print_status $YELLOW "Downloading Go dependencies..."
go mod download 2>&1 || {
    print_status $RED "Error: Failed to download Go dependencies"
    print_status $YELLOW "Attempting to fix go.mod file..."

    # Try to fix go.mod file if needed
    sed -i '/^```/d' go.mod 2>/dev/null || true

    # Try downloading again
    go mod download 2>&1 || {
        print_status $RED "Error: Still failed to download dependencies"
        print_status $YELLOW "Please check your network connection and try again"
        exit 1
    }
}

print_status $GREEN "✓ Dependencies downloaded"

# Build the application
print_status $YELLOW "Compiling Go application..."
go build -o fabric-client main.go 2>&1 || {
    print_status $RED "Error: Failed to build client application"
    print_status $YELLOW "Checking build errors..."
    go build -o fabric-client main.go
}

if [ ! -f "fabric-client" ]; then
    print_status $RED "Error: Binary file not found after build"
    exit 1
fi

print_status $GREEN "✓ Client application built successfully"

# Set environment variables for the client
export WALLET_PATH="${PROJECT_ROOT}/fabric-network/client/wallet"
export CHANNEL_ID="${CHANNEL_NAME}"
export CHAINCODE_ID="${CHAINCODE_NAME}"
export SERVER_PORT="${CLIENT_PORT}"
export GIN_MODE="${CLIENT_MODE}"
export TLS_CERT_PATH="${PROJECT_ROOT}/fabric-network/client/crypto"
export CONNECTION_PROFILE="${PROJECT_ROOT}/fabric-network/client/crypto/connection-profile.yaml"

# Create wallet directory
mkdir -p "$WALLET_PATH"

# Start the client application
print_status $YELLOW "Starting client application..."

# Check if already running
if pgrep -f "fabric-client" > /dev/null; then
    print_status $YELLOW "Client application is already running. Stopping it..."
    pkill -f "fabric-client"
    sleep 2
fi

# Start the application in background
nohup ./fabric-client > /tmp/fabric-client.log 2>&1 &
CLIENT_PID=$!

# Wait for application to start
sleep 3

# Check if application is running
if ps -p $CLIENT_PID > /dev/null; then
    print_status $GREEN "✓ Client application started successfully (PID: $CLIENT_PID)"
else
    print_status $RED "✗ Failed to start client application"
    print_status $YELLOW "Check logs: cat /tmp/fabric-client.log"
    exit 1
fi

# Wait for server to be ready
print_status $YELLOW "Waiting for server to be ready..."

MAX_ATTEMPTS=30
ATTEMPT=1

while [ $ATTEMPT -le $MAX_ATTEMPTS ]; do
    if curl -s http://localhost:${CLIENT_PORT}/health > /dev/null 2>&1; then
        print_status $GREEN "✓ Server is ready and listening on port ${CLIENT_PORT}"
        break
    fi

    echo "  Attempt $ATTEMPT/$MAX_ATTEMPTS: Waiting for server..."
    sleep 2
    ATTEMPT=$((ATTEMPT + 1))
done

if [ $ATTEMPT -gt $MAX_ATTEMPTS ]; then
    print_status $RED "✗ Server failed to start within timeout period"
    print_status $YELLOW "Check logs: cat /tmp/fabric-client.log"
    exit 1
fi

# Test health endpoint
print_status $YELLOW "Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s http://localhost:${CLIENT_PORT}/health)
echo "  Response: $HEALTH_RESPONSE"

if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    print_status $GREEN "✓ Health check passed"
else
    print_status $YELLOW "Health check response unexpected"
fi

# Display API endpoints
print_status $YELLOW "=== API Endpoints ==="
echo ""
echo "Base URL: http://localhost:${CLIENT_PORT}/api/v1"
echo ""
echo "Health Check:"
echo "  GET  /health"
echo ""
echo "Assets:"
echo "  GET    /assets                    - Get all assets"
echo "  GET    /assets/:id                - Get specific asset"
echo "  POST   /assets                    - Create new asset"
echo "  PUT    /assets/:id                - Update asset"
echo "  DELETE /assets/:id                - Delete asset"
echo "  POST   /assets/:id/transfer       - Transfer asset ownership"
echo "  GET    /assets/:id/history        - Get asset history"
echo "  GET    /assets/range?start=X&end=Y - Get assets by range"
echo ""
echo "Channels:"
echo "  GET /channels            - Get all channels"
echo "  GET /channels/:channelId - Get channel info"
echo ""
echo "Chaincodes:"
echo "  GET /chaincodes             - Get all chaincodes"
echo "  GET /chaincodes/:chaincodeId - Get chaincode info"
echo ""
echo "Network:"
echo "  GET /network/peers          - Get network peers"
echo "  GET /network/organizations  - Get network organizations"
echo ""
echo "Transactions:"
echo "  GET /transactions        - Get all transactions"
echo "  GET /transactions/:txId - Get specific transaction"
echo ""

# Display useful commands
print_status $YELLOW "=== Useful Commands ==="
echo ""
echo "View logs:"
echo "  tail -f /tmp/fabric-client.log"
echo ""
echo "Test health endpoint:"
echo "  curl http://localhost:${CLIENT_PORT}/health"
echo ""
echo "Get all assets:"
echo "  curl http://localhost:${CLIENT_PORT}/api/v1/assets"
echo ""
echo "Create asset:"
echo '  curl -X POST http://localhost:'${CLIENT_PORT}'/api/v1/assets \'
echo '    -H "Content-Type: application/json" \'
echo '    -d '"'"'{"ID":"test1","color":"red","size":10,"owner":"Alice","appraisedValue":100}'"'"''
echo ""
echo "Get specific asset:"
echo "  curl http://localhost:${CLIENT_PORT}/api/v1/assets/test1"
echo ""
echo "Update asset:"
echo '  curl -X PUT http://localhost:'${CLIENT_PORT}'/api/v1/assets/test1 \'
echo '    -H "Content-Type: application/json" \'
echo '    -d '"'"'{"color":"blue","size":15,"owner":"Bob","appraisedValue":150}'"'"''
echo ""
echo "Transfer asset:"
echo '  curl -X POST http://localhost:'${CLIENT_PORT}'/api/v1/assets/test1/transfer \'
echo '    -H "Content-Type: application/json" \'
echo '    -d '"'"'{"newOwner":"Charlie"}'"'"''
echo ""
echo "Stop client application:"
echo "  pkill -f fabric-client"
echo ""

# Create a quick start script
cat > "${PROJECT_ROOT}/fabric-network/client/start.sh" << 'EOF'
#!/bin/bash
# Quick start script for Fabric Client

cd "$(dirname "$0")"

# Set environment variables
export WALLET_PATH="./wallet"
export CHANNEL_ID="mychannel"
export CHAINCODE_ID="basic"
export SERVER_PORT="8080"
export GIN_MODE="debug"
export TLS_CERT_PATH="./crypto"
export CONNECTION_PROFILE="./crypto/connection-profile.yaml"

# Start application
./fabric-client
EOF

chmod +x "${PROJECT_ROOT}/fabric-network/client/start.sh"

# Create a test script
cat > "${PROJECT_ROOT}/fabric-network/client/test-api.sh" << 'EOF'
#!/bin/bash
# Test script for Fabric Client API

BASE_URL="http://localhost:8080/api/v1"

echo "=== Testing Fabric Client API ==="
echo ""

# Health check
echo "1. Health Check:"
curl -s "$BASE_URL/../health" | jq '.'
echo ""

# Get all assets
echo "2. Get All Assets:"
curl -s "$BASE_URL/assets" | jq '.'
echo ""

# Create a new asset
echo "3. Create New Asset:"
curl -X POST "$BASE_URL/assets" \
  -H "Content-Type: application/json" \
  -d '{"ID":"api_test_1","color":"purple","size":20,"owner":"API_User","appraisedValue":500}' | jq '.'
echo ""

# Get specific asset
echo "4. Get Specific Asset:"
curl -s "$BASE_URL/assets/api_test_1" | jq '.'
echo ""

# Update asset
echo "5. Update Asset:"
curl -X PUT "$BASE_URL/assets/api_test_1" \
  -H "Content-Type: application/json" \
  -d '{"color":"orange","size":25,"owner":"API_User2","appraisedValue":600}' | jq '.'
echo ""

# Transfer asset
echo "6. Transfer Asset:"
curl -X POST "$BASE_URL/assets/api_test_1/transfer" \
  -H "Content-Type: application/json" \
  -d '{"newOwner":"API_User3"}' | jq '.'
echo ""

# Get asset history
echo "7. Get Asset History:"
curl -s "$BASE_URL/assets/api_test_1/history" | jq '.'
echo ""

echo "=== API Testing Complete ==="
EOF

chmod +x "${PROJECT_ROOT}/fabric-network/client/test-api.sh"

print_status $GREEN "=== Client Application Started Successfully ==="
print_status $YELLOW "Quick start scripts created:"
echo "  ${PROJECT_ROOT}/fabric-network/client/start.sh - Start the client application"
echo "  ${PROJECT_ROOT}/fabric-network/client/test-api.sh - Test the API endpoints"
echo ""
print_status $GREEN "Your Hyperledger Fabric network is now ready!"
print_status $YELLOW "Next steps:"
echo "  1. Test the API using the provided test script"
echo "  2. Build your custom client application"
echo "  3. Deploy your own chaincode"
echo "  4. Integrate with your existing systems"
echo ""
print_status $GREEN "Happy Blockchain Development!"
