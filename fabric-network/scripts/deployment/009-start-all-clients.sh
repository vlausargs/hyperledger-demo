#!/bin/bash

# Script 009b: Start Client Application for All Orgs (Org1, Org2, Org3)
# Org1 → port 8080, Org2 → port 8081, Org3 → port 8082

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
    print_status $RED "Error: .env not found at ${PROJECT_ROOT}/.env"
    exit 1
fi
source "${PROJECT_ROOT}/.env"

CLIENT_DIR="${PROJECT_ROOT}/fabric-network/client"

# Prefer gvm go1.25.10 if available
if [ -d "$HOME/.gvm/gos/go1.25.10/bin" ]; then
    export GOROOT="$HOME/.gvm/gos/go1.25.10"
    export PATH="$HOME/.gvm/gos/go1.25.10/bin:$PATH"
fi

# ─── Build binary (once) ─────────────────────────────────────────────────────

build_client() {
    print_status $YELLOW "Building fabric-client binary..."
    cd "$CLIENT_DIR"
    go mod download 2>&1 || { print_status $RED "go mod download failed"; exit 1; }
    go build -o fabric-client main.go 2>&1 || { print_status $RED "go build failed"; exit 1; }
    print_status $GREEN "✓ Binary built"
}

# ─── Setup crypto for one org ─────────────────────────────────────────────────

setup_crypto_for_org() {
    local org_num=$1      # 1, 2, 3
    local org_domain=$2   # org1.example.com
    local org_msp=$3      # Org1MSP
    local peer_port=$4    # 8051
    local ca_port=$5      # 8054
    local ca_name=$6      # ca-org1

    local crypto_dir="${CLIENT_DIR}/crypto-org${org_num}"
    local orgs_dir="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}"

    mkdir -p "${crypto_dir}/signcerts" "${crypto_dir}/keystore"

    # TLS CA cert (bundled all-org ca.crt for the peer)
    cp "${orgs_dir}/peers/peer0.${org_domain}/tls/ca.crt" "${crypto_dir}/ca.crt"

    # User identity
    local user_msp="${orgs_dir}/users/user1.${org_domain}/msp"
    cp "${user_msp}/signcerts/cert.pem"                      "${crypto_dir}/signcerts/cert.pem"
    cp "$(find ${user_msp}/keystore -name '*_sk' | head -1)" "${crypto_dir}/keystore/priv_sk"
    cp "${user_msp}/config.yaml"                             "${crypto_dir}/config.yaml"

    # Connection profile
    local tls_pem
    tls_pem=$(cat "${orgs_dir}/peers/peer0.${org_domain}/tls/ca.crt" | sed 's/^/        /')
    local ca_pem
    ca_pem=$(cat "${orgs_dir}/ca/msp/cacerts/localhost-${ca_port}.pem" | sed 's/^/        /')

    cat > "${crypto_dir}/connection-profile.yaml" << EOF
name: "supply-chain-org${org_num}"
version: "1.0.0"
client:
  organization: Org${org_num}
  connection:
    timeout:
      peer:
        endorser: '300'
organizations:
  Org${org_num}:
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
${tls_pem}
    grpcOptions:
      ssl-target-name-override: peer0.${org_domain}
certificateAuthorities:
  ca.${org_domain}:
    url: https://localhost:${ca_port}
    caName: ${ca_name}
    tlsCACerts:
      pem: |
${ca_pem}
    httpOptions:
      verify: false
EOF

    print_status $GREEN "✓ Crypto set up for Org${org_num} (${org_msp})"
}

# ─── Start one client process ─────────────────────────────────────────────────

start_client_for_org() {
    local org_num=$1
    local org_domain=$2
    local org_msp=$3
    local peer_port=$4
    local ca_port=$5
    local ca_name=$6
    local server_port=$7

    local crypto_dir="${CLIENT_DIR}/crypto-org${org_num}"
    local wallet_dir="${CLIENT_DIR}/wallet-org${org_num}"
    local log_file="/tmp/fabric-client-org${org_num}.log"
    local admin_msp_dir="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/bootstrap-admin.${org_domain}/msp"

    mkdir -p "$wallet_dir"

    # Kill existing instance on this port
    local existing_pid
    existing_pid=$(pgrep -f "fabric-client.*-org${org_num}" 2>/dev/null || true)
    if [ -n "$existing_pid" ]; then
        kill "$existing_pid" 2>/dev/null || true
        sleep 1
    fi
    # Also kill by port
    fuser -k "${server_port}/tcp" 2>/dev/null || true
    sleep 1

    print_status $YELLOW "Starting Org${org_num} client on port ${server_port}..."

    GODEBUG=netdns=cgo \
    WALLET_PATH="$wallet_dir" \
    CHANNEL_ID="${CHANNEL_NAME}" \
    CHAINCODE_ID="${CHAINCODE_NAME}" \
    SERVER_PORT="${server_port}" \
    GIN_MODE="${CLIENT_MODE}" \
    TLS_CERT_PATH="$crypto_dir" \
    CONNECTION_PROFILE="${crypto_dir}/connection-profile.yaml" \
    JWT_SECRET="${JWT_SECRET:-hlf-demo-jwt-secret-change-this-in-production}" \
    CA_URL="http://localhost:${ca_port}" \
    CA_NAME="${ca_name}" \
    MSP_ID="${org_msp}" \
    CA_ADMIN_MSP_DIR="$admin_msp_dir" \
    nohup "${CLIENT_DIR}/fabric-client" > "$log_file" 2>&1 &

    local pid=$!
    # Rename process for easier identification (best-effort)
    disown $pid 2>/dev/null || true

    # Wait for ready
    local max=30
    local i=1
    while [ $i -le $max ]; do
        if curl -s "http://localhost:${server_port}/health" > /dev/null 2>&1; then
            print_status $GREEN "✓ Org${org_num} client ready on :${server_port} (PID $pid)"
            return 0
        fi
        sleep 2
        i=$((i+1))
    done

    print_status $RED "✗ Org${org_num} client failed to start — check $log_file"
    return 1
}

# ─── Main ─────────────────────────────────────────────────────────────────────

print_status $GREEN "=== Starting Supply Chain Clients (All Orgs) ==="

# Build once
build_client

# Setup crypto for each org
print_status $YELLOW "Setting up crypto materials..."
setup_crypto_for_org 1 "${ORG1_DOMAIN}" "${ORG1_NAME}" "${PEER0_ORG1_PORT}" "${CA_ORG1_PORT}" "${CA_ORG1_NAME}"
setup_crypto_for_org 2 "${ORG2_DOMAIN}" "${ORG2_NAME}" "${PEER0_ORG2_PORT}" "${CA_ORG2_PORT}" "${CA_ORG2_NAME}"
setup_crypto_for_org 3 "${ORG3_DOMAIN}" "${ORG3_NAME}" "${PEER0_ORG3_PORT}" "${CA_ORG3_PORT}" "${CA_ORG3_NAME}"

# Start clients
start_client_for_org 1 "${ORG1_DOMAIN}" "${ORG1_NAME}" "${PEER0_ORG1_PORT}" "${CA_ORG1_PORT}" "${CA_ORG1_NAME}" "8080"
start_client_for_org 2 "${ORG2_DOMAIN}" "${ORG2_NAME}" "${PEER0_ORG2_PORT}" "${CA_ORG2_PORT}" "${CA_ORG2_NAME}" "8081"
start_client_for_org 3 "${ORG3_DOMAIN}" "${ORG3_NAME}" "${PEER0_ORG3_PORT}" "${CA_ORG3_PORT}" "${CA_ORG3_NAME}" "8082"

print_status $GREEN ""
print_status $GREEN "=== All Clients Running ==="
echo ""
echo "  Org1 (Manufacturer) → http://localhost:8080  log: /tmp/fabric-client-org1.log"
echo "  Org2 (Distributor)  → http://localhost:8081  log: /tmp/fabric-client-org2.log"
echo "  Org3 (Retailer)     → http://localhost:8082  log: /tmp/fabric-client-org3.log"
echo ""
echo "  Health checks:"
echo "    curl http://localhost:8080/health"
echo "    curl http://localhost:8081/health"
echo "    curl http://localhost:8082/health"
echo ""
echo "  Stop all clients:"
echo "    pkill -f fabric-client"
echo ""
print_status $YELLOW "Login credentials: username=admin  password=asdqwe123"
echo ""
print_status $YELLOW "Org2 accepts custody transfers initiated by Org1."
print_status $YELLOW "Org3 accepts custody transfers initiated by Org2."
