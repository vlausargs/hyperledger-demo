#!/bin/bash

# Script 007: Create and Join Channel
# This script creates the application channel and joins peers to it

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

print_status $GREEN "=== Starting Channel Creation and Join ==="
print_status $YELLOW "Deployment Mode:"
echo "  Orderer: $DEPLOY_ORDERER"
echo "  Org1: $DEPLOY_ORG1"
echo "  Org2: $DEPLOY_ORG2"
echo ""

# Verify prerequisites
print_status $YELLOW "Verifying prerequisites..."

# Check if orderer is running
if ! docker ps | grep -q "orderer1.${ORDERER_DOMAIN}"; then
    print_status $RED "Error: Orderer is not running. Please run script 005 first."
    exit 1
fi

# Check if channel configuration transaction exists
if [ ! -f "${PROJECT_ROOT}/config/channel-artifacts/${CHANNEL_NAME}.tx" ]; then
    print_status $RED "Error: Channel configuration transaction not found. Please run script 004 first."
    exit 1
fi

print_status $GREEN "✓ Prerequisites verified"

# Function to create channel
create_channel() {
    print_status $YELLOW "Creating channel '${CHANNEL_NAME}'..."

    # Determine which org's peer to use for channel creation
    if [ "$DEPLOY_ORG1" = true ]; then
        local peer_container="peer0.${ORG1_DOMAIN}"
        local msp_id="${ORG1_NAME}"
    elif [ "$DEPLOY_ORG2" = true ]; then
        local peer_container="peer0.${ORG2_DOMAIN}"
        local msp_id="${ORG2_NAME}"
    else
        print_status $RED "Error: No organization deployed to create channel"
        exit 1
    fi

    # Create the channel using orderer's TLS CA certificate and admin identity
    docker exec -e CORE_PEER_LOCALMSPID="${msp_id}" \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt \
        -e CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/admin-msp \
        ${peer_container} \
        peer channel create \
        -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} \
        -c ${CHANNEL_NAME} \
        -f /etc/hyperledger/fabric/channel-artifacts/${CHANNEL_NAME}.tx \
        --outputBlock /etc/hyperledger/fabric/channel-artifacts/${CHANNEL_NAME}.block \
        --tls \
        --cafile /etc/hyperledger/fabric/orderer-tls-ca.crt

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Channel '${CHANNEL_NAME}' created successfully"

        # Fix file permissions and ownership (Docker creates files as root)
        sudo chown $(id -u):$(id -g) "${PROJECT_ROOT}/config/channel-artifacts/${CHANNEL_NAME}.block"
        sudo chmod 644 "${PROJECT_ROOT}/config/channel-artifacts/${CHANNEL_NAME}.block"
    else
        print_status $RED "✗ Failed to create channel '${CHANNEL_NAME}'"
        exit 1
    fi
}

# Function to join peer to channel
join_channel() {
    local peer_container=$1
    local org_name=$2

    print_status $YELLOW "Joining ${peer_container} to channel '${CHANNEL_NAME}'..."

    # Join the peer to the channel
    docker exec -e CORE_PEER_LOCALMSPID="${org_name}" \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt \
        -e CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/admin-msp \
        ${peer_container} \
        peer channel join \
        -b /etc/hyperledger/fabric/channel-artifacts/${CHANNEL_NAME}.block \
        --tls \
        --cafile /etc/hyperledger/fabric/orderer-tls-ca.crt

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ ${peer_container} joined channel '${CHANNEL_NAME}' successfully"
    else
        print_status $RED "✗ Failed to join ${peer_container} to channel '${CHANNEL_NAME}'"
        return 1
    fi
}

# Function to update anchor peer
update_anchor_peer() {
    local peer_container=$1
    local org_name=$2
    local org_domain=$3
    local anchor_tx_file=$4

    print_status $YELLOW "Updating anchor peer for ${org_name}..."

    # Update anchor peer using orderer's TLS CA certificate and admin identity
    docker exec -e CORE_PEER_LOCALMSPID="${org_name}" \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt \
        -e CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/admin-msp \
        ${peer_container} \
        peer channel update \
        -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} \
        -c ${CHANNEL_NAME} \
        -f /etc/hyperledger/fabric/channel-artifacts/${anchor_tx_file} \
        --tls \
        --cafile /etc/hyperledger/fabric/orderer-tls-ca.crt

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Anchor peer updated for ${org_name}"
    else
        print_status $RED "✗ Failed to update anchor peer for ${org_name}"
        return 1
    fi
}

# Function to list channels
list_channels() {
    local peer_container=$1
    local org_name=$2

    print_status $YELLOW "Listing channels for ${peer_container}..."

    docker exec -e CORE_PEER_LOCALMSPID="${org_name}" \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt \
        ${peer_container} \
        peer channel list \
        --tls \
        --cafile /etc/hyperledger/fabric/orderer-tls-ca.crt
}

# Copy orderer's TLS CA certificate to peer containers
print_status $YELLOW "Copying orderer's TLS CA certificate to peer containers..."

if [ "$DEPLOY_ORG1" = true ]; then
    docker exec orderer1.${ORDERER_DOMAIN} cat /etc/hyperledger/fabric/tls/ca.crt | \
        docker exec -i peer0.${ORG1_DOMAIN} sh -c 'cat > /etc/hyperledger/fabric/orderer-tls-ca.crt'
    print_status $GREEN "✓ Orderer TLS CA copied to peer0.${ORG1_DOMAIN}"
fi

if [ "$DEPLOY_ORG2" = true ]; then
    docker exec orderer1.${ORDERER_DOMAIN} cat /etc/hyperledger/fabric/tls/ca.crt | \
        docker exec -i peer0.${ORG2_DOMAIN} sh -c 'cat > /etc/hyperledger/fabric/orderer-tls-ca.crt'
    print_status $GREEN "✓ Orderer TLS CA copied to peer0.${ORG2_DOMAIN}"
fi



# Create channel if not exists
print_status $YELLOW "Checking if channel '${CHANNEL_NAME}' already exists..."

if [ "$DEPLOY_ORG1" = true ]; then
    if docker exec peer0.${ORG1_DOMAIN} peer channel list 2>&1 | grep -q "${CHANNEL_NAME}"; then
        print_status $YELLOW "Channel '${CHANNEL_NAME}' already exists. Skipping creation."
    else
        create_channel
    fi
elif [ "$DEPLOY_ORG2" = true ]; then
    if docker exec peer0.${ORG2_DOMAIN} peer channel list 2>&1 | grep -q "${CHANNEL_NAME}"; then
        print_status $YELLOW "Channel '${CHANNEL_NAME}' already exists. Skipping creation."
    else
        create_channel
    fi
fi

# Join peers to channel
if [ "$DEPLOY_ORG1" = true ]; then
    # Check if peer is already joined
    if ! docker exec peer0.${ORG1_DOMAIN} peer channel list 2>&1 | grep -q "${CHANNEL_NAME}"; then
        join_channel "peer0.${ORG1_DOMAIN}" "${ORG1_NAME}"

        # Update anchor peer
        update_anchor_peer "peer0.${ORG1_DOMAIN}" "${ORG1_NAME}" "${ORG1_DOMAIN}" "${ORG1_NAME}anchors.tx"
    else
        print_status $YELLOW "peer0.${ORG1_DOMAIN} already joined to channel '${CHANNEL_NAME}'"
    fi
fi

if [ "$DEPLOY_ORG2" = true ]; then
    # Check if peer is already joined
    if ! docker exec peer0.${ORG2_DOMAIN} peer channel list 2>&1 | grep -q "${CHANNEL_NAME}"; then
        join_channel "peer0.${ORG2_DOMAIN}" "${ORG2_NAME}"

        # Update anchor peer
        update_anchor_peer "peer0.${ORG2_DOMAIN}" "${ORG2_NAME}" "${ORG2_DOMAIN}" "${ORG2_NAME}anchors.tx"
    else
        print_status $YELLOW "peer0.${ORG2_DOMAIN} already joined to channel '${CHANNEL_NAME}'"
    fi
fi

# Verify channel membership
print_status $YELLOW "=== Verifying Channel Membership ==="

if [ "$DEPLOY_ORG1" = true ]; then
    list_channels "peer0.${ORG1_DOMAIN}" "${ORG1_NAME}"
fi

if [ "$DEPLOY_ORG2" = true ]; then
    list_channels "peer0.${ORG2_DOMAIN}" "${ORG2_NAME}"
fi

# Get channel info
print_status $YELLOW "=== Getting Channel Information ==="

if [ "$DEPLOY_ORG1" = true ]; then
    docker exec -e CORE_PEER_LOCALMSPID="${ORG1_NAME}" \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt \
        peer0.${ORG1_DOMAIN} \
        peer channel getinfo \
        -c ${CHANNEL_NAME} \
        --tls \
        --cafile /etc/hyperledger/fabric/orderer-tls-ca.crt
fi

# Display useful commands
print_status $YELLOW "=== Useful Commands ==="
echo "  List channels from Org1:"
echo "    docker exec peer0.${ORG1_DOMAIN} peer channel list --tls --cafile /etc/hyperledger/fabric/tls/ca.crt"
echo ""
echo "  List channels from Org2:"
echo "    docker exec peer0.${ORG2_DOMAIN} peer channel list --tls --cafile /etc/hyperledger/fabric/tls/ca.crt"
echo ""
echo "  Get channel info:"
echo "    docker exec peer0.${ORG1_DOMAIN} peer channel getinfo -c ${CHANNEL_NAME} --tls --cafile /etc/hyperledger/fabric/tls/ca.crt"
echo ""
echo "  Fetch channel block:"
echo "    docker exec peer0.${ORG1_DOMAIN} peer channel fetch 0 ${CHANNEL_NAME}.block -c ${CHANNEL_NAME} --tls --cafile /etc/hyperledger/fabric/tls/ca.crt"
echo ""
echo "  View channel configuration:"
echo "    docker exec peer0.${ORG1_DOMAIN} peer channel getinfo -c ${CHANNEL_NAME} --tls --cafile /etc/hyperledger/fabric/tls/ca.crt"
echo ""

print_status $GREEN "=== Channel Creation and Join Completed Successfully ==="
print_status $YELLOW "Next step: Run 008-deploy-chaincode.sh"
