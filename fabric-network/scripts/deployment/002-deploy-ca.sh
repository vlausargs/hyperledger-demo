#!/bin/bash

# Script 002: Deploy Fabric CA Services
# This script deploys Fabric CA services for the orderer and organizations

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

# Check deployment flags
DEPLOY_ORDERER=${DEPLOY_ORDERER:-true}
DEPLOY_ORG1=${DEPLOY_ORG1:-true}
DEPLOY_ORG2=${DEPLOY_ORG2:-true}

print_status $GREEN "=== Starting Fabric CA Deployment ==="
print_status $YELLOW "Deployment Mode:"
echo "  Orderer CA: $DEPLOY_ORDERER"
echo "  Org1 CA: $DEPLOY_ORG1"
echo "  Org2 CA: $DEPLOY_ORG2"
echo ""

# Function to create CA configuration file
create_ca_config() {
    local ca_name=$1
    local ca_hostname=$2
    local ca_port=$3
    local db_host=$4
    local db_user=$5
    local db_pass=$6
    local db_name=$7
    local config_file=$8

    cat > "$config_file" << EOF
services:
  ${ca_name}:
    image: hyperledger/fabric-ca:${CA_VERSION}
    container_name: ${ca_name}
    environment:
      - FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server
      - FABRIC_CA_SERVER_CA_NAME=${ca_name}
      - FABRIC_CA_SERVER_CA_CERTFILE=/etc/hyperledger/fabric-ca-server-config/ca.${ca_hostname}-cert.pem
      - FABRIC_CA_SERVER_CA_KEYFILE=/etc/hyperledger/fabric-ca-server-config/priv_sk
      - FABRIC_CA_SERVER_TLS_ENABLED=false
      - FABRIC_CA_SERVER_DB_TYPE=postgres
      - FABRIC_CA_SERVER_DB_DATASOURCE=host=${db_host} port=5432 user=${db_user} password=${db_pass} dbname=${db_name} sslmode=disable
    ports:
      - "${ca_port}:7054"
    volumes:
      - ${ca_name}_hlf_data:/etc/hyperledger/fabric-ca-server
    networks:
      - ${NETWORK_NAME}
    command: sh -c "fabric-ca-server start -b ${ca_hostname}-admin:${ca_hostname}-adminpw -d"
    healthcheck:
      test: ["CMD", "sh", "-c", "nc -z localhost 7054 || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: ${CA_MEMORY_LIMIT}
          cpus: '${CA_CPU_LIMIT}'
        reservations:
          memory: ${CA_MEMORY_RESERVE}
          cpus: '${CA_CPU_RESERVE}'

volumes:
  ${ca_name}_hlf_data:
    driver: local

networks:
  ${NETWORK_NAME}:
    driver: bridge
    name: ${NETWORK_NAME}
EOF
}

# Create docker-compose directory for CA
mkdir -p "${PROJECT_ROOT}/docker-compose/ca"

# Deploy Orderer CA
if [ "$DEPLOY_ORDERER" = true ]; then
    print_status $YELLOW "Deploying Orderer CA..."

    compose_file="${PROJECT_ROOT}/docker-compose/ca/orderer-ca.yml"
    create_ca_config \
        "${CA_ORDERER_NAME}" \
        "${CA_ORDERER_HOSTNAME}" \
        "${CA_ORDERER_PORT}" \
        "${POSTGRES_ORDERER_HOST}" \
        "${POSTGRES_ORDERER_USER}" \
        "${POSTGRES_ORDERER_PASSWORD}" \
        "${POSTGRES_ORDERER_DB}" \
        "$compose_file"

    docker-compose -f "$compose_file" up -d
    print_status $GREEN "✓ Orderer CA deployed successfully"
fi

# Deploy Org1 CA
if [ "$DEPLOY_ORG1" = true ]; then
    print_status $YELLOW "Deploying Org1 CA..."

    compose_file="${PROJECT_ROOT}/docker-compose/ca/org1-ca.yml"
    create_ca_config \
        "${CA_ORG1_NAME}" \
        "${CA_ORG1_HOSTNAME}" \
        "${CA_ORG1_PORT}" \
        "${POSTGRES_ORG1_HOST}" \
        "${POSTGRES_ORG1_USER}" \
        "${POSTGRES_ORG1_PASSWORD}" \
        "${POSTGRES_ORG1_DB}" \
        "$compose_file"

    docker-compose -f "$compose_file" up -d
    print_status $GREEN "✓ Org1 CA deployed successfully"
fi

# Deploy Org2 CA
if [ "$DEPLOY_ORG2" = true ]; then
    print_status $YELLOW "Deploying Org2 CA..."

    compose_file="${PROJECT_ROOT}/docker-compose/ca/org2-ca.yml"
    create_ca_config \
        "${CA_ORG2_NAME}" \
        "${CA_ORG2_HOSTNAME}" \
        "${CA_ORG2_PORT}" \
        "${POSTGRES_ORG2_HOST}" \
        "${POSTGRES_ORG2_USER}" \
        "${POSTGRES_ORG2_PASSWORD}" \
        "${POSTGRES_ORG2_DB}" \
        "$compose_file"

    docker-compose -f "$compose_file" up -d
    print_status $GREEN "✓ Org2 CA deployed successfully"
fi

# Wait for CA services to be ready
print_status $YELLOW "Waiting for Fabric CA services to be ready..."
sleep 15

# Verify CA services
verify_ca() {
    local ca_name=$1
    local ca_hostname=$2
    local ca_port=$3

    print_status $YELLOW "Verifying $ca_name..."

    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:${ca_port}/cainfo > /dev/null 2>&1; then
            print_status $GREEN "✓ $ca_name is ready"
            return 0
        fi

        echo "  Attempt $attempt/$max_attempts: Waiting for CA service..."
        sleep 2
        attempt=$((attempt + 1))
    done

    print_status $RED "✗ $ca_name failed to start"
    return 1
}

# Verify all CAs if they're deployed
if [ "$DEPLOY_ORDERER" = true ]; then
    verify_ca "${CA_ORDERER_NAME}" "${CA_ORDERER_HOSTNAME}" "${CA_ORDERER_PORT}"
fi

if [ "$DEPLOY_ORG1" = true ]; then
    verify_ca "${CA_ORG1_NAME}" "${CA_ORG1_HOSTNAME}" "${CA_ORG1_PORT}"
fi

if [ "$DEPLOY_ORG2" = true ]; then
    verify_ca "${CA_ORG2_NAME}" "${CA_ORG2_HOSTNAME}" "${CA_ORG2_PORT}"
fi

print_status $GREEN "=== Fabric CA Deployment Completed Successfully ==="
print_status $YELLOW "Next step: Run 003-setup-ca.sh"
