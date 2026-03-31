
# Script 008: Deploy Chaincode
# This script packages, approves, and commits the chaincode to the network

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
    echo -e "${color}${message}${NC}" >&2
}

# Load environment variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

if [ ! -f "${PROJECT_ROOT}/.env" ]; then
    print_status $RED "Error: .env file not found at ${PROJECT_ROOT}/.env"
    exit 1
fi

source "${PROJECT_ROOT}/.env"

# Source fabric-env.sh helper script for local binary usage
source "${PROJECT_ROOT}/fabric-network/scripts/helpers/fabric-env.sh"

# Set FABRIC_CFG_PATH for peer commands
export FABRIC_CFG_PATH="${FABRIC_CONFIG_PATH}"

# Verify fabric binaries are available
verify_binaries

# Check deployment flags
DEPLOY_ORG1=${DEPLOY_ORG1:-true}
DEPLOY_ORG2=${DEPLOY_ORG2:-true}

print_status $GREEN "=== Starting Chaincode Deployment ==="
print_status $YELLOW "Deployment Mode:"
echo "  Org1: $DEPLOY_ORG1"
echo "  Org2: $DEPLOY_ORG2"
echo ""

# Verify prerequisites
print_status $YELLOW "Verifying prerequisites..."

# Check if chaincode directory exists
if [ ! -d "${PROJECT_ROOT}/fabric-network/chaincode/basic" ]; then
    print_status $RED "Error: Chaincode directory not found at ${PROJECT_ROOT}/fabric-network/chaincode/basic"
    exit 1
fi

# Check if peers are running
if [ "$DEPLOY_ORG1" = true ]; then
    if ! docker ps | grep -q "peer0.${ORG1_DOMAIN}"; then
        print_status $RED "Error: Org1 peer is not running. Please run script 006 first."
        exit 1
    fi
fi

if [ "$DEPLOY_ORG2" = true ]; then
    if ! docker ps | grep -q "peer0.${ORG2_DOMAIN}"; then
        print_status $RED "Error: Org2 peer is not running. Please run script 006 first."
        exit 1
    fi
fi

print_status $GREEN "✓ Prerequisites verified"

# Function to package chaincode on host
package_chaincode_host() {
    print_status $YELLOW "Packaging chaincode on host..."

    # Create temporary directory for packaging
    local temp_dir="${PROJECT_ROOT}/fabric-network/temp-package"
    mkdir -p "${temp_dir}"

    # Copy chaincode to temp directory with proper structure
    cp -r "${PROJECT_ROOT}/fabric-network/chaincode/basic" "${temp_dir}/basic"

    # Use a temporary container with fabric-tools to package the chaincode
    docker run --rm \
        -v "${temp_dir}:/chaincode" \
        -v "${PROJECT_ROOT}/config/channel-artifacts:/output" \
        -w /chaincode \
        hyperledger/fabric-tools:${FABRIC_VERSION} \
        peer lifecycle chaincode package \
        /output/${CHAINCODE_NAME}.tar.gz \
        --path /chaincode/basic \
        --lang ${CHAINCODE_LANGUAGE} \
        --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Chaincode packaged successfully on host"

        # Store the package file path in a global variable
        PACKAGE_FILE="${PROJECT_ROOT}/config/channel-artifacts/${CHAINCODE_NAME}.tar.gz"

        # Fix file permissions and ownership (Docker creates files as root)
        sudo chown $(id -u):$(id -g) "${PACKAGE_FILE}"
        sudo chmod 644 "${PACKAGE_FILE}"

        # Clean up temp directory
        rm -rf "${temp_dir}"
        return 0
    else
        print_status $RED "✗ Failed to package chaincode on host"

        # Clean up temp directory
        rm -rf "${temp_dir}"

        return 1
    fi
}

# Function to install chaincode
install_chaincode() {
    local org_domain=$1
    local org_name=$2
    local package_file=$3

    print_status $YELLOW "Installing chaincode for ${org_name}..."

    # Set up environment variables for local peer binary
    local peer_name="peer0"
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"
    local local_package_file="${PROJECT_ROOT}/config/channel-artifacts/${CHAINCODE_NAME}.tar.gz"

    # Install chaincode using local peer binary
    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    "${FABRIC_BIN_PATH}/peer" lifecycle chaincode install \
        "${local_package_file}" \
        --tls \
        --cafile "${orderer_tls_ca_path}"

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Chaincode installed successfully for ${org_name}"

        # Get the package ID
        local package_id=$("${FABRIC_BIN_PATH}/peer" lifecycle chaincode queryinstalled \
            --tls \
            --cafile "${orderer_tls_ca_path}" \
            --output json | jq -r ".installed_chaincodes[] | select(.label==\"${CHAINCODE_NAME}_${CHAINCODE_VERSION}\") | .package_id")

        if [ -n "$package_id" ]; then
            print_status $GREEN "✓ Package ID: $package_id"
            echo "$package_id"
        else
            print_status $RED "✗ Failed to get package ID"
            return 1
        fi
    else
        print_status $RED "✗ Failed to install chaincode for ${org_name}"
        return 1
    fi
}

# Function to approve chaincode
approve_chaincode() {
    local org_domain=$1
    local org_name=$2
    local package_id=$3

    print_status $YELLOW "Approving chaincode for ${org_name}..."

    # Set up environment variables for local peer binary
    local peer_name="peer0"
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    # Query to check if already approved
    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    local approved=$("${FABRIC_BIN_PATH}/peer" lifecycle chaincode queryapproved \
        -C ${CHANNEL_NAME} \
        -n ${CHAINCODE_NAME} \
        --tls \
        --cafile "${orderer_tls_ca_path}" 2>&1 || echo "not_approved")

    if echo "$approved" | grep -Fq "${package_id}"; then
        print_status $YELLOW "Chaincode already approved for ${org_name}"
        return 0
    fi

    # Approve chaincode
    "${FABRIC_BIN_PATH}/peer" lifecycle chaincode approveformyorg \
        -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} \
        --channelID ${CHANNEL_NAME} \
        --name ${CHAINCODE_NAME} \
        --version ${CHAINCODE_VERSION} \
        --package-id "${package_id}" \
        --sequence ${CHAINCODE_SEQUENCE} \
        --tls \
        --cafile "${orderer_tls_ca_path}" \
        --signature-policy "${CHAINCODE_ENDORSEMENT_POLICY}"

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Chaincode approved successfully for ${org_name}"
    else
        print_status $RED "✗ Failed to approve chaincode for ${org_name}"
        return 1
    fi
}

# Function to check commit readiness
check_commit_readiness() {
    print_status $YELLOW "Checking commit readiness..."

    local org_domain
    local org_name

    if [ "$DEPLOY_ORG1" = true ]; then
        org_domain="${ORG1_DOMAIN}"
        org_name="${ORG1_NAME}"
    elif [ "$DEPLOY_ORG2" = true ]; then
        org_domain="${ORG2_DOMAIN}"
        org_name="${ORG2_NAME}"
    else
        print_status $RED "Error: No organization deployed to check commit readiness"
        return 1
    fi

    # Set up environment variables for local peer binary
    local peer_name="peer0"
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    "${FABRIC_BIN_PATH}/peer" lifecycle chaincode checkcommitreadiness \
        --channelID ${CHANNEL_NAME} \
        --name ${CHAINCODE_NAME} \
        --version ${CHAINCODE_VERSION} \
        --sequence ${CHAINCODE_SEQUENCE} \
        --tls \
        --cafile "${orderer_tls_ca_path}" \
        --output json | jq '.'
}

# Function to commit chaincode
commit_chaincode() {
    print_status $YELLOW "Committing chaincode to channel ${CHANNEL_NAME}..."

    local org_domain
    local org_name
    local org1_tls_cert=""
    local org2_tls_cert=""

    # Determine which peer to use for committing and set up TLS cert paths
    if [ "$DEPLOY_ORG1" = true ]; then
        org_domain="${ORG1_DOMAIN}"
        org_name="${ORG1_NAME}"

        # Use local paths for TLS CA certificates
        org1_tls_cert="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt"

        if [ "$DEPLOY_ORG2" = true ]; then
            org2_tls_cert="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/ca.crt"
        fi
    elif [ "$DEPLOY_ORG2" = true ]; then
        org_domain="${ORG2_DOMAIN}"
        org_name="${ORG2_NAME}"

        # Use local paths for TLS CA certificates
        org2_tls_cert="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/ca.crt"

        if [ "$DEPLOY_ORG1" = true ]; then
            org1_tls_cert="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt"
        fi
    else
        print_status $RED "Error: No organization deployed to commit chaincode"
        return 1
    fi

    # Set up environment variables for local peer binary
    local peer_name="peer0"
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    # Build commit command with appropriate peer addresses
    local commit_cmd="${FABRIC_BIN_PATH}/peer lifecycle chaincode commit \
        -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} \
        --channelID ${CHANNEL_NAME} \
        --name ${CHAINCODE_NAME} \
        --version ${CHAINCODE_VERSION} \
        --sequence ${CHAINCODE_SEQUENCE} \
        --tls \
        --cafile \"${orderer_tls_ca_path}\" \
        --signature-policy \"${CHAINCODE_ENDORSEMENT_POLICY}\""

    if [ "$DEPLOY_ORG1" = true ]; then
        commit_cmd="${commit_cmd} --peerAddresses peer0.${ORG1_DOMAIN}:${PEER0_ORG1_PORT}"
        commit_cmd="${commit_cmd} --tlsRootCertFiles \"${org1_tls_cert}\""
    fi

    if [ "$DEPLOY_ORG2" = true ]; then
        commit_cmd="${commit_cmd} --peerAddresses peer0.${ORG2_DOMAIN}:${PEER0_ORG2_PORT}"
        commit_cmd="${commit_cmd} --tlsRootCertFiles \"${org2_tls_cert}\""
    fi

    eval "$commit_cmd"

    if [ $? -eq 0 ]; then
        print_status $GREEN "✓ Chaincode committed successfully to channel ${CHANNEL_NAME}"
    else
        print_status $RED "✗ Failed to commit chaincode to channel ${CHANNEL_NAME}"
        return 1
    fi
}

# Function to query committed chaincode
query_committed() {
    print_status $YELLOW "Querying committed chaincode..."

    local org_domain
    local org_name

    if [ "$DEPLOY_ORG1" = true ]; then
        org_domain="${ORG1_DOMAIN}"
        org_name="${ORG1_NAME}"
    elif [ "$DEPLOY_ORG2" = true ]; then
        org_domain="${ORG2_DOMAIN}"
        org_name="${ORG2_NAME}"
    else
        print_status $RED "Error: No organization deployed to query committed chaincode"
        return 1
    fi

    # Set up environment variables for local peer binary
    local peer_name="peer0"
    local msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    local tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls"
    local orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"

    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    "${FABRIC_BIN_PATH}/peer" lifecycle chaincode querycommitted \
        --channelID ${CHANNEL_NAME} \
        --name ${CHAINCODE_NAME} \
        --tls \
        --cafile "${orderer_tls_ca_path}" \
        --output json | jq '.'
}

# Package chaincode on host first
print_status $YELLOW "=== Packaging Chaincode ==="
PACKAGE_FILE=""

package_chaincode_host

if [ $? -ne 0 ] || [ -z "$PACKAGE_FILE" ]; then
    print_status $RED "Error: Failed to package chaincode"
    exit 1
fi

print_status $GREEN "✓ Package file: $PACKAGE_FILE"

# Chaincode package is already in local channel-artifacts directory
print_status $GREEN "✓ Chaincode package available locally at ${PACKAGE_FILE}"
print_status $YELLOW "Note: No need to copy chaincode package to containers - using local binary"

# Install chaincode on peers
print_status $YELLOW "=== Installing Chaincode on Peers ==="
PACKAGE_IDS=()

if [ "$DEPLOY_ORG1" = true ]; then
    pkg_id=$(install_chaincode "${ORG1_DOMAIN}" "${ORG1_NAME}" "${CHAINCODE_NAME}.tar.gz")
    if [ $? -eq 0 ] && [ -n "$pkg_id" ]; then
        PACKAGE_IDS+=("$pkg_id")
    fi
fi

if [ "$DEPLOY_ORG2" = true ]; then
    pkg_id=$(install_chaincode "${ORG2_DOMAIN}" "${ORG2_NAME}" "${CHAINCODE_NAME}.tar.gz")
    if [ $? -eq 0 ] && [ -n "$pkg_id" ]; then
        PACKAGE_IDS+=("$pkg_id")
    fi
fi

# Use the first package ID for approval
if [ ${#PACKAGE_IDS[@]} -gt 0 ]; then
    PACKAGE_ID="${PACKAGE_IDS[0]}"
fi

if [ -z "$PACKAGE_ID" ]; then
    print_status $RED "Error: Failed to get package ID"
    exit 1
fi
print_status $YELLOW "deployed chaincode with PACKAGE_ID ${PACKAGE_ID}"

# Approve chaincode
if [ "$DEPLOY_ORG1" = true ]; then
    approve_chaincode "${ORG1_DOMAIN}" "${ORG1_NAME}" "$PACKAGE_ID"
fi

if [ "$DEPLOY_ORG2" = true ]; then
    approve_chaincode "${ORG2_DOMAIN}" "${ORG2_NAME}" "$PACKAGE_ID"
fi

check_commit_readiness
# Wait for approvals to propagate
print_status $YELLOW "Waiting for approvals to propagate..."
sleep 5
check_commit_readiness
# Commit chaincode
commit_chaincode

# Query committed chaincode
query_committed

# Initialize chaincode (if needed)
print_status $YELLOW "Initializing chaincode..."

if [ "$DEPLOY_ORG1" = true ]; then
    # Set up environment variables for local peer binary
    peer_name="peer0"
    msp_path="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp"
    tls_path="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/${peer_name}.${ORG1_DOMAIN}/tls"
    orderer_tls_ca_path="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt"
    org1_tls_cert="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt"
    org2_tls_cert=""

    if [ "$DEPLOY_ORG2" = true ]; then
        org2_tls_cert="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/ca.crt"
    fi

    export CORE_PEER_ID="peer0.${ORG1_DOMAIN}"
    export CORE_PEER_ADDRESS="${PEER0_ORG1_EXTERNAL_HOST}:${PEER0_ORG1_PORT}"
    export CORE_PEER_LOCALMSPID="${ORG1_NAME}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${tls_path}/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${msp_path}"

    # Check if chaincode is already initialized
    initialized=$("${FABRIC_BIN_PATH}/peer" chaincode query \
        -C ${CHANNEL_NAME} \
        -n ${CHAINCODE_NAME} \
        -c '{"Args":["GetAllAssets"]}' \
        --tls \
        --cafile "${orderer_tls_ca_path}" 2>&1 || echo "not_initialized")

    if ! echo "$initialized" | grep -q "Error"; then
        print_status $YELLOW "Chaincode appears to be already initialized"
    else
        print_status $YELLOW "Initializing chaincode ledger..."

        # Build invoke command with appropriate peer addresses
        local invoke_cmd="${FABRIC_BIN_PATH}/peer chaincode invoke \
            -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} \
            -C ${CHANNEL_NAME} \
            -n ${CHAINCODE_NAME} \
            -c '{\"Args\":[\"InitLedger\"]}' \
            --tls \
            --cafile \"${orderer_tls_ca_path}\" \
            --peerAddresses peer0.${ORG1_DOMAIN}:${PEER0_ORG1_PORT} \
            --tlsRootCertFiles \"${org1_tls_cert}\""

        if [ "$DEPLOY_ORG2" = true ]; then
            invoke_cmd="${invoke_cmd} --peerAddresses peer0.${ORG2_DOMAIN}:${PEER0_ORG2_PORT}"
            invoke_cmd="${invoke_cmd} --tlsRootCertFiles \"${org2_tls_cert}\""
        fi

        eval "$invoke_cmd"

        if [ $? -eq 0 ]; then
            print_status $GREEN "✓ Chaincode initialized successfully"
        else
            print_status $YELLOW "Chaincode initialization might have failed (this is normal if already initialized)"
        fi
    fi
fi

# Display useful commands
print_status $YELLOW "=== Useful Commands ==="
echo ""
echo "  Query all assets:"
echo "    export CORE_PEER_LOCALMSPID='${ORG1_NAME}'"
echo "    export CORE_PEER_TLS_ROOTCERT_FILE='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt'"
echo "    export CORE_PEER_MSPCONFIGPATH='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp'"
echo "    ${FABRIC_BIN_PATH}/peer chaincode query -C ${CHANNEL_NAME} -n ${CHAINCODE_NAME} -c '{\"Args\":[\"GetAllAssets\"]}' --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt'"
echo ""
echo "  Create asset:"
echo "    export CORE_PEER_LOCALMSPID='${ORG1_NAME}'"
echo "    export CORE_PEER_TLS_ROOTCERT_FILE='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt'"
echo "    export CORE_PEER_MSPCONFIGPATH='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp'"
echo "    ${FABRIC_BIN_PATH}/peer chaincode invoke -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} -C ${CHANNEL_NAME} -n ${CHAINCODE_NAME} -c '{\"Args\":[\"CreateAsset\",\"asset7\",\"purple\",20,\"Owner\",800]}' --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt' --peerAddresses peer0.${ORG1_DOMAIN}:${PEER0_ORG1_PORT} --tlsRootCertFiles '${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt' --peerAddresses peer0.${ORG2_DOMAIN}:${PEER0_ORG2_PORT} --tlsRootCertFiles '${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/ca.crt'"
echo ""
echo "  Read asset:"
echo "    export CORE_PEER_LOCALMSPID='${ORG1_NAME}'"
echo "    export CORE_PEER_TLS_ROOTCERT_FILE='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt'"
echo "    export CORE_PEER_MSPCONFIGPATH='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp'"
echo "    ${FABRIC_BIN_PATH}/peer chaincode query -C ${CHANNEL_NAME} -n ${CHAINCODE_NAME} -c '{\"Args\":[\"ReadAsset\",\"asset1\"]}' --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt'"
echo ""
echo "  Transfer asset:"
echo "    export CORE_PEER_LOCALMSPID='${ORG1_NAME}'"
echo "    export CORE_PEER_TLS_ROOTCERT_FILE='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt'"
echo "    export CORE_PEER_MSPCONFIGPATH='${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users/admin.${ORG1_DOMAIN}/msp'"
echo "    ${FABRIC_BIN_PATH}/peer chaincode invoke -o ${ORDERER_EXTERNAL_HOST}:${ORDERER_PORT} -C ${CHANNEL_NAME} -n ${CHAINCODE_NAME} -c '{\"Args\":[\"TransferAsset\",\"asset1\",\"NewOwner\"]}' --tls --cafile '${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/ca.crt' --peerAddresses peer0.${ORG1_DOMAIN}:${PEER0_ORG1_PORT} --tlsRootCertFiles '${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/ca.crt' --peerAddresses peer0.${ORG2_DOMAIN}:${PEER0_ORG2_PORT} --tlsRootCertFiles '${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/ca.crt'"
echo ""

print_status $GREEN "=== Chaincode Deployment Completed Successfully ==="
print_status $YELLOW "Next step: Run 009-start-client.sh"
