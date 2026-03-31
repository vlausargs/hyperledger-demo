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

# Load Fabric environment helper functions
source "${SCRIPT_DIR}/../helpers/fabric-env.sh"

# Set FABRIC_CFG_PATH for peer commands
export FABRIC_CFG_PATH="${FABRIC_CONFIG_PATH}"

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

# # Check if orderer is running
# if ! docker ps | grep -q "orderer1.${ORDERER_DOMAIN}"; then
#     print_status $RED "Error: Orderer is not running. Please run script 005 first."
#     exit 1
# fi

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
    local org_domain=""
    local org_name=""
    local peer_name="peer0"

    if [ "$DEPLOY_ORG1" = true ]; then
        org_domain="${ORG1_DOMAIN}"
        org_name="${ORG1_NAME}"
    elif [ "$DEPLOY_ORG2" = true ]; then
        org_domain="${ORG2_DOMAIN}"
        org_name="${ORG2_NAME}"
    else
        print_status $RED "Error: No organization deployed to create channel"
        exit 1
    fi

    # Set up paths
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls"
    local channel_tx_path="${PROJECT_ROOT}/config/channel-artifacts/${CHANNEL_NAME}.tx"
    local channel_block_path="${PROJECT_ROOT}/config/channel-artifacts/${CHANNEL_NAME}.block"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    # Create the channel using local peer binary
    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    "${FABRIC_BIN_PATH}/peer" channel create \
        -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} \
        -c ${CHANNEL_NAME} \
        -f "${channel_tx_path}" \
        --outputBlock "${channel_block_path}" \
        --tls \
        --cafile "${orderer_tls_ca_path}"

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Channel '${CHANNEL_NAME}' created successfully"
    else
        print_status $RED "✗ Failed to create channel '${CHANNEL_NAME}'"
        exit 1
    fi
}

# Function to join peer to channel
join_channel() {
    local org_domain=$1
    local org_name=$2

    print_status $YELLOW "Joining peer to channel '${CHANNEL_NAME}'..."

    # Set up paths
    local peer_name="peer0"
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls"
    local channel_block_path="${PROJECT_ROOT}/config/channel-artifacts/${CHANNEL_NAME}.block"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    # Join the peer to the channel using local peer binary
    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    "${FABRIC_BIN_PATH}/peer" channel join \
        -b "${channel_block_path}" \
        --tls \
        --cafile "${orderer_tls_ca_path}"

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Peer from ${org_domain} joined channel '${CHANNEL_NAME}' successfully"
    else
        print_status $RED "✗ Failed to join peer from ${org_domain} to channel '${CHANNEL_NAME}'"
        return 1
    fi
}

# Function to update anchor peer
update_anchor_peer() {
    local org_domain=$1
    local org_name=$2
    local org_domain_param=$3
    local anchor_tx_file=$4

    print_status $YELLOW "Updating anchor peer for ${org_name}..."

    # Set up paths
    local peer_name="peer0"
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain_param}/users/admin.${org_domain_param}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain_param}/peers/${peer_name}.${org_domain_param}/tls"
    local anchor_tx_path="${PROJECT_ROOT}/config/channel-artifacts/${anchor_tx_file}"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    # Update anchor peer using local peer binary
    local external_host=$(get_external_host "peer" "${org_domain_param}")
    local peer_port=$(get_peer_port "${org_domain_param}")

    export CORE_PEER_ID="${peer_name}.${org_domain_param}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    "${FABRIC_BIN_PATH}/peer" channel update \
        -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} \
        -c ${CHANNEL_NAME} \
        -f "${anchor_tx_path}" \
        --tls \
        --cafile "${orderer_tls_ca_path}"

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Anchor peer updated for ${org_name}"
    else
        print_status $RED "✗ Failed to update anchor peer for ${org_name}"
        return 1
    fi
}

# Function to list channels
list_channels() {
    local org_domain=$1
    local org_name=$2

    print_status $YELLOW "Listing channels for ${org_domain}..."

    # Set up paths
    local peer_name="peer0"
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    # List channels using local peer binary
    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    "${FABRIC_BIN_PATH}/peer" channel list \
        --tls \
        --cafile "${orderer_tls_ca_path}"
}

# Ensure orderer TLS CA is accessible for peer operations
print_status $YELLOW "Ensuring orderer TLS CA certificate is accessible..."

orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

if [ ! -f "${orderer_tls_ca_path}" ]; then
    print_status $RED "Error: Orderer TLS CA certificate not found at ${orderer_tls_ca_path}"
    exit 1
fi

print_status $GREEN "✓ Orderer TLS CA certificate accessible"




# Create channel if not exists
print_status $YELLOW "Checking if channel '${CHANNEL_NAME}' already exists..."

channel_exists=false

if [ "$DEPLOY_ORG1" = true ]; then
    if "${FABRIC_BIN_PATH}/peer" channel list 2>&1 | grep -q "${CHANNEL_NAME}"; then
        channel_exists=true
    fi
elif [ "$DEPLOY_ORG2" = true ]; then
    if "${FABRIC_BIN_PATH}/peer" channel list 2>&1 | grep -q "${CHANNEL_NAME}"; then
        channel_exists=true
    fi
fi

if [ "$channel_exists" = true ]; then
    print_status $YELLOW "Channel '${CHANNEL_NAME}' already exists. Skipping creation."
else
    create_channel
fi

# Join peers to channel
if [ "$DEPLOY_ORG1" = true ]; then
    # Check if peer is already joined by setting up environment and checking channel list
    export CORE_PEER_ID="peer0.${ORG1_DOMAIN}"
    export CORE_PEER_ADDRESS="${PEER0_ORG1_EXTERNAL_HOST}:${PEER0_ORG1_PORT}"
    export CORE_PEER_LOCALMSPID="${ORG1_NAME}"
    msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp"
    tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls"
    orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    if ! "${FABRIC_BIN_PATH}/peer" channel list 2>&1 | grep -q "${CHANNEL_NAME}"; then
        join_channel "${ORG1_DOMAIN}" "${ORG1_NAME}"

        # Update anchor peer
        update_anchor_peer "${ORG1_DOMAIN}" "${ORG1_NAME}" "${ORG1_DOMAIN}" "${ORG1_NAME}anchors.tx"
    else
        print_status $YELLOW "Peer from ${ORG1_DOMAIN} already joined to channel '${CHANNEL_NAME}'"
    fi
fi

if [ "$DEPLOY_ORG2" = true ]; then
    # Check if peer is already joined by setting up environment and checking channel list
    export CORE_PEER_ID="peer0.${ORG2_DOMAIN}"
    export CORE_PEER_ADDRESS="${PEER0_ORG2_EXTERNAL_HOST}:${PEER0_ORG2_PORT}"
    export CORE_PEER_LOCALMSPID="${ORG2_NAME}"
    msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/users/admin.${ORG2_DOMAIN}/msp"
    tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls"
    orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    if ! "${FABRIC_BIN_PATH}/peer" channel list 2>&1 | grep -q "${CHANNEL_NAME}"; then
        join_channel "${ORG2_DOMAIN}" "${ORG2_NAME}"

        # Update anchor peer
        update_anchor_peer "${ORG2_DOMAIN}" "${ORG2_NAME}" "${ORG2_DOMAIN}" "${ORG2_NAME}anchors.tx"
    else
        print_status $YELLOW "Peer from ${ORG2_DOMAIN} already joined to channel '${CHANNEL_NAME}'"
    fi
fi

# Verify channel membership
print_status $YELLOW "=== Verifying Channel Membership ==="

if [ "$DEPLOY_ORG1" = true ]; then
    list_channels "${ORG1_DOMAIN}" "${ORG1_NAME}"
fi

if [ "$DEPLOY_ORG2" = true ]; then
    list_channels "${ORG2_DOMAIN}" "${ORG2_NAME}"
fi

# Get channel info
print_status $YELLOW "=== Getting Channel Information ==="

if [ "$DEPLOY_ORG1" = true ]; then
    # Set up paths
    msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp"
    tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls"
    orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    export CORE_PEER_ID="peer0.${ORG1_DOMAIN}"
    export CORE_PEER_ADDRESS="${PEER0_ORG1_EXTERNAL_HOST}:${PEER0_ORG1_PORT}"
    export CORE_PEER_LOCALMSPID="${ORG1_NAME}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    "${FABRIC_BIN_PATH}/peer" channel getinfo \
        -c ${CHANNEL_NAME} \
        --tls \
        --cafile "${orderer_tls_ca_path}"
fi

# Display useful commands
print_status $YELLOW "=== Useful Commands ==="
echo "  List channels from Org1:"
echo "    export CORE_PEER_LOCALMSPID='${ORG1_NAME}'"
echo "    export CORE_PEER_TLS_ROOTCERT_FILE='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt'"
echo "    export CORE_PEER_MSPCONFIGPATH='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp'"
echo "    ${FABRIC_BIN_PATH}/peer channel list --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt'"
echo ""
echo "  List channels from Org2:"
echo "    export CORE_PEER_LOCALMSPID='${ORG2_NAME}'"
echo "    export CORE_PEER_TLS_ROOTCERT_FILE='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/ca.crt'"
echo "    export CORE_PEER_MSPCONFIGPATH='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/users/admin.${ORG2_DOMAIN}/msp'"
echo "    ${FABRIC_BIN_PATH}/peer channel list --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt'"
echo ""
echo "  Get channel info:"
echo "    ${FABRIC_BIN_PATH}/peer channel getinfo -c ${CHANNEL_NAME} --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt'"
echo ""
echo "  Fetch channel block:"
echo "    ${FABRIC_BIN_PATH}/peer channel fetch 0 ${CHANNEL_NAME}.block -c ${CHANNEL_NAME} --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt'"
echo ""
echo "  View channel configuration:"
echo "    ${FABRIC_BIN_PATH}/peer channel getinfo -c ${CHANNEL_NAME} --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt'"
echo ""

print_status $GREEN "=== Channel Creation and Join Completed Successfully ==="
print_status $YELLOW "Next step: Run 008-deploy-chaincode.sh"
