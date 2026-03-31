#!/bin/bash

# Script 003: Setup CA - Register and Enroll Identities
# This script registers and enrolls identities for the orderer and organizations

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

# Load Fabric environment helper functions
source "${SCRIPT_DIR}/../helpers/fabric-env.sh"

if [ ! -f "${PROJECT_ROOT}/.env" ]; then
    print_status $RED "Error: .env file not found at ${PROJECT_ROOT}/.env"
    exit 1
fi

source "${PROJECT_ROOT}/.env"

# Check deployment flags
DEPLOY_ORDERER=${DEPLOY_ORDERER:-true}
DEPLOY_ORG1=${DEPLOY_ORG1:-true}
DEPLOY_ORG2=${DEPLOY_ORG2:-true}

print_status $GREEN "=== Starting CA Identity Setup ==="
print_status $YELLOW "Deployment Mode:"
echo "  Orderer: $DEPLOY_ORDERER"
echo "  Org1: $DEPLOY_ORG1"
echo "  Org2: $DEPLOY_ORG2"
echo ""

# Create organizations directory structure
mkdir -p "${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers"
mkdir -p "${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/ca"
mkdir -p "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers"
mkdir -p "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/users"
mkdir -p "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/ca"
mkdir -p "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers"
mkdir -p "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/users"
mkdir -p "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/ca"

# Function to get CA certificate
get_ca_cert() {
    local ca_name=$1
    local ca_url=$2
    local org_type=$3
    local org_domain=$4

    local ca_dir="${PROJECT_ROOT}/organizations/${org_type}Organizations/${org_domain}/ca"

    print_status $YELLOW "Getting CA certificate for ${ca_name}"

    # Set FABRIC_CA_CLIENT_HOME to the CA directory
    export FABRIC_CA_CLIENT_HOME="${ca_dir}"

    # Get CA certificate
    fabric-ca-client getcacert -u "http://${ca_url}"

    print_status $GREEN "✓ CA certificate obtained for ${ca_name}"
}

# Function to enroll bootstrap admin
enroll_bootstrap_admin() {
    local ca_name=$1
    local ca_url=$2
    local bootstrap_admin_id=$3
    local bootstrap_admin_secret=$4
    local org_type=$5
    local org_domain=$6

    local admin_dir="${PROJECT_ROOT}/organizations/${org_type}Organizations/${org_domain}/users/bootstrap-admin.${org_domain}"
    local msp_dir="${admin_dir}/msp"

    print_status $YELLOW "Enrolling bootstrap admin: ${bootstrap_admin_id}"

    # Clean up existing enrollment if it exists
    rm -rf "${admin_dir}"

    # Enroll the bootstrap admin
    FABRIC_CA_CLIENT_HOME="${admin_dir}" \
    fabric-ca-client enroll \
        -u "http://${bootstrap_admin_id}:${bootstrap_admin_secret}@${ca_url}" \
        -M "${msp_dir}" \
        --caname "${ca_name}"

    print_status $GREEN "✓ Bootstrap admin ${bootstrap_admin_id} enrolled"
    echo "${admin_dir}"
}

# Function to add affiliation
add_affiliation() {
    local admin_home=$1
    local affiliation=$2
    local ca_name=$3

    print_status $YELLOW "Adding affiliation: ${affiliation}"

    # Set FABRIC_CA_CLIENT_HOME to the admin directory
    export FABRIC_CA_CLIENT_HOME="${admin_home}"

    # Add the affiliation
    if output=$(fabric-ca-client affiliation add "${affiliation}" --caname "${ca_name}" 2>&1); then
        print_status $GREEN "✓ Affiliation ${affiliation} added"
    elif echo "$output" | grep -q "already exists"; then
        print_status $YELLOW "⚠ Affiliation ${affiliation} already exists, skipping..."
    else
        echo "$output" >&2
        exit 1
    fi
}

# Function to register identity
register_identity() {
    local admin_home=$1
    local enrollment_id=$2
    local enrollment_secret=$3
    local identity_type=$4
    local affiliation=$5
    local ca_name=$6

    print_status $YELLOW "Registering identity: ${enrollment_id}"

    # Set FABRIC_CA_CLIENT_HOME to the admin directory
    export FABRIC_CA_CLIENT_HOME="${admin_home}"

    # Register the identity
    if output=$(fabric-ca-client register \
        --id.name "${enrollment_id}" \
        --id.secret "${enrollment_secret}" \
        --id.type "${identity_type}" \
        --id.affiliation "${affiliation}" \
        --caname "${ca_name}" 2>&1); then
        print_status $GREEN "✓ Identity ${enrollment_id} registered"
    elif echo "$output" | grep -q "already registered"; then
        print_status $YELLOW "⚠ Identity ${enrollment_id} is already registered, skipping..."
    else
        echo "$output" >&2
        exit 1
    fi
}

# Function to get external host based on organization domain
get_external_host() {
    local org_type=$1
    local org_domain=$2

    # Map org domains to their external host variables
    if [ "${org_domain}" = "${ORDERER_DOMAIN}" ]; then
        echo "${ORDERER_EXTERNAL_HOST}"
    elif [ "${org_domain}" = "${ORG1_DOMAIN}" ]; then
        echo "${PEER0_ORG1_EXTERNAL_HOST}"
    elif [ "${org_domain}" = "${ORG2_DOMAIN}" ]; then
        echo "${PEER0_ORG2_EXTERNAL_HOST}"
    else
        echo ""
    fi
}

# Function to enroll identity
enroll_identity() {
    local ca_name=$1
    local ca_url=$2
    local enrollment_id=$3
    local enrollment_secret=$4
    local identity_type=$5
    local org_type=$6
    local org_domain=$7

    local identity_dir="${PROJECT_ROOT}/organizations/${org_type}Organizations/${org_domain}/${identity_type}s/${enrollment_id}.${org_domain}"
    local msp_dir="${identity_dir}/msp"

    # Get external host to include in CSR hosts
    local external_host=$(get_external_host "${org_type}" "${org_domain}")

    # Build CSR hosts list
    local csr_hosts="${enrollment_id}.${org_domain},localhost"
    if [ -n "${external_host}" ]; then
        csr_hosts="${csr_hosts},${external_host}"
    fi

    print_status $YELLOW "Enrolling identity: ${enrollment_id} with CSR hosts: ${csr_hosts}"

    # Clean up existing enrollment if it exists
    rm -rf "${identity_dir}"

    # Enroll the identity
    FABRIC_CA_CLIENT_HOME="${identity_dir}" \
    fabric-ca-client enroll \
        -u "http://${enrollment_id}:${enrollment_secret}@${ca_url}" \
        -M "${msp_dir}" \
        --csr.hosts "${csr_hosts}" \
        --caname "${ca_name}"

    # Create config.yaml for MSP
    cat > "${msp_dir}/config.yaml" << EOF
NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-${ca_url##*:}-${ca_name}.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-${ca_url##*:}-${ca_name}.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-${ca_url##*:}-${ca_name}.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-${ca_url##*:}-${ca_name}.pem
    OrganizationalUnitIdentifier: orderer
EOF

    print_status $GREEN "✓ Identity ${enrollment_id} enrolled"

    # Generate TLS certificates for peer identities
    if [ "${identity_type}" = "peer" ] || [ "${identity_type}" = "orderer" ]; then
        print_status $YELLOW "Enrolling TLS certificate for ${enrollment_id} with CSR hosts: ${csr_hosts}..."

        local tls_dir="${identity_dir}/tls"

        # Enroll TLS certificate
        FABRIC_CA_CLIENT_HOME="${tls_dir}" \
        fabric-ca-client enroll \
            -u "http://${enrollment_id}:${enrollment_secret}@${ca_url}" \
            -M "${tls_dir}" \
            --enrollment.profile tls \
            --csr.hosts "${csr_hosts}" \
            --caname "${ca_name}"

        # Create TLS file structure in expected format
        create_tls_file_structure "${tls_dir}"

        print_status $GREEN "✓ TLS certificate enrolled for ${enrollment_id}"
    fi
}

# Function to create TLS certificates in expected format for Fabric
create_tls_file_structure() {
    local tls_dir=$1

    print_status $YELLOW "Creating TLS file structure for Fabric..."

    # Copy signed certificate to server.crt
    if [ -f "${tls_dir}/signcerts/cert.pem" ]; then
        cp "${tls_dir}/signcerts/cert.pem" "${tls_dir}/server.crt"
        print_status $GREEN "✓ server.crt created"
    else
        print_status $RED "✗ Failed to create server.crt - signcerts/cert.pem not found"
        return 1
    fi

    # Copy private key to server.key (the key file has a generated name in keystore)
    if [ -d "${tls_dir}/keystore" ]; then
        cp "${tls_dir}/keystore/"*_sk "${tls_dir}/server.key" 2>/dev/null || true
        print_status $GREEN "✓ server.key created"
    else
        print_status $RED "✗ Failed to create server.key - keystore directory not found"
        return 1
    fi

    # Copy CA certificate to ca.crt (use tlscacerts for TLS enrollment)
    if [ -d "${tls_dir}/tlscacerts" ]; then
        cp "${tls_dir}/tlscacerts/"*.pem "${tls_dir}/ca.crt" 2>/dev/null || true
        print_status $GREEN "✓ ca.crt created"
    elif [ -d "${tls_dir}/cacerts" ]; then
        cp "${tls_dir}/cacerts/"*.pem "${tls_dir}/ca.crt" 2>/dev/null || true
        print_status $GREEN "✓ ca.crt created (from cacerts)"
    else
        print_status $RED "✗ Failed to create ca.crt - no CA certificate directory found"
        return 1
    fi

    print_status $GREEN "✓ TLS file structure created"
}

# Function to create organization-level MSP directory structure
create_org_msp_structure() {
    local org_type=$1
    local org_domain=$2
    local identity_dir=$3

    local org_msp_dir="${PROJECT_ROOT}/organizations/${org_type}Organizations/${org_domain}/msp"

    print_status $YELLOW "Creating organization-level MSP structure for ${org_domain}..."

    # Create organization MSP directory structure
    mkdir -p "${org_msp_dir}/cacerts"
    mkdir -p "${org_msp_dir}/tlscacerts"
    mkdir -p "${org_msp_dir}/intermediatecerts"
    mkdir -p "${org_msp_dir}/admincerts"

    # Copy CA certificates from the enrolled identity to the org-level MSP
    if [ -d "${identity_dir}/msp/cacerts" ]; then
        cp -r "${identity_dir}/msp/cacerts/"*.pem "${org_msp_dir}/cacerts/" 2>/dev/null || true
        print_status $GREEN "✓ CA certificates copied to ${org_msp_dir}/cacerts"
    fi

    # Copy TLS CA certificates if they exist
    # For orderers, TLS CA certificates are in the tls directory, not msp directory
    if [ -d "${identity_dir}/tls/tlscacerts" ]; then
        cp -r "${identity_dir}/tls/tlscacerts/"*.pem "${org_msp_dir}/tlscacerts/" 2>/dev/null || true
        print_status $GREEN "✓ TLS CA certificates copied from tls/tlscacerts to ${org_msp_dir}/tlscacerts"
    elif [ -d "${identity_dir}/msp/tlscacerts" ]; then
        cp -r "${identity_dir}/msp/tlscacerts/"*.pem "${org_msp_dir}/tlscacerts/" 2>/dev/null || true
        print_status $GREEN "✓ TLS CA certificates copied from msp/tlscacerts to ${org_msp_dir}/tlscacerts"
    else
        print_status $YELLOW "⚠ No TLS CA certificates found for ${org_domain}"
    fi

    # Copy intermediate certificates if they exist
    if [ -d "${identity_dir}/msp/intermediatecerts" ]; then
        cp -r "${identity_dir}/msp/intermediatecerts/"*.pem "${org_msp_dir}/intermediatecerts/" 2>/dev/null || true
        print_status $GREEN "✓ Intermediate certificates copied to ${org_msp_dir}/intermediatecerts"
    fi

    # Copy config.yaml for Node OUs if it exists
    if [ -f "${identity_dir}/msp/config.yaml" ]; then
        cp "${identity_dir}/msp/config.yaml" "${org_msp_dir}/config.yaml"
        print_status $GREEN "✓ config.yaml copied to ${org_msp_dir}"
    fi

    print_status $GREEN "✓ Organization-level MSP structure created for ${org_domain}"
}

# Setup Orderer CA identities
if [ "$DEPLOY_ORDERER" = true ]; then
    print_status $YELLOW "=== Setting up Orderer CA Identities ==="

    ca_name="ca-orderer"
    ca_url="localhost:${CA_ORDERER_PORT}"
    org_type="orderer"
    org_domain="${ORDERER_DOMAIN}"

    # Get CA certificate
    get_ca_cert "${ca_name}" "${ca_url}" "${org_type}" "${org_domain}"

    # Enroll bootstrap admin
    ca_hostname="ca.${org_domain}"
    bootstrap_admin_id="${ca_hostname}-admin"
    bootstrap_admin_secret="${ca_hostname}-adminpw"
    admin_home=$(enroll_bootstrap_admin "${ca_name}" "${ca_url}" "${bootstrap_admin_id}" "${bootstrap_admin_secret}" "${org_type}" "${org_domain}")

    # Add affiliation
    add_affiliation "${admin_home}" "orderer" "${ca_name}"

    # Register and enroll orderer admin
    register_identity "${admin_home}" "${CA_ORDERER_ENROLLMENT_ID}" "${CA_ORDERER_ENROLLMENT_SECRET}" "admin" "" "${ca_name}"
    enroll_identity "${ca_name}" "${ca_url}" "${CA_ORDERER_ENROLLMENT_ID}" "${CA_ORDERER_ENROLLMENT_SECRET}" "user" "${org_type}" "${org_domain}"

    # Register and enroll orderer
    register_identity "${admin_home}" "${ORDERER_ENROLLMENT_ID}" "${ORDERER_ENROLLMENT_SECRET}" "orderer" "" "${ca_name}"
    enroll_identity "${ca_name}" "${ca_url}" "${ORDERER_ENROLLMENT_ID}" "${ORDERER_ENROLLMENT_SECRET}" "orderer" "${org_type}" "${org_domain}"

    # Create organization-level MSP structure using the orderer identity
    orderer_identity_dir="${PROJECT_ROOT}/organizations/${org_type}Organizations/${org_domain}/orderers/${ORDERER_ENROLLMENT_ID}.${org_domain}"
    create_org_msp_structure "${org_type}" "${org_domain}" "${orderer_identity_dir}"
fi

# Setup Org1 CA identities
if [ "$DEPLOY_ORG1" = true ]; then
    print_status $YELLOW "=== Setting up Org1 CA Identities ==="

    ca_name="ca-org1"
    ca_url="localhost:${CA_ORG1_PORT}"
    org_type="peer"
    org_domain="${ORG1_DOMAIN}"

    # Get CA certificate
    get_ca_cert "${ca_name}" "${ca_url}" "${org_type}" "${org_domain}"

    # Enroll bootstrap admin
    ca_hostname="ca.${org_domain}"
    bootstrap_admin_id="${ca_hostname}-admin"
    bootstrap_admin_secret="${ca_hostname}-adminpw"
    admin_home=$(enroll_bootstrap_admin "${ca_name}" "${ca_url}" "${bootstrap_admin_id}" "${bootstrap_admin_secret}" "${org_type}" "${org_domain}")

    # Add affiliation
    add_affiliation "${admin_home}" "org1" "${ca_name}"

    # Register and enroll org1 admin
    register_identity "${admin_home}" "${ORG1_ADMIN_ENROLLMENT_ID}" "${ORG1_ADMIN_ENROLLMENT_SECRET}" "admin" "" "${ca_name}"
    enroll_identity "${ca_name}" "${ca_url}" "${ORG1_ADMIN_ENROLLMENT_ID}" "${ORG1_ADMIN_ENROLLMENT_SECRET}" "user" "${org_type}" "${org_domain}"

    # Register and enroll org1 peer
    register_identity "${admin_home}" "${ORG1_PEER_ENROLLMENT_ID}" "${ORG1_PEER_ENROLLMENT_SECRET}" "peer" "" "${ca_name}"
    enroll_identity "${ca_name}" "${ca_url}" "${ORG1_PEER_ENROLLMENT_ID}" "${ORG1_PEER_ENROLLMENT_SECRET}" "peer" "${org_type}" "${org_domain}"

    # Register and enroll org1 user
    register_identity "${admin_home}" "${ORG1_USER_ENROLLMENT_ID}" "${ORG1_USER_ENROLLMENT_SECRET}" "client" "" "${ca_name}"
    enroll_identity "${ca_name}" "${ca_url}" "${ORG1_USER_ENROLLMENT_ID}" "${ORG1_USER_ENROLLMENT_SECRET}" "user" "${org_type}" "${org_domain}"

    # Create organization-level MSP structure using the peer identity
    org1_peer_identity_dir="${PROJECT_ROOT}/organizations/${org_type}Organizations/${org_domain}/peers/${ORG1_PEER_ENROLLMENT_ID}.${org_domain}"
    create_org_msp_structure "${org_type}" "${org_domain}" "${org1_peer_identity_dir}"
fi

# Setup Org2 CA identities
if [ "$DEPLOY_ORG2" = true ]; then
    print_status $YELLOW "=== Setting up Org2 CA Identities ==="

    ca_name="ca-org2"
    ca_url="localhost:${CA_ORG2_PORT}"
    org_type="peer"
    org_domain="${ORG2_DOMAIN}"

    # Get CA certificate
    get_ca_cert "${ca_name}" "${ca_url}" "${org_type}" "${org_domain}"

    # Enroll bootstrap admin
    ca_hostname="ca.${org_domain}"
    bootstrap_admin_id="${ca_hostname}-admin"
    bootstrap_admin_secret="${ca_hostname}-adminpw"
    admin_home=$(enroll_bootstrap_admin "${ca_name}" "${ca_url}" "${bootstrap_admin_id}" "${bootstrap_admin_secret}" "${org_type}" "${org_domain}")

    # Add affiliation
    add_affiliation "${admin_home}" "org2" "${ca_name}"

    # Register and enroll org2 admin
    register_identity "${admin_home}" "${ORG2_ADMIN_ENROLLMENT_ID}" "${ORG2_ADMIN_ENROLLMENT_SECRET}" "admin" "" "${ca_name}"
    enroll_identity "${ca_name}" "${ca_url}" "${ORG2_ADMIN_ENROLLMENT_ID}" "${ORG2_ADMIN_ENROLLMENT_SECRET}" "user" "${org_type}" "${org_domain}"

    # Register and enroll org2 peer
    register_identity "${admin_home}" "${ORG2_PEER_ENROLLMENT_ID}" "${ORG2_PEER_ENROLLMENT_SECRET}" "peer" "" "${ca_name}"
    enroll_identity "${ca_name}" "${ca_url}" "${ORG2_PEER_ENROLLMENT_ID}" "${ORG2_PEER_ENROLLMENT_SECRET}" "peer" "${org_type}" "${org_domain}"

    # Register and enroll org2 user
    register_identity "${admin_home}" "${ORG2_USER_ENROLLMENT_ID}" "${ORG2_USER_ENROLLMENT_SECRET}" "client" "" "${ca_name}"
    enroll_identity "${ca_name}" "${ca_url}" "${ORG2_USER_ENROLLMENT_ID}" "${ORG2_USER_ENROLLMENT_SECRET}" "user" "${org_type}" "${org_domain}"

    # Create organization-level MSP structure using the peer identity
    org2_peer_identity_dir="${PROJECT_ROOT}/organizations/${org_type}Organizations/${org_domain}/peers/${ORG2_PEER_ENROLLMENT_ID}.${org_domain}"
    create_org_msp_structure "${org_type}" "${org_domain}" "${org2_peer_identity_dir}"
fi

# Distribute TLS CA certificates between organizations for cross-org TLS communication
print_status $YELLOW "=== Distributing TLS CA certificates between organizations ==="

# Function to copy TLS CA certificate to a peer
copy_tls_ca_to_peer() {
    local source_tls_ca=$1
    local target_peer_tls_dir=$2
    local source_org=$3

    if [ -f "${source_tls_ca}" ] && [ -d "${target_peer_tls_dir}" ]; then
        cp "${source_tls_ca}" "${target_peer_tls_dir}/"
        print_status $GREEN "✓ Copied ${source_org} TLS CA certificate to peer"
    else
        print_status $RED "✗ Failed to copy ${source_org} TLS CA certificate - file or directory not found"
    fi
}

# Function to regenerate ca.crt file by concatenating all TLS CA certificates
regenerate_peer_ca_crt() {
    local peer_tls_dir=$1
    local peer_name=$2

    if [ -d "${peer_tls_dir}" ]; then
        if [ -d "${peer_tls_dir}/tlscacerts" ]; then
            # Concatenate all TLS CA certificates into ca.crt
            cat "${peer_tls_dir}/tlscacerts/"*.pem > "${peer_tls_dir}/ca.crt" 2>/dev/null
            print_status $GREEN "✓ Regenerated ca.crt for ${peer_name} with all TLS CA certificates"
        else
            print_status $RED "✗ Failed to regenerate ca.crt for ${peer_name} - tlscacerts directory not found"
        fi
    else
        print_status $RED "✗ Failed to regenerate ca.crt for ${peer_name} - TLS directory not found"
    fi
}

# If both Org1 and Org2 are deployed, distribute TLS CA certificates between them
if [ "$DEPLOY_ORG1" = true ] && [ "$DEPLOY_ORG2" = true ]; then
    # Org1 TLS CA certificate path
    org1_tls_ca="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/tlscacerts/tls-localhost-${CA_ORG1_PORT}-ca-org1.pem"

    # Org2 TLS CA certificate path
    org2_tls_ca="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/tlscacerts/tls-localhost-${CA_ORG2_PORT}-ca-org2.pem"

    # Copy Org1 TLS CA to Org2 peers
    print_status $YELLOW "Distributing Org1 TLS CA certificate to Org2 peers..."
    copy_tls_ca_to_peer "${org1_tls_ca}" "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/tlscacerts" "Org1"

    # Copy Org2 TLS CA to Org1 peers
    print_status $YELLOW "Distributing Org2 TLS CA certificate to Org1 peers..."
    copy_tls_ca_to_peer "${org2_tls_ca}" "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/tlscacerts" "Org2"

    print_status $GREEN "✓ TLS CA certificates distributed between organizations"
fi

# # Also distribute Orderer TLS CA to peer organizations for TLS communication with orderer
# if [ "$DEPLOY_ORDERER" = true ]; then
#     if [ "$DEPLOY_ORG1" = true ]; then
#         # Copy Orderer TLS CA to Org1 peers
#         print_status $YELLOW "Distributing Orderer TLS CA certificate to Org1 peers..."
#         orderer_tls_ca="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/tlscacerts/tls-localhost-${CA_ORDERER_PORT}-${CA_ORDERER_NAME}.pem"
#         copy_tls_ca_to_peer "${orderer_tls_ca}" "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls/tlscacerts" "Orderer"
#     fi

#     if [ "$DEPLOY_ORG2" = true ]; then
#         # Copy Orderer TLS CA to Org2 peers
#         print_status $YELLOW "Distributing Orderer TLS CA certificate to Org2 peers..."
#         orderer_tls_ca="${PROJECT_ROOT}/organizations/ordererOrganizations/${ORDERER_DOMAIN}/orderers/orderer1.${ORDERER_DOMAIN}/tls/tlscacerts/tls-localhost-${CA_ORDERER_PORT}-${CA_ORDERER_NAME}.pem"
#         copy_tls_ca_to_peer "${orderer_tls_ca}" "${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls/tlscacerts" "Orderer"
#     fi
# fi

# Regenerate ca.crt files for all peers to include all distributed TLS CA certificates
print_status $YELLOW "=== Regenerating ca.crt files for all peers ==="

if [ "$DEPLOY_ORG1" = true ]; then
    peer1_tls_dir="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG1_DOMAIN}/peers/peer0.${ORG1_DOMAIN}/tls"
    regenerate_peer_ca_crt "${peer1_tls_dir}" "peer0.${ORG1_DOMAIN}"
fi

if [ "$DEPLOY_ORG2" = true ]; then
    peer2_tls_dir="${PROJECT_ROOT}/organizations/peerOrganizations/${ORG2_DOMAIN}/peers/peer0.${ORG2_DOMAIN}/tls"
    regenerate_peer_ca_crt "${peer2_tls_dir}" "peer0.${ORG2_DOMAIN}"
fi

print_status $GREEN "=== CA Identity Setup Completed Successfully ==="
print_status $YELLOW "Next step: Run 004-generate-configtx.sh"
