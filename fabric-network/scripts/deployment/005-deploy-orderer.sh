#!/bin/bash

# Script 005: Deploy Orderer Service
# This script deploys the orderer service for the Hyperledger Fabric network

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

# Load Fabric environment helper functions
source "${SCRIPT_DIR}/../helpers/fabric-env.sh"

if [ ! -f "${PROJECT_ROOT}/.env" ]; then
    print_status $RED "Error: .env file not found at ${PROJECT_ROOT}/.env"
    exit 1
fi

source "${PROJECT_ROOT}/.env"

# Orderer type is loaded from .env file (etcdraft or solo)

# Check deployment flags
DEPLOY_ORDERER=${DEPLOY_ORDERER:-true}

if [ "$DEPLOY_ORDERER" != true ]; then
    print_status $YELLOW "Orderer deployment is disabled. Skipping..."
    exit 0
fi

print_status $GREEN "=== Starting Orderer Deployment ==="
print_status $YELLOW "Orderer deployment mode: ENABLED"
echo ""

# Verify prerequisites
print_status $YELLOW "Verifying prerequisites..."

# Check if genesis block exists
if [ ! -f "${PROJECT_ROOT}/config/channel-artifacts/genesis.block" ]; then
    print_status $RED "Error: Genesis block not found. Please run script 004 first."
    exit 1
fi

# Check if orderer MSP exists
if [ ! -d "${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/msp" ]; then
    print_status $RED "Error: Orderer MSP certificates not found. Please run script 003 first."
    exit 1
fi

# Check if orderer TLS certificates exist
if [ ! -d "${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls" ]; then
    print_status $RED "Error: Orderer TLS certificates not found. Please run script 003 first."
    exit 1
fi

print_status $GREEN "✓ Prerequisites verified"

# Create docker-compose directory
mkdir -p "${PROJECT_ROOT}/docker-compose/orderer"

# Create orderer configuration file
print_status $YELLOW "Creating orderer configuration..."

ORDERER_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose/orderer/orderer.yml"

cat > "$ORDERER_COMPOSE_FILE" << EOF
version: '3.8'

services:
  orderer1.${ORDERER_DOMAIN}:
    container_name: orderer1.${ORDERER_DOMAIN}
    image: hyperledger/fabric-orderer:\${FABRIC_VERSION}
    environment:
      - FABRIC_LOGGING_SPEC=\${ORDERER_LOG_LEVEL}
      - ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
      - ORDERER_GENERAL_LISTENPORT=\${ORDERER_PORT}
      - ORDERER_GENERAL_LOCALMSPID=\${ORDERER_ORG_NAME}
      - ORDERER_GENERAL_LOCALMSPDIR=/etc/hyperledger/fabric/msp
      - ORDERER_GENERAL_TLS_ENABLED=\${TLS_ENABLED}
      - ORDERER_GENERAL_TLS_PRIVATEKEY=/etc/hyperledger/fabric/tls/server.key
      - ORDERER_GENERAL_TLS_CERTIFICATE=/etc/hyperledger/fabric/tls/server.crt
      - ORDERER_GENERAL_TLS_ROOTCAS=/etc/hyperledger/fabric/tls/ca.crt
      - ORDERER_KAFKA_TOPIC_REPLICATIONFACTOR=1
      - ORDERER_KAFKA_VERBOSE=true
      - ORDERER_GENERAL_CLUSTER_LISTENADDRESS=0.0.0.0
      - ORDERER_GENERAL_CLUSTER_LISTENPORT=\${ORDERER_SSL_PORT}
      - ORDERER_GENERAL_CLUSTER_CLIENTAUTHREQUIRED=\${TLS_CLIENTAUTHREQUIRED}
      - ORDERER_GENERAL_CLUSTER_CLIENTROOTCAS_FILES=/etc/hyperledger/fabric/tls/ca.crt,/etc/hyperledger/fabric/org-msp/tlscacerts/*.pem
      - ORDERER_GENERAL_CLUSTER_SERVERCERTIFICATE=/etc/hyperledger/fabric/tls/server.crt
      - ORDERER_GENERAL_CLUSTER_SERVERPRIVATEKEY=/etc/hyperledger/fabric/tls/server.key
      - ORDERER_GENERAL_GENESISFILE=/etc/hyperledger/fabric/channel-artifacts/genesis.block
      - ORDERER_GENERAL_BOOTSTRAPMETHOD=file
      - ORDERER_CHANNELPARTICIPATION_ENABLED=true
      - ORDERER_ADMIN_TLS_ENABLED=true
      - ORDERER_ADMIN_TLS_CERTIFICATE=/etc/hyperledger/fabric/tls/server.crt
      - ORDERER_ADMIN_TLS_PRIVATEKEY=/etc/hyperledger/fabric/tls/server.key
      - ORDERER_ADMIN_TLS_CLIENTROOTCAS=/etc/hyperledger/fabric/tls/ca.crt
      - ORDERER_ADMIN_TLS_CLIENTCERTREQUIRED=true
      - ORDERER_ADMIN_LISTENADDRESS=0.0.0.0:7053
      - ORDERER_METRICS_PROVIDER=prometheus
      - ORDERER_OPERATIONS_LISTENADDRESS=0.0.0.0:8443
      - ORDERER_DEBUG_BROADCASTTRACEDETECTION=data
    working_dir: /opt/gopath/src/github.com/hyperledger/fabric/orderers/orderer1.${ORDERER_DOMAIN}
    command: sh -c "apt-get update && apt-get install -y curl && orderer"
    volumes:
      - ${PROJECT_ROOT}/config/channel-artifacts:/etc/hyperledger/fabric/channel-artifacts
      - ${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/msp:/etc/hyperledger/fabric/msp
      - ${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls:/etc/hyperledger/fabric/tls
      - ${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/msp:/etc/hyperledger/fabric/org-msp
      - orderer1.${ORDERER_DOMAIN}:/var/hyperledger/production/orderer
    network_mode: host
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8443/metrics"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 40s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: \${ORDERER_MEMORY_LIMIT}
          cpus: '\${ORDERER_CPU_LIMIT}'
        reservations:
          memory: \${ORDERER_MEMORY_RESERVE}
          cpus: '\${ORDERER_CPU_RESERVE}'

volumes:
  orderer1.${ORDERER_DOMAIN}:
    driver: local

EOF

print_status $GREEN "✓ Orderer configuration created"

# Deploy the orderer
print_status $YELLOW "Deploying orderer service..."

docker compose -f "$ORDERER_COMPOSE_FILE" --env-file "${PROJECT_ROOT}/.env" up -d

print_status $GREEN "✓ Orderer service deployed"

# Wait for orderer to be ready
print_status $YELLOW "Waiting for orderer to be ready..."

max_attempts=120
attempt=1

while [ $attempt -le $max_attempts ]; do
    # Check if container is running and healthy
    if docker ps | grep -q "orderer1.${ORDERER_DOMAIN}" && \
       docker inspect orderer1.${ORDERER_DOMAIN} | grep -q '"Status": "healthy"'; then
        # Verify orderer is responsive by checking metrics endpoint
        if curl -s http://localhost:8443/metrics > /dev/null 2>&1; then
            print_status $GREEN "✓ Orderer is ready"
            break
        fi
    fi

    echo "  Attempt $attempt/$max_attempts: Waiting for orderer..."
    sleep 3
    attempt=$((attempt + 1))
done

if [ $attempt -gt $max_attempts ]; then
    print_status $RED "✗ Orderer failed to start within timeout period"
    print_status $YELLOW "Check orderer logs with: docker logs orderer.${ORDERER_DOMAIN}"
    exit 1
fi

# Verify orderer status
print_status $YELLOW "Verifying orderer status..."
sleep 10;

# Check if container is running
if docker ps | grep -q "orderer1.${ORDERER_DOMAIN}"; then
    print_status $GREEN "✓ Orderer container is running"
else
    print_status $RED "✗ Orderer container is not running"
    exit 1
fi

# Verify local binary availability
if "${FABRIC_BIN_PATH}/orderer" version 2>&1 | grep -q "Version:"; then
    orderer_version=$("${FABRIC_BIN_PATH}/orderer" version | grep "Version:" | cut -d: -f2 | xargs)
    print_status $GREEN "✓ Orderer binary available (Version: $orderer_version)"
else
    print_status $YELLOW "⚠ Orderer binary not found at ${FABRIC_BIN_PATH}"
fi

# Check if orderer is listening on the correct ports
print_status $YELLOW "Checking orderer network connectivity..."

if netstat -tlnp 2>&1 | grep -q ":${ORDERER_PORT}"; then
    print_status $GREEN "✓ Orderer listening on port ${ORDERER_PORT}"
elif ss -tlnp 2>&1 | grep -q ":${ORDERER_PORT}"; then
    print_status $GREEN "✓ Orderer listening on port ${ORDERER_PORT}"
else
    print_status $RED "✗ Orderer not listening on port ${ORDERER_PORT}"
    exit 1
fi

if [ "$ORDERER_TYPE" = "etcdraft" ]; then
    if netstat -tlnp 2>&1 | grep -q ":${ORDERER_SSL_PORT}"; then
        print_status $GREEN "✓ Orderer listening on port ${ORDERER_SSL_PORT} (cluster)"
    elif ss -tlnp 2>&1 | grep -q ":${ORDERER_SSL_PORT}"; then
        print_status $GREEN "✓ Orderer listening on port ${ORDERER_SSL_PORT} (cluster)"
    else
        print_status $RED "✗ Orderer not listening on port ${ORDERER_SSL_PORT}"
        exit 1
    fi
else
    print_status $YELLOW "⚠ Skipping cluster port check (not required for this orderer type)"
fi

# Display orderer information
print_status $YELLOW "=== Orderer Information ==="
echo "  Container Name: orderer1.${ORDERER_DOMAIN}"
echo "  MSP ID: ${ORDERER_ORG_NAME}"
echo "  Listen Address: 0.0.0.0:${ORDERER_PORT}"
echo "  Cluster Port: ${ORDERER_SSL_PORT}"
echo "  Admin Port: 7053"
echo "  Metrics Port: 8443"
echo "  Domain: ${ORDERER_DOMAIN}"
echo ""

# Display useful commands
print_status $YELLOW "=== Useful Commands ==="
echo "  View logs: docker logs -f orderer.${ORDERER_DOMAIN}"
echo "  Check status: docker ps | grep orderer"
echo "  Execute command locally: ${FABRIC_BIN_PATH}/orderer <command>"
echo "  View metrics: curl http://localhost:8443/metrics"
echo ""

print_status $GREEN "=== Orderer Deployment Completed Successfully ==="
print_status $YELLOW "Next step: Run 006-deploy-peers.sh"
