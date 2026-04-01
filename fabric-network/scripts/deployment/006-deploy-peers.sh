#!/bin/bash

# Script 006: Deploy Peer Services
# This script deploys peer services for both organizations with CouchDB

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

# Check deployment flags
DEPLOY_ORG1=${DEPLOY_ORG1:-true}
DEPLOY_ORG2=${DEPLOY_ORG2:-true}

print_status $GREEN "=== Starting Peer Deployment ==="
print_status $YELLOW "Deployment Mode:"
echo "  Org1: $DEPLOY_ORG1"
echo "  Org2: $DEPLOY_ORG2"
echo ""

# Function to generate CouchDB configuration file
create_couchdb_config() {
    local org_domain=$1
    local couchdb_port=$2

    # Create config directory based on org domain
    local org_dir=$(echo "$org_domain" | cut -d'.' -f1)
    local config_dir="${PROJECT_ROOT}/config/couchdb/${org_dir}"

    mkdir -p "$config_dir"

    # Generate CouchDB port configuration file for official Apache CouchDB image
    cat > "${config_dir}/10-port.ini" << EOF
[httpd]
port = ${couchdb_port}

[chttpd]
port = ${couchdb_port}
bind_address = 127.0.0.1
EOF

    print_status $GREEN "✓ Generated CouchDB config: ${config_dir}/10-port.ini"
}

# Function to create peer docker-compose file
create_peer_compose() {
    local org_domain=$1
    local org_name=$2
    local peer_name=$3
    local peer_port=$4
    local peer_ssl_port=$5
    local couchdb_port=$6
    local couchdb_user=$7
    local couchdb_pass=$8
    local peer_metrics_port=$9
    local peer_profile_port=${10}
    local compose_file=${11}

    cat > "$compose_file" << EOF
version: '3.8'

services:
  couchdb.${org_domain}:
    container_name: couchdb.${org_domain}
    image: couchdb:3.3
    environment:
      - COUCHDB_USER=${couchdb_user}
      - COUCHDB_PASSWORD=${couchdb_pass}
      - ERL_FLAGS="-kernel inet_dist_use_interface {127,0,0,1}"
    volumes:
      - couchdb.${org_domain}_data:/opt/couchdb/data
      - ${PROJECT_ROOT}/config/couchdb/${org_dir}/10-port.ini:/opt/couchdb/etc/local.d/10-port.ini
    network_mode: host
    healthcheck:
      test: ["CMD", "curl", "-f", "-s", "http://localhost:${couchdb_port}/"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 60s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: \${COUCHDB_MEMORY_LIMIT}
          cpus: '\${COUCHDB_CPU_LIMIT}'
        reservations:
          memory: \${COUCHDB_MEMORY_RESERVE}
          cpus: '\${COUCHDB_CPU_RESERVE}'

  ${peer_name}.${org_domain}:
    container_name: ${peer_name}.${org_domain}
    image: hyperledger/fabric-peer:\${FABRIC_VERSION}
    environment:
      - CORE_PEER_ID=${peer_name}.${org_domain}
      - CORE_PEER_ADDRESS=${peer_name}.${org_domain}:${peer_port}
      - CORE_PEER_LISTENADDRESS=0.0.0.0:${peer_port}
      - CORE_PEER_CHAINCODEADDRESS=${peer_name}.${org_domain}:${peer_ssl_port}
      - CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:${peer_ssl_port}
      - CORE_PEER_GOSSIP_BOOTSTRAP=${peer_name}.${org_domain}:${peer_port}
      - CORE_PEER_GOSSIP_EXTERNALENDPOINT=${peer_name}.${org_domain}:${peer_port}
      - CORE_PEER_LOCALMSPID=${org_name}
      - CORE_PEER_TLS_ENABLED=\${TLS_ENABLED}
      - CORE_PEER_TLS_CERT_FILE=/etc/hyperledger/fabric/tls/server.crt
      - CORE_PEER_TLS_KEY_FILE=/etc/hyperledger/fabric/tls/server.key
      - CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
      - CORE_LEDGER_STATE_STATEDATABASE=CouchDB
      - CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS=localhost:${couchdb_port}
      - CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME=${couchdb_user}
      - CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD=${couchdb_pass}
      - FABRIC_LOGGING_SPEC=\${PEER_LOG_LEVEL}
      - CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock
      - DOCKER_HOST=unix:///host/var/run/docker.sock
      - CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=host
      - FABRIC_CFG_PATH=/etc/hyperledger/fabric
      - CORE_OPERATIONS_LISTENADDRESS=0.0.0.0:$peer_metrics_port
      - CORE_METRICS_PROVIDER=prometheus
      - CORE_PEER_PROFILE_ENABLED=true
      - CORE_PEER_PROFILE_LISTENADDRESS=0.0.0.0:$peer_profile_port
      - CORE_CHAINCODE_LOGGING_LEVEL=INFO
      - CORE_CHAINCODE_LOGGING_SHIM=INFO
      - CORE_CHAINCODE_LOGGING_FORMAT= '%{color}%{time:2006-01-02 15:04:15.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}'
      - CORE_CHAINCODE_MODE=dev
      - GOPROXY=https://goproxy.cn,direct
      - GOSUMDB=off
      - GO111MODULE=on
      - CORE_CHAINCODE_BUILDER=hyperledger/fabric-ccenv:2.5
      - CORE_CHAINCODE_EXTERNALBUILDERS=[]
    working_dir: /opt/gopath/src/github.com/hyperledger/fabric/peer
    command: sh -c "apt-get update && apt-get install -y curl docker-cli && peer node start"
    volumes:
      - /var/run/docker.sock:/host/var/run/docker.sock
      - ${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/msp:/etc/hyperledger/fabric/msp
      - ${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls:/etc/hyperledger/fabric/tls
      - ${PROJECT_ROOT}/config/channel-artifacts:/etc/hyperledger/fabric/channel-artifacts
      - ${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp:/etc/hyperledger/fabric/admin-msp
      - ${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/user1.${org_domain}/msp:/etc/hyperledger/fabric/client-msp
      - ${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/users/ca-admin.${ORDERER_DOMAIN}/msp:/etc/hyperledger/fabric/orderer-admin-msp
      - ${peer_name}.${org_domain}:/var/hyperledger/production
    depends_on:
      - couchdb.${org_domain}
    network_mode: host
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:$peer_metrics_port/metrics"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 40s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: \${PEER_MEMORY_LIMIT}
          cpus: '\${PEER_CPU_LIMIT}'
        reservations:
          memory: \${PEER_MEMORY_RESERVE}
          cpus: '\${PEER_CPU_RESERVE}'

volumes:
  couchdb.${org_domain}_data:
    driver: local
  ${peer_name}.${org_domain}:
    driver: local
EOF
}

# Function to verify prerequisites
verify_peer_prerequisites() {
    local org_domain=$1
    local peer_name=$2

    # Check if peer MSP exists
    if [ ! -d "${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/msp" ]; then
        print_status $RED "Error: Peer MSP certificates not found for ${peer_name}.${org_domain}. Please run script 003 first."
        return 1
    fi

    # Check if peer TLS certificates exist
    if [ ! -d "${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls" ]; then
        print_status $RED "Error: Peer TLS certificates not found for ${peer_name}.${org_domain}. Please run script 003 first."
        return 1
    fi

    return 0
}

# Function to deploy peer
deploy_peer() {
    local org_domain=$1
    local org_name=$2
    local peer_name=$3
    local peer_port=$4
    local peer_ssl_port=$5
    local couchdb_port=$6
    local couchdb_user=$7
    local couchdb_pass=$8
    local peer_metrics_port=$9
    local peer_profile_port=${10}

    print_status $YELLOW "=== Deploying ${peer_name}.${org_domain} ==="

    # Verify prerequisites
    verify_peer_prerequisites "$org_domain" "$peer_name"
    if [ $? -ne 0 ]; then
        exit 1
    fi

    # Create docker-compose directory
    local org_dir=$(echo "$org_domain" | cut -d'.' -f1)
    mkdir -p "${PROJECT_ROOT}/docker-compose/${org_dir}"

    # Generate CouchDB configuration file
    create_couchdb_config "$org_domain" "$couchdb_port"

    # Create peer docker-compose file
    local compose_file="${PROJECT_ROOT}/docker-compose/${org_dir}/peer.yml"
    rm -f "$compose_file"
    create_peer_compose \
        "$org_domain" \
        "$org_name" \
        "$peer_name" \
        "$peer_port" \
        "$peer_ssl_port" \
        "$couchdb_port" \
        "$couchdb_user" \
        "$couchdb_pass" \
        "$peer_metrics_port" \
        "$peer_profile_port" \
        "$compose_file"

    # Deploy peer
    print_status $YELLOW "Deploying ${peer_name}.${org_domain} service..."

    # Function to check if a port is in use
    is_port_in_use() {
        local port=$1
        if ss -tlnp 2>&1 | grep -q ":${port} " || netstat -tlnp 2>&1 | grep -q ":${port} "; then
            return 0  # Port is in use
        fi
        return 1  # Port is free
    }

    # Function to wait for a port to be free
    wait_for_port_free() {
        local port=$1
        local max_wait=30
        local waited=0

        while is_port_in_use $port && [ $waited -lt $max_wait ]; do
            echo "    Port ${port} still in use, waiting... ($waited/${max_wait}s)"
            sleep 2
            waited=$((waited + 2))
        done

        if is_port_in_use $port; then
            print_status $RED "✗ Port ${port} still in use after ${max_wait}s"
            return 1
        fi

        print_status $GREEN "✓ Port ${port} is now free"
        return 0
    }

    # Stop and remove existing containers if they exist
    if docker ps -a | grep -q "${peer_name}.${org_domain}"; then
        print_status $YELLOW "Stopping existing ${peer_name}.${org_domain} container..."
        docker stop ${peer_name}.${org_domain} 2>/dev/null || true
        docker rm ${peer_name}.${org_domain} 2>/dev/null || true
    fi

    if docker ps -a | grep -q "couchdb.${org_domain}"; then
        print_status $YELLOW "Stopping existing couchdb.${org_domain} container..."
        docker stop couchdb.${org_domain} 2>/dev/null || true
        docker rm couchdb.${org_domain} 2>/dev/null || true
    fi

    # Wait for ports to be released
    print_status $YELLOW "Waiting for ports to be released..."
    sleep 5

    # Kill any zombie processes that might be holding the port
    if is_port_in_use ${couchdb_port}; then
        print_status $YELLOW "Attempting to kill processes on port ${couchdb_port}..."
        fuser -k ${couchdb_port}/tcp 2>/dev/null || true
        sleep 2
    fi

    # Ensure CouchDB port is free before starting
    if ! wait_for_port_free ${couchdb_port}; then
        print_status $RED "✗ Cannot free port ${couchdb_port}, aborting deployment"
        return 1
    fi

    # Additional wait to ensure port is fully released
    sleep 3

    docker compose -f "$compose_file" --env-file "${PROJECT_ROOT}/.env" up -d

    # Give containers time to start
    sleep 2

    print_status $GREEN "✓ ${peer_name}.${org_domain} deployed"

    # Wait for CouchDB to be ready first
    print_status $YELLOW "Waiting for CouchDB (${couchdb_port}) to be ready..."

    local couchdb_max_attempts=10
    local couchdb_attempt=1

    while [ $couchdb_attempt -le $couchdb_max_attempts ]; do
        # Check if CouchDB container is running and healthy
        if docker ps | grep -q "couchdb.${org_domain}" && \
           docker inspect couchdb.${org_domain} | grep -q '"Status": "healthy'; then
            # Verify CouchDB is responsive
            if curl -s http://localhost:${couchdb_port}/ > /dev/null 2>&1; then
                print_status $GREEN "✓ CouchDB is ready"
                break
            fi
        fi

        echo "  Attempt $couchdb_attempt/$couchdb_max_attempts: Waiting for CouchDB..."
        sleep 2
        couchdb_attempt=$((couchdb_attempt + 1))
    done

    if [ $couchdb_attempt -gt $couchdb_max_attempts ]; then
        print_status $RED "✗ CouchDB failed to start within timeout period"
        print_status $YELLOW "Check CouchDB logs with: docker logs couchdb.${org_domain}"
        return 1
    fi

    # Wait for peer to be ready
    print_status $YELLOW "Waiting for ${peer_name}.${org_domain} to be ready..."

    local max_attempts=60
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        # Check if container is running and healthy
        local container_running=false
        local container_healthy=false
        local metrics_accessible=false

        if docker ps | grep -q "${peer_name}.${org_domain}"; then
            container_running=true
        fi

        if $container_running; then
            local health_status=$(docker inspect ${peer_name}.${org_domain} --format='{{.State.Health.Status}}' 2>/dev/null || echo "unknown")
            if [ "$health_status" = "healthy" ]; then
                container_healthy=true
            fi
        fi

        if $container_healthy; then
            if curl -s http://localhost:${peer_metrics_port}/metrics > /dev/null 2>&1; then
                metrics_accessible=true
            fi
        fi

        if $container_running && $container_healthy && $metrics_accessible; then
            print_status $GREEN "✓ ${peer_name}.${org_domain} is ready"
            break
        fi

        # Show diagnostic information
        echo "  Attempt $attempt/$max_attempts: Waiting for peer..."
        echo "    Container running: $container_running"
        if $container_running; then
            local health_status=$(docker inspect ${peer_name}.${org_domain} --format='{{.State.Health.Status}}' 2>/dev/null || echo "unknown")
            echo "    Health status: $health_status"
        fi
        echo "    Metrics accessible: $metrics_accessible"
        echo "    Checking port ${peer_metrics_port}..."

        sleep 3
        attempt=$((attempt + 1))
    done

    if [ $attempt -gt $max_attempts ]; then
        print_status $RED "✗ ${peer_name}.${org_domain} failed to start within timeout period"
        print_status $YELLOW "Check peer logs with: docker logs --tail 50 ${peer_name}.${org_domain}"
        print_status $YELLOW "Check CouchDB logs with: docker logs --tail 50 couchdb.${org_domain}"
        print_status $YELLOW "Test metrics endpoint: curl http://localhost:${peer_metrics_port}/metrics"
        print_status $YELLOW "Test CouchDB: curl http://localhost:${couchdb_port}/"
        return 1
    fi

    # Verify peer status
    print_status $YELLOW "Verifying ${peer_name}.${org_domain} status..."

    # Check if container is running
    if docker ps | grep -q "${peer_name}.${org_domain}"; then
        print_status $GREEN "✓ ${peer_name}.${org_domain} container is running"
    else
        print_status $RED "✗ ${peer_name}.${org_domain} container is not running"
        return 1
    fi

    # Verify local binary availability
    if "${FABRIC_BIN_PATH}/peer" version 2>&1 | grep -q "Version:"; then
        local peer_version=$("${FABRIC_BIN_PATH}/peer" version | grep "Version:" | cut -d: -f2 | xargs)
        print_status $GREEN "✓ Peer binary available (Version: $peer_version)"
    else
        print_status $YELLOW "⚠ Peer binary not found at ${FABRIC_BIN_PATH}"
    fi



    return 0
}

# Deploy Org1 Peer
if [ "$DEPLOY_ORG1" = true ]; then
    deploy_peer \
        "${ORG1_DOMAIN}" \
        "${ORG1_NAME}" \
        "peer0" \
        "${PEER0_ORG1_PORT}" \
        "${PEER0_ORG1_SSL_PORT}" \
        "${COUCHDB_ORG1_PORT}" \
        "${COUCHDB_ORG1_USER}" \
        "${COUCHDB_ORG1_PASSWORD}" \
        "${PEER0_ORG1_METRICS_PORT}" \
        "${PEER0_ORG1_PROFILE_PORT}" \
        "${PROJECT_ROOT}/docker-compose/org1/peer.yml"
fi

# Deploy Org2 Peer
if [ "$DEPLOY_ORG2" = true ]; then
    deploy_peer \
        "${ORG2_DOMAIN}" \
        "${ORG2_NAME}" \
        "peer0" \
        "${PEER0_ORG2_PORT}" \
        "${PEER0_ORG2_SSL_PORT}" \
        "${COUCHDB_ORG2_PORT}" \
        "${COUCHDB_ORG2_USER}" \
        "${COUCHDB_ORG2_PASSWORD}" \
        "${PEER0_ORG2_METRICS_PORT}" \
        "${PEER0_ORG2_PROFILE_PORT}" \
        "${PROJECT_ROOT}/docker-compose/org2/peer.yml"
fi

sleep 20;
if [ "$DEPLOY_ORG1" = true ]; then
    # Check if peer is listening on the correct ports
    print_status $YELLOW "Checking ${ORG1_COMMON_NAME} network connectivity..."

    if netstat -tlnp 2>&1 | grep -q ":${PEER0_ORG1_PORT}" || ss -tlnp 2>&1 | grep -q ":${PEER0_ORG1_PORT}"; then
        print_status $GREEN "✓ ${ORG1_COMMON_NAME} listening on port ${PEER0_ORG1_PORT}"
    else
        print_status $RED "✗ ${ORG1_COMMON_NAME} not listening on port ${PEER0_ORG1_PORT}"
        exit 1
    fi
fi
if [ "$DEPLOY_ORG2" = true ]; then
    # Check if peer is listening on the correct ports
    print_status $YELLOW "Checking ${ORG2_COMMON_NAME} network connectivity..."

    if netstat -tlnp 2>&1 | grep -q ":${PEER0_ORG2_PORT}" || ss -tlnp 2>&1 | grep -q ":${PEER0_ORG2_PORT}"; then
        print_status $GREEN "✓ ${ORG2_COMMON_NAME} listening on port ${PEER0_ORG2_PORT}"
    else
        print_status $RED "✗ ${ORG2_COMMON_NAME} not listening on port ${PEER0_ORG2_PORT}"
        exit 1
    fi
fi

# Display peer information
print_status $YELLOW "=== Peer Information ==="

if [ "$DEPLOY_ORG1" = true ]; then
    echo "  Org1 Peer:"
    echo "    Container Name: peer0.${ORG1_DOMAIN}"
    echo "    MSP ID: ${ORG1_NAME}"
    echo "    Listen Address: 0.0.0.0:${PEER0_ORG1_PORT}"
    echo "    Chaincode Port: ${PEER0_ORG1_SSL_PORT}"
    echo "    CouchDB Port: ${COUCHDB_ORG1_PORT}"
    echo "    Domain: ${ORG1_DOMAIN}"
    echo ""
fi

if [ "$DEPLOY_ORG2" = true ]; then
    echo "  Org2 Peer:"
    echo "    Container Name: peer0.${ORG2_DOMAIN}"
    echo "    MSP ID: ${ORG2_NAME}"
    echo "    Listen Address: 0.0.0.0:${PEER0_ORG2_PORT}"
    echo "    Chaincode Port: ${PEER0_ORG2_SSL_PORT}"
    echo "    CouchDB Port: ${COUCHDB_ORG2_PORT}"
    echo "    Domain: ${ORG2_DOMAIN}"
    echo ""
fi

# Display useful commands
print_status $YELLOW "=== Useful Commands ==="

if [ "$DEPLOY_ORG1" = true ]; then
    echo "  Org1 Peer:"
    echo "    View logs: docker logs -f peer0.${ORG1_DOMAIN}"
    echo "    Check status: docker ps | grep peer0.${ORG1_DOMAIN}"
    echo "    Execute peer command locally:"
    echo "      export CORE_PEER_LOCALMSPID='${ORG1_NAME}'"
    echo "      export CORE_PEER_TLS_ROOTCERT_FILE='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt'"
    echo "      export CORE_PEER_MSPCONFIGPATH='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp'"
    echo "      ${FABRIC_BIN_PATH}/peer <command>"
    echo "    View CouchDB: curl http://localhost:${COUCHDB_ORG1_PORT}/_utils"
    echo ""
fi

if [ "$DEPLOY_ORG2" = true ]; then
    echo "  Org2 Peer:"
    echo "    View logs: docker logs -f peer0.${ORG2_DOMAIN}"
    echo "    Check status: docker ps | grep peer0.${ORG2_DOMAIN}"
    echo "    Execute peer command locally:"
    echo "      export CORE_PEER_LOCALMSPID='${ORG2_NAME}'"
    echo "      export CORE_PEER_TLS_ROOTCERT_FILE='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/ca.crt'"
    echo "      export CORE_PEER_MSPCONFIGPATH='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/users/admin.${ORG2_DOMAIN}/msp'"
    echo "      ${FABRIC_BIN_PATH}/peer <command>"
    echo "    View CouchDB: curl http://localhost:${COUCHDB_ORG2_PORT}/_utils"
    echo ""
fi

print_status $GREEN "=== Peer Deployment Completed Successfully ==="
print_status $YELLOW "Next step: Run 007-create-channel.sh"
echo ""
