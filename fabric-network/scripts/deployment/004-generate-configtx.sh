#!/bin/bash

# Script 004: Generate Channel Configuration
# This script generates the channel configuration using configtxgen

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

print_status $GREEN "=== Starting Channel Configuration Generation ==="
print_status $YELLOW "Deployment Mode:"
echo "  Orderer: $DEPLOY_ORDERER"
echo "  Org1: $DEPLOY_ORG1"
echo "  Org2: $DEPLOY_ORG2"
echo ""
print_status $YELLOW "Orderer Type: $ORDERER_TYPE"
echo ""

# Create config directory
mkdir -p "${PROJECT_ROOT}/config/channel-artifacts"

# For etcdraft orderer type, we need TLS certificates in the configtx.yaml
ORDERER_TLS_CERT=""
ORDERER_TLS_KEY=""

if [ "$DEPLOY_ORDERER" = true ] && [ "$ORDERER_TYPE" = "etcdraft" ]; then
    ORDERER_TLS_DIR="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls"

    # If TLS directory doesn't exist, create it and copy MSP certificate as workaround
    if [ ! -d "$ORDERER_TLS_DIR" ]; then
        print_status $YELLOW "TLS certificates not found for etcdraft, creating minimal TLS configuration..."
        mkdir -p "$ORDERER_TLS_DIR"

        # Copy MSP certificate as a temporary workaround for TLS (development only)
        cp "${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/msp/signcerts/cert.pem" \
           "${ORDERER_TLS_DIR}/server.crt"
        cp "${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/msp/keystore/"* \
           "${ORDERER_TLS_DIR}/server.key" 2>/dev/null || true
        cp "${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/msp/cacerts/"*.pem \
           "${ORDERER_TLS_DIR}/ca.crt" 2>/dev/null || true

        print_status $GREEN "✓ Minimal TLS configuration created for development"
    fi

    # Get TLS certificate file paths for etcdraft configuration
    print_status $YELLOW "Preparing TLS certificate paths for etcdraft consensus..."
    ORDERER_TLS_CERT_PATH="${ORDERER_TLS_DIR}/server.crt"
    print_status $GREEN "✓ TLS certificate paths configured"
    echo ""
fi

# Create configtx.yaml file
print_status $YELLOW "Creating configtx.yaml configuration file..."

CONFIGTX_FILE="${PROJECT_ROOT}/config/channel-artifacts/configtx.yaml"

cat > "$CONFIGTX_FILE" << EOF
# ---------------------------------------------------------------------------
#   Copyright IBM Corp. 2014, 2017 All Rights Reserved
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
# ---------------------------------------------------------------------------

# ---------------------------------------------------------------------------
#   Section: Organizations
# ---------------------------------------------------------------------------
Organizations:

    # ---------------------------------------------------------------------------
    #   Orderer Org
    # ---------------------------------------------------------------------------
    - &OrdererOrg
        Name: OrdererMSP
        ID: OrdererMSP
        MSPDir: ${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/msp
        Policies:
            Readers:
                Type: Signature
                Rule: "OR('OrdererMSP.member')"
            Writers:
                Type: Signature
                Rule: "OR('OrdererMSP.member')"
            Admins:
                Type: Signature
                Rule: "OR('OrdererMSP.admin')"
        OrdererEndpoints:
            - ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT}

    # ---------------------------------------------------------------------------
    #   Peer Organization 1
    # ---------------------------------------------------------------------------
    - &Org1
        Name: ${ORG1_NAME}
        ID: ${ORG1_NAME}
        MSPDir: ${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/msp
        Policies:
            Readers:
                Type: Signature
                Rule: "OR('${ORG1_NAME}.admin', '${ORG1_NAME}.peer', '${ORG1_NAME}.client')"
            Writers:
                Type: Signature
                Rule: "OR('${ORG1_NAME}.admin', '${ORG1_NAME}.client')"
            Admins:
                Type: Signature
                Rule: "OR('${ORG1_NAME}.admin')"
            Endorsement:
                Type: Signature
                Rule: "OR('${ORG1_NAME}.member')"
        AnchorPeers:
            - Host: ${PEER0_ORG1_EXTERNAL_HOST}
              Port: ${PEER0_ORG1_PORT}

    # ---------------------------------------------------------------------------
    #   Peer Organization 2
    # ---------------------------------------------------------------------------
    - &Org2
        Name: ${ORG2_NAME}
        ID: ${ORG2_NAME}
        MSPDir: ${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/msp
        Policies:
            Readers:
                Type: Signature
                Rule: "OR('${ORG2_NAME}.admin', '${ORG2_NAME}.peer', '${ORG2_NAME}.client')"
            Writers:
                Type: Signature
                Rule: "OR('${ORG2_NAME}.admin', '${ORG2_NAME}.client')"
            Admins:
                Type: Signature
                Rule: "OR('${ORG2_NAME}.admin')"
            Endorsement:
                Type: Signature
                Rule: "OR('${ORG2_NAME}.member')"
        AnchorPeers:
            - Host: ${PEER0_ORG2_EXTERNAL_HOST}
              Port: ${PEER0_ORG2_PORT}

# ---------------------------------------------------------------------------
#   Capabilities
# ---------------------------------------------------------------------------
Capabilities:
    Channel: &ChannelCapabilities
        V2_0: true
    Orderer: &OrdererCapabilities
        V2_0: true
    Application: &ApplicationCapabilities
        V2_0: true

# ---------------------------------------------------------------------------
#   Application
# ---------------------------------------------------------------------------
Application: &ApplicationDefaults
    Organizations:
    Policies:
        Readers:
            Type: ImplicitMeta
            Rule: "ANY Readers"
        Writers:
            Type: ImplicitMeta
            Rule: "ANY Writers"
        Admins:
            Type: ImplicitMeta
            Rule: "MAJORITY Admins"
        LifecycleEndorsement:
            Type: ImplicitMeta
            Rule: "MAJORITY Endorsement"
        Endorsement:
            Type: ImplicitMeta
            Rule: "MAJORITY Endorsement"
    Capabilities:
        <<: *ApplicationCapabilities

# ---------------------------------------------------------------------------
#   Orderer
# ---------------------------------------------------------------------------
Orderer: &OrdererDefaults
    OrdererType: ${ORDERER_TYPE}
    Addresses:
        - ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT}
    BatchTimeout: 2s
    BatchSize:
        MaxMessageCount: 10
        AbsoluteMaxBytes: 99 MB
        PreferredMaxBytes: 512 KB
    Organizations:
EOF

# Add EtcdRaft section for etcdraft
if [ "$DEPLOY_ORDERER" = true ] && [ "$ORDERER_TYPE" = "etcdraft" ]; then
    cat >> "$CONFIGTX_FILE" << EOF
    EtcdRaft:
        Consenters:
            - Host: ${ORDERER_EXTERNAL_HOST}
              Port: ${ORDERER_PORT}
              ClientTLSCert: ${ORDERER_TLS_CERT_PATH}
              ServerTLSCert: ${ORDERER_TLS_CERT_PATH}
        Options:
            TickInterval: 500ms
            ElectionTick: 10
            HeartbeatTick: 1
            MaxInflightBlocks: 5
            SnapshotIntervalSize: 16 MB
EOF
fi

# Add Policies section
cat >> "$CONFIGTX_FILE" << EOF
    Policies:
        Readers:
            Type: ImplicitMeta
            Rule: "ANY Readers"
        Writers:
            Type: ImplicitMeta
            Rule: "ANY Writers"
        Admins:
            Type: ImplicitMeta
            Rule: "MAJORITY Admins"
        BlockValidation:
            Type: ImplicitMeta
            Rule: "ANY Writers"
    Capabilities:
        <<: *OrdererCapabilities

# ---------------------------------------------------------------------------
#   Channel
# ---------------------------------------------------------------------------
Channel: &ChannelDefaults
    Policies:
        Readers:
            Type: ImplicitMeta
            Rule: "ANY Readers"
        Writers:
            Type: ImplicitMeta
            Rule: "ANY Writers"
        Admins:
            Type: ImplicitMeta
            Rule: "MAJORITY Admins"
    Capabilities:
        <<: *ChannelCapabilities

# ---------------------------------------------------------------------------
#   Profiles
# ---------------------------------------------------------------------------
Profiles:

    # ---------------------------------------------------------------------------
    #   TwoOrgsOrdererGenesis
    # ---------------------------------------------------------------------------
    TwoOrgsOrdererGenesis:
        <<: *ChannelDefaults
        Orderer:
            <<: *OrdererDefaults
            Organizations:
                - *OrdererOrg
        Consortiums:
            SampleConsortium:
                Organizations:
                    - *Org1
                    - *Org2

    # ---------------------------------------------------------------------------
    #   TwoOrgsChannel
    # ---------------------------------------------------------------------------
    TwoOrgsChannel:
        Consortium: SampleConsortium
        <<: *ChannelDefaults
        Application:
            <<: *ApplicationDefaults
            Organizations:
                - *Org1
                - *Org2
EOF

print_status $GREEN "✓ configtx.yaml created successfully"

# Set FABRIC_CFG_PATH to point to the config directory
export FABRIC_CFG_PATH="${PROJECT_ROOT}/config/channel-artifacts"

# Generate genesis block for the orderer
if [ "$DEPLOY_ORDERER" = true ]; then
    print_status $YELLOW "Generating genesis block for orderer..."

    configtxgen \
        -profile ${ORDERER_GENESIS_PROFILE} \
        -channelID system-channel \
        -outputBlock "${PROJECT_ROOT}/config/channel-artifacts/genesis.block" \
        -configPath "${PROJECT_ROOT}/config/channel-artifacts"

    if [ -f "${PROJECT_ROOT}/config/channel-artifacts/genesis.block" ]; then
        print_status $GREEN "✓ Genesis block generated successfully"
    else
        print_status $RED "✗ Failed to generate genesis block"
        exit 1
    fi
fi

# Generate channel configuration transaction
if [ "$DEPLOY_ORG1" = true ] || [ "$DEPLOY_ORG2" = true ]; then
    print_status $YELLOW "Generating channel configuration transaction..."

    configtxgen \
        -profile ${CHANNEL_PROFILE} \
        -channelID ${CHANNEL_NAME} \
        -outputCreateChannelTx "${PROJECT_ROOT}/config/channel-artifacts/${CHANNEL_NAME}.tx" \
        -configPath "${PROJECT_ROOT}/config/channel-artifacts"

    if [ -f "${PROJECT_ROOT}/config/channel-artifacts/${CHANNEL_NAME}.tx" ]; then
        print_status $GREEN "✓ Channel configuration transaction generated successfully"
    else
        print_status $RED "✗ Failed to generate channel configuration transaction"
        exit 1
    fi
fi

# Generate anchor peer update transactions for organizations
if [ "$DEPLOY_ORG1" = true ]; then
    print_status $YELLOW "Generating anchor peer update for Org1..."

    configtxgen \
        -profile ${CHANNEL_PROFILE} \
        -channelID ${CHANNEL_NAME} \
        -outputAnchorPeersUpdate "${PROJECT_ROOT}/config/channel-artifacts/${ORG1_NAME}anchors.tx" \
        -asOrg ${ORG1_NAME} \
        -configPath "${PROJECT_ROOT}/config/channel-artifacts"

    if [ -f "${PROJECT_ROOT}/config/channel-artifacts/${ORG1_NAME}anchors.tx" ]; then
        print_status $GREEN "✓ Org1 anchor peer update generated successfully"
    else
        print_status $RED "✗ Failed to generate Org1 anchor peer update"
        exit 1
    fi
fi

if [ "$DEPLOY_ORG2" = true ]; then
    print_status $YELLOW "Generating anchor peer update for Org2..."

    configtxgen \
        -profile ${CHANNEL_PROFILE} \
        -channelID ${CHANNEL_NAME} \
        -outputAnchorPeersUpdate "${PROJECT_ROOT}/config/channel-artifacts/${ORG2_NAME}anchors.tx" \
        -asOrg ${ORG2_NAME} \
        -configPath "${PROJECT_ROOT}/config/channel-artifacts"

    if [ -f "${PROJECT_ROOT}/config/channel-artifacts/${ORG2_NAME}anchors.tx" ]; then
        print_status $GREEN "✓ Org2 anchor peer update generated successfully"
    else
        print_status $RED "✗ Failed to generate Org2 anchor peer update"
        exit 1
    fi
fi

# List generated files
print_status $YELLOW "Generated channel artifacts:"
ls -lh "${PROJECT_ROOT}/config/channel-artifacts/"

print_status $GREEN "=== Channel Configuration Generation Completed Successfully ==="
print_status $YELLOW "Next step: Run 005-deploy-orderer.sh"
