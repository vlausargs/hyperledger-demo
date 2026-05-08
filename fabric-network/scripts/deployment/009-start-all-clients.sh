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
# Args: org_domain  org_msp  peer_port  ca_port  ca_name

setup_crypto_for_org() {
    local org_domain=$1   # org1.example.com
    local org_msp=$2      # Org1MSP
    local peer_port=$3    # 8051
    local ca_port=$4      # 8054
    local ca_name=$5      # ca-org1

    local crypto_dir="${CLIENT_DIR}/crypto-${org_msp}"
    local orgs_dir="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}"

    mkdir -p "${crypto_dir}/signcerts" "${crypto_dir}/keystore"

    # TLS CA cert
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
name: "supply-chain-${org_msp}"
version: "1.0.0"
client:
  organization: ${org_msp}
  connection:
    timeout:
      peer:
        endorser: '300'
organizations:
  ${org_msp}:
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

    print_status $GREEN "✓ Crypto set up for ${org_msp}"
}

# ─── Start one client process ─────────────────────────────────────────────────
# Args: org_domain  org_msp  peer_port  ca_port  ca_name  server_port

start_client_for_org() {
    local org_domain=$1
    local org_msp=$2
    local peer_port=$3
    local ca_port=$4
    local ca_name=$5
    local server_port=$6

    local crypto_dir="${CLIENT_DIR}/crypto-${org_msp}"
    local wallet_dir="${CLIENT_DIR}/wallet-${org_msp}"
    local log_file="/tmp/fabric-client-${org_msp}.log"
    local admin_msp_dir="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/bootstrap-admin.${org_domain}/msp"

    mkdir -p "$wallet_dir"

    # Kill existing instance on this port
    fuser -k "${server_port}/tcp" 2>/dev/null || true
    sleep 1

    print_status $YELLOW "Starting ${org_msp} client on port ${server_port}..."

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
    disown $pid 2>/dev/null || true

    # Wait for ready
    local max=30
    local i=1
    while [ $i -le $max ]; do
        if curl -s "http://localhost:${server_port}/health" > /dev/null 2>&1; then
            print_status $GREEN "✓ ${org_msp} client ready on :${server_port} (PID $pid)"
            return 0
        fi
        sleep 2
        i=$((i+1))
    done

    print_status $RED "✗ ${org_msp} client failed to start — check $log_file"
    return 1
}

# ─── Main ─────────────────────────────────────────────────────────────────────

print_status $GREEN "=== Starting Supply Chain Clients ==="
echo "  DEPLOY_ORG1=${DEPLOY_ORG1}  DEPLOY_ORG2=${DEPLOY_ORG2}  DEPLOY_ORG3=${DEPLOY_ORG3}"
echo ""

# Build once
build_client

# Setup crypto and start clients for enabled orgs only
print_status $YELLOW "Setting up crypto materials..."
[ "${DEPLOY_ORG1}" = "true" ] && setup_crypto_for_org "${ORG1_DOMAIN}" "${ORG1_NAME}" "${PEER0_ORG1_PORT}" "${CA_ORG1_PORT}" "${CA_ORG1_NAME}"
[ "${DEPLOY_ORG2}" = "true" ] && setup_crypto_for_org "${ORG2_DOMAIN}" "${ORG2_NAME}" "${PEER0_ORG2_PORT}" "${CA_ORG2_PORT}" "${CA_ORG2_NAME}"
[ "${DEPLOY_ORG3}" = "true" ] && setup_crypto_for_org "${ORG3_DOMAIN}" "${ORG3_NAME}" "${PEER0_ORG3_PORT}" "${CA_ORG3_PORT}" "${CA_ORG3_NAME}"

[ "${DEPLOY_ORG1}" = "true" ] && start_client_for_org "${ORG1_DOMAIN}" "${ORG1_NAME}" "${PEER0_ORG1_PORT}" "${CA_ORG1_PORT}" "${CA_ORG1_NAME}" "8080"
[ "${DEPLOY_ORG2}" = "true" ] && start_client_for_org "${ORG2_DOMAIN}" "${ORG2_NAME}" "${PEER0_ORG2_PORT}" "${CA_ORG2_PORT}" "${CA_ORG2_NAME}" "8081"
[ "${DEPLOY_ORG3}" = "true" ] && start_client_for_org "${ORG3_DOMAIN}" "${ORG3_NAME}" "${PEER0_ORG3_PORT}" "${CA_ORG3_PORT}" "${CA_ORG3_NAME}" "8082"

print_status $GREEN ""
print_status $GREEN "=== Clients Running ==="
echo ""
[ "${DEPLOY_ORG1}" = "true" ] && echo "  ${ORG1_NAME} (Manufacturer) → http://localhost:8080  log: /tmp/fabric-client-${ORG1_NAME}.log"
[ "${DEPLOY_ORG2}" = "true" ] && echo "  ${ORG2_NAME} (Distributor)  → http://localhost:8081  log: /tmp/fabric-client-${ORG2_NAME}.log"
[ "${DEPLOY_ORG3}" = "true" ] && echo "  ${ORG3_NAME} (Retailer)     → http://localhost:8082  log: /tmp/fabric-client-${ORG3_NAME}.log"
echo ""
echo "  Health checks:"
[ "${DEPLOY_ORG1}" = "true" ] && echo "    curl http://localhost:8080/health"
[ "${DEPLOY_ORG2}" = "true" ] && echo "    curl http://localhost:8081/health"
[ "${DEPLOY_ORG3}" = "true" ] && echo "    curl http://localhost:8082/health"
echo ""
echo "  Stop all clients:"
echo "    pkill -f fabric-client"
echo ""
print_status $YELLOW "Login credentials: username=admin  password=asdqwe123"
echo ""
[ "${DEPLOY_ORG2}" = "true" ] && print_status $YELLOW "${ORG2_NAME} accepts custody transfers initiated by ${ORG1_NAME}."
[ "${DEPLOY_ORG3}" = "true" ] && print_status $YELLOW "${ORG3_NAME} accepts custody transfers initiated by ${ORG2_NAME}."
