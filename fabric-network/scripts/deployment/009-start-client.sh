#!/bin/bash

# Script 009: Start Client Application
# Starts one client process per org: Org1→8080, Org2→8081, Org3→8082

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() { echo -e "${1}${2}${NC}"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

source "${PROJECT_ROOT}/fabric-network/scripts/helpers/fabric-env.sh"

if [ ! -f "${PROJECT_ROOT}/.env" ]; then
    print_status $RED "Error: .env file not found at ${PROJECT_ROOT}/.env"
    exit 1
fi

source "${PROJECT_ROOT}/.env"

DEPLOY_ORG1=${DEPLOY_ORG1:-true}
DEPLOY_ORG2=${DEPLOY_ORG2:-true}
DEPLOY_ORG3=${DEPLOY_ORG3:-false}

print_status $GREEN "=== Starting Fabric Client Application ==="
echo "  Channel:   ${CHANNEL_NAME}"
echo "  Chaincode: ${CHAINCODE_NAME}"
echo "  Org1:      ${DEPLOY_ORG1} → port 8080"
echo "  Org2:      ${DEPLOY_ORG2} → port 8081"
echo "  Org3:      ${DEPLOY_ORG3} → port 8082"
echo ""

# Verify Go
print_status $YELLOW "Verifying prerequisites..."
if [ -d "$HOME/.gvm/gos/go1.25.10/bin" ]; then
    export GOROOT="$HOME/.gvm/gos/go1.25.10"
    export PATH="$HOME/.gvm/gos/go1.25.10/bin:$PATH"
fi
if ! command -v go &> /dev/null; then
    print_status $RED "Error: Go is not installed."
    exit 1
fi
print_status $GREEN "✓ Go version: $(go version | awk '{print $3}')"
print_status $GREEN "✓ Prerequisites verified"

# -------------------------------------------------------------------
# setup_org_crypto <org_domain> <org_msp> <ca_port> <peer_port> <crypto_dir>
# -------------------------------------------------------------------
setup_org_crypto() {
    local org_domain=$1
    local org_msp=$2
    local ca_port=$3
    local peer_port=$4
    local crypto_dir=$5

    mkdir -p "${crypto_dir}/signcerts" "${crypto_dir}/keystore"

    # TLS CA cert
    cp "${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/peer0.${org_domain}/tls/ca.crt" \
       "${crypto_dir}/ca.crt"

    # User cert + key
    local user_msp="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/user1.${org_domain}/msp"

    local cert="${user_msp}/signcerts/cert.pem"
    [ -f "$cert" ] || { print_status $RED "✗ Cert not found: $cert"; exit 1; }
    cp "$cert" "${crypto_dir}/signcerts/cert.pem"

    local key=$(find "${user_msp}/keystore" -name "*_sk" | head -n 1)
    [ -n "$key" ] || { print_status $RED "✗ Key not found in ${user_msp}/keystore"; exit 1; }
    cp "$key" "${crypto_dir}/keystore/priv_sk"

    [ -f "${user_msp}/config.yaml" ] && cp "${user_msp}/config.yaml" "${crypto_dir}/config.yaml"

    # Connection profile
    local org_short="${org_msp%MSP}"  # Org1MSP → Org1
    cat > "${crypto_dir}/connection-profile.yaml" << EOF
name: "fabric-network-${org_short}"
version: "1.0.0"
client:
  organization: ${org_short}
  connection:
    timeout:
      peer:
        endorser: '300'
organizations:
  ${org_short}:
    mspid: ${org_msp}
    peers:
    - peer0.${org_domain}
    certificateAuthorities:
    - ca.${org_domain}
peers:
  peer0.${org_domain}:
    url: grpcs://peer0.${org_domain}:${peer_port}
    tlsCACerts:
      pem: |
$(cat "${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/peer0.${org_domain}/tls/ca.crt" | sed 's/^/        /')
    grpcOptions:
      ssl-target-name-override: peer0.${org_domain}
certificateAuthorities:
  ca.${org_domain}:
    url: https://localhost:${ca_port}
    caName: ca.${org_domain}
    tlsCACerts:
      pem: |
$(cat "${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/ca/msp/cacerts/localhost-${ca_port}.pem" | sed 's/^/        /')
    httpOptions:
      verify: false
EOF

    print_status $GREEN "✓ Crypto + connection profile ready for ${org_msp} (port ${peer_port})"
}

# -------------------------------------------------------------------
# start_org_client <org_domain> <org_msp> <ca_name> <ca_port> <server_port> <bootstrap_admin_dir>
# -------------------------------------------------------------------
start_org_client() {
    local org_domain=$1
    local org_msp=$2
    local ca_name=$3
    local ca_port=$4
    local server_port=$5
    local bootstrap_admin_dir=$6

    local org_short="${org_msp%MSP}"
    local crypto_dir="${PROJECT_ROOT}/fabric-network/client/crypto-${org_short}"
    local wallet_dir="${PROJECT_ROOT}/fabric-network/client/wallet-${org_short}"
    local log_file="/tmp/fabric-client-${org_short}.log"

    print_status $YELLOW "Setting up ${org_msp} client (port ${server_port})..."

    local peer_port
    case "$org_msp" in
        Org1MSP) peer_port="${PEER0_ORG1_PORT}" ;;
        Org2MSP) peer_port="${PEER0_ORG2_PORT}" ;;
        Org3MSP) peer_port="${PEER0_ORG3_PORT}" ;;
    esac

    setup_org_crypto "${org_domain}" "${org_msp}" "${ca_port}" "${peer_port}" "${crypto_dir}"

    mkdir -p "$wallet_dir"

    WALLET_PATH="${wallet_dir}" \
    CHANNEL_ID="${CHANNEL_NAME}" \
    CHAINCODE_ID="${CHAINCODE_NAME}" \
    SERVER_PORT="${server_port}" \
    GIN_MODE="${CLIENT_MODE}" \
    TLS_CERT_PATH="${crypto_dir}" \
    CONNECTION_PROFILE="${crypto_dir}/connection-profile.yaml" \
    JWT_SECRET="${JWT_SECRET:-hlf-demo-jwt-secret-change-this-in-production}" \
    CA_URL="http://localhost:${ca_port}" \
    CA_NAME="${ca_name}" \
    MSP_ID="${org_msp}" \
    CA_ADMIN_MSP_DIR="${bootstrap_admin_dir}" \
    GODEBUG=netdns=cgo \
    nohup "${PROJECT_ROOT}/fabric-network/client/fabric-client" > "${log_file}" 2>&1 &

    local pid=$!
    sleep 3

    if ! ps -p $pid > /dev/null 2>&1; then
        print_status $RED "✗ ${org_msp} client failed to start — check ${log_file}"
        exit 1
    fi

    # Wait for health
    local attempt=1
    while [ $attempt -le 30 ]; do
        if curl -s "http://localhost:${server_port}/health" > /dev/null 2>&1; then
            print_status $GREEN "✓ ${org_msp} client ready on port ${server_port} (PID: ${pid})"
            return 0
        fi
        sleep 2
        attempt=$((attempt + 1))
    done

    print_status $RED "✗ ${org_msp} client timed out — check ${log_file}"
    exit 1
}

# -------------------------------------------------------------------
# Build binary once
# -------------------------------------------------------------------
print_status $YELLOW "Building client application..."
cd "${PROJECT_ROOT}/fabric-network/client"

print_status $YELLOW "Downloading Go dependencies..."
go mod download 2>&1 || go mod download

print_status $YELLOW "Compiling..."
go build -o fabric-client main.go

print_status $GREEN "✓ Client binary built"

# Stop any existing instances
if pgrep -f "fabric-client" > /dev/null; then
    print_status $YELLOW "Stopping existing client instances..."
    pkill -f "fabric-client" || true
    sleep 2
fi

# -------------------------------------------------------------------
# Start one client per deployed org
# -------------------------------------------------------------------
if [ "$DEPLOY_ORG1" = true ]; then
    start_org_client \
        "${ORG1_DOMAIN}" \
        "${ORG1_NAME}" \
        "${CA_ORG1_NAME}" \
        "${CA_ORG1_PORT}" \
        "8080" \
        "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/bootstrap-admin.${ORG1_DOMAIN}/msp"
fi

if [ "$DEPLOY_ORG2" = true ]; then
    start_org_client \
        "${ORG2_DOMAIN}" \
        "${ORG2_NAME}" \
        "${CA_ORG2_NAME}" \
        "${CA_ORG2_PORT}" \
        "8081" \
        "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/users/bootstrap-admin.${ORG2_DOMAIN}/msp"
fi

if [ "$DEPLOY_ORG3" = true ]; then
    start_org_client \
        "${ORG3_DOMAIN}" \
        "${ORG3_NAME}" \
        "${CA_ORG3_NAME}" \
        "${CA_ORG3_PORT}" \
        "8082" \
        "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG3_DOMAIN}/users/bootstrap-admin.${ORG3_DOMAIN}/msp"
fi

print_status $GREEN "=== Client Application Started Successfully ==="
echo ""
[ "$DEPLOY_ORG1" = true ] && echo "  Org1 (Manufacturer): http://localhost:8080  logs: /tmp/fabric-client-Org1.log"
[ "$DEPLOY_ORG2" = true ] && echo "  Org2 (Distributor):  http://localhost:8081  logs: /tmp/fabric-client-Org2.log"
[ "$DEPLOY_ORG3" = true ] && echo "  Org3 (Retailer):     http://localhost:8082  logs: /tmp/fabric-client-Org3.log"
echo ""
print_status $YELLOW "Stop all: pkill -f fabric-client"
