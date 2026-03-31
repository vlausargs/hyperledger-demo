#!/bin/bash
# Helper script for Fabric environment setup and binary paths
# This script provides common functions to use local Fabric binaries instead of docker exec

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Set up Fabric binaries path
export FABRIC_BIN_PATH="${PROJECT_ROOT}/fabric-binaries/bin"
export FABRIC_CONFIG_PATH="${PROJECT_ROOT}/config"

# Set up organizations path
export FABRIC_ORGANIZATIONS_PATH="${PROJECT_ROOT}/organizations"

# Set up channel artifacts path
export FABRIC_CHANNEL_ARTIFACTS_PATH="${PROJECT_ROOT}/config/channel-artifacts"

# Add Fabric binaries to PATH if not already present
if [[ ":$PATH:" != *":${FABRIC_BIN_PATH}:"* ]]; then
    export PATH="${FABRIC_BIN_PATH}:${PATH}"
fi

# Function to run orderer command locally
run_orderer() {
    local peer_org="$1"
    local orderer_name="orderer1.${ORDERER_DOMAIN}"

    # Set up environment variables for orderer
    export ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
    export ORDERER_GENERAL_LISTENPORT=${ORDERER_PORT}
    export ORDERER_GENERAL_LOCALMSPID=${ORDERER_ORG_NAME}
    export ORDERER_GENERAL_LOCALMSPDIR="${FABRIC_ORGANIZATIONS_PATH}/ordererOrganizations/${ORDERER_DOMAIN}/orderers/${orderer_name}/msp"
    export ORDERER_GENERAL_TLS_ENABLED=${TLS_ENABLED}
    export ORDERER_GENERAL_TLS_PRIVATEKEY="${FABRIC_ORGANIZATIONS_PATH}/ordererOrganizations/${ORDERER_DOMAIN}/orderers/${orderer_name}/tls/server.key"
    export ORDERER_GENERAL_TLS_CERTIFICATE="${FABRIC_ORGANIZATIONS_PATH}/ordererOrganizations/${ORDERER_DOMAIN}/orderers/${orderer_name}/tls/server.crt"
    export ORDERER_GENERAL_TLS_ROOTCAS="${FABRIC_ORGANIZATIONS_PATH}/ordererOrganizations/${ORDERER_DOMAIN}/orderers/${orderer_name}/tls/ca.crt"
    export ORDERER_GENERAL_GENESISFILE="${FABRIC_CHANNEL_ARTIFACTS_PATH}/genesis.block"

    # Run the orderer command
    "${FABRIC_BIN_PATH}/orderer" "$@"
}

# Function to run peer command locally
run_peer() {
    local org_domain="$1"
    local org_name="$2"
    local peer_name="$3"
    shift 3  # Remove the first three arguments

    # Get external host and port for this organization
    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    # Set up environment variables for peer
    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LISTENADDRESS=0.0.0.0:${peer_port}
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ENABLED=${TLS_ENABLED}
    export CORE_PEER_TLS_CERT_FILE="${FABRIC_ORGANIZATIONS_PATH}/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls/server.crt"
    export CORE_PEER_TLS_KEY_FILE="${FABRIC_ORGANIZATIONS_PATH}/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls/server.key"
    export CORE_PEER_TLS_ROOTCERT_FILE="${FABRIC_ORGANIZATIONS_PATH}/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls/ca.crt"
    export FABRIC_CFG_PATH="${FABRIC_CONFIG_PATH}"

    # Run the peer command
    "${FABRIC_BIN_PATH}/peer" "$@"
}

# Function to run peer command with full environment
run_peer_full_env() {
    local org_domain="$1"
    local org_name="$2"
    local peer_name="$3"
    shift 3  # Remove the first three arguments

    # Get external host and port for this organization
    local external_host=$(get_external_host "peer" "${org_domain}")
    local peer_port=$(get_peer_port "${org_domain}")

    # Set up environment variables for peer
    export CORE_PEER_ID="${peer_name}.${org_domain}"
    export CORE_PEER_ADDRESS="${external_host}:${peer_port}"
    export CORE_PEER_LISTENADDRESS=0.0.0.0:${peer_port}
    export CORE_PEER_LOCALMSPID="${org_name}"
    export CORE_PEER_TLS_ENABLED=${TLS_ENABLED}
    export CORE_PEER_TLS_CERT_FILE="${FABRIC_ORGANIZATIONS_PATH}/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls/server.crt"
    export CORE_PEER_TLS_KEY_FILE="${FABRIC_ORGANIZATIONS_PATH}/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls/server.key"
    export CORE_PEER_TLS_ROOTCERT_FILE="${FABRIC_ORGANIZATIONS_PATH}/peerOrganizations/${org_domain}/peers/${peer_name}.${org_domain}/tls/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="${FABRIC_ORGANIZATIONS_PATH}/peerOrganizations/${org_domain}/users/admin.${org_domain}/msp"
    export CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock
    export DOCKER_HOST=unix:///host/var/run/docker.sock
    export FABRIC_CFG_PATH="${FABRIC_CONFIG_PATH}"

    # Run the peer command
    "${FABRIC_BIN_PATH}/peer" "$@"
}

# Function to run fabric-ca-client command
run_fabric_ca_client() {
    local ca_name="$1"
    local ca_url="$2"
    shift 2

    # Set up environment variables for fabric-ca-client
    export FABRIC_CA_CLIENT_HOME="${FABRIC_CONFIG_PATH}/fabric-ca-client/${ca_name}"

    # Run the fabric-ca-client command
    "${FABRIC_BIN_PATH}/fabric-ca-client" "$@"
}

# Function to get CA URL for an organization
get_ca_url() {
    local org_type="$1"
    local org_domain="$2"

    case "${org_domain}" in
        "${ORDERER_DOMAIN}")
            echo "${CA_ORDERER_HOSTNAME}:${CA_ORDERER_PORT}"
            ;;
        "${ORG1_DOMAIN}")
            echo "${CA_ORG1_HOSTNAME}:${CA_ORG1_PORT}"
            ;;
        "${ORG2_DOMAIN}")
            echo "${CA_ORG2_HOSTNAME}:${CA_ORG2_PORT}"
            ;;
        *)
            echo ""
            ;;
    esac
}

# Function to get peer port for an organization
get_peer_port() {
    local org_domain="$1"

    case "${org_domain}" in
        "${ORG1_DOMAIN}")
            echo "${PEER0_ORG1_PORT}"
            ;;
        "${ORG2_DOMAIN}")
            echo "${PEER0_ORG2_PORT}"
            ;;
        *)
            echo ""
            ;;
    esac
}

# Function to get external host for an organization
get_external_host() {
    local org_type="$1"
    local org_domain="$2"

    case "${org_domain}" in
        "${ORDERER_DOMAIN}")
            echo "${ORDERER_EXTERNAL_HOST}"
            ;;
        "${ORG1_DOMAIN}")
            echo "${PEER0_ORG1_EXTERNAL_HOST}"
            ;;
        "${ORG2_DOMAIN}")
            echo "${PEER0_ORG2_EXTERNAL_HOST}"
            ;;
        *)
            echo ""
            ;;
    esac
}

# Function to get org name from domain
get_org_name() {
    local org_domain="$1"

    case "${org_domain}" in
        "${ORDERER_DOMAIN}")
            echo "${ORDERER_ORG_NAME}"
            ;;
        "${ORG1_DOMAIN}")
            echo "${ORG1_NAME}"
            ;;
        "${ORG2_DOMAIN}")
            echo "${ORG2_NAME}"
            ;;
        *)
            echo ""
            ;;
    esac
}

# Function to verify binary exists
verify_binary() {
    local binary_name="$1"

    if [ ! -f "${FABRIC_BIN_PATH}/${binary_name}" ]; then
        echo "Error: ${binary_name} not found at ${FABRIC_BIN_PATH}"
        return 1
    fi

    return 0
}

# Function to verify all required binaries
verify_binaries() {
    local binaries=("configtxgen" "cryptogen" "peer" "orderer" "fabric-ca-client")
    local missing=0

    for binary in "${binaries[@]}"; do
        if ! verify_binary "${binary}"; then
            missing=$((missing + 1))
        fi
    done

    if [ ${missing} -gt 0 ]; then
        echo "Error: ${missing} required binary/binaries missing"
        return 1
    fi

    return 0
}

# Function to start local fabric-ca-server
start_ca_server() {
    local ca_name="$1"
    local ca_config_dir="$2"
    local ca_port="$3"
    local db_host="$4"
    local db_user="$5"
    local db_pass="$6"
    local db_name="$7"

    echo "Starting ${ca_name} on port ${ca_port}..."

    # Create CA data directory
    local ca_data_dir="${ca_config_dir}/data"
    mkdir -p "${ca_data_dir}"

    # Set up environment variables for CA server
    export FABRIC_CA_HOME="${ca_config_dir}"
    export FABRIC_CA_SERVER_HOME="${ca_config_dir}"

    # Start CA server in background
    nohup "${FABRIC_BIN_PATH}/fabric-ca-server" start \
        -b "${ca_name}-admin:${ca_name}-adminpw" \
        -d \
        -c "${ca_config_dir}/fabric-ca-server-config.yaml" \
        > "${ca_config_dir}/ca-server.log" 2>&1 &

    # Save PID
    echo $! > "${ca_config_dir}/ca-server.pid"

    echo "${ca_name} started with PID $(cat ${ca_config_dir}/ca-server.pid)"
}

# Function to stop local fabric-ca-server
stop_ca_server() {
    local ca_config_dir="$1"
    local pid_file="${ca_config_dir}/ca-server.pid"

    if [ -f "${pid_file}" ]; then
        local pid=$(cat "${pid_file}")
        if ps -p ${pid} > /dev/null 2>&1; then
            echo "Stopping CA server (PID: ${pid})..."
            kill ${pid}
            rm -f "${pid_file}"
            echo "CA server stopped"
        else
            echo "CA server is not running"
            rm -f "${pid_file}"
        fi
    else
        echo "CA server PID file not found"
    fi
}

# Function to create CA server configuration
create_ca_server_config() {
    local ca_name="$1"
    local ca_hostname="$2"
    local ca_config_dir="$3"
    local db_host="$4"
    local db_user="$5"
    local db_pass="$6"
    local db_name="$7"
    local ca_port="$8"
    local db_port="$9"

    mkdir -p "${ca_config_dir}"

    cat > "${ca_config_dir}/fabric-ca-server-config.yaml" << EOF
ca:
  name: ${ca_name}
  keyfile: priv_sk
  certfile: ca.${ca_hostname}-cert.pem
  chainfile: ca-chain.pem

address: 0.0.0.0
port: ${ca_port}

intermediate:
  parentserver:
    url:
    caname:

crl:
  expiry: 24h

registry:
  maxenrollments: -1

  identities:
     - name: ${ca_hostname}-admin
       pass: ${ca_hostname}-adminpw
       type: client
       affiliation: ""
       attrs:
          hf.Registrar.Roles: "client,user,peer,orderer"
          hf.Registrar.DelegateRoles: "client,user,peer,orderer"
          hf.Revoker: true
          hf.IntermediateCA: true
          hf.GenCRL: true
          hf.Registrar.Attributes: "*"
          hf.AffiliationMgr: true

database:
  type: postgres
  datasource: host=${db_host} port=${db_port} user=${db_user} password=${db_pass} dbname=${db_name} sslmode=disable
  tls:
      enabled: false
      certfiles:
      client:
        certfile:
        keyfile:

ldap:
   enabled: false
   url: ldap://<adminDN>:<adminPassword>@<host>:<port>/<base>
   tls:
      certfiles:
      client:
         certfile:
         keyfile:
      CACerts:

affiliations:
   orderer:
   org1:
   org2:

signing:
    default:
      expiry: 8760h
    profiles:
      tls:
         usage:
           - digital signature
           - key encipherment
           - server auth
           - client auth
         expiry: 8760h

csr:
   cn: ${ca_name}
   names:
      - C: US
        ST: "New York"
        L: "New York"
        O: Hyperledger
        OU: Fabric
   hosts:
     - ${ca_hostname}
     - localhost
   ca:
      expiry: 131400h
      pathlen: 1

bccsp:
    default: SW
    sw:
        hash: SHA2
        security: 256
        filekeystore:
            keystore: msp/keystore

cacount:

mtime: 2024-01-01T00:00:00Z
EOF

    echo "CA server configuration created at ${ca_config_dir}/fabric-ca-server-config.yaml"
}

# Export functions for use in other scripts
export -f run_orderer
export -f run_peer
export -f run_peer_full_env
export -f run_fabric_ca_client
export -f get_ca_url
export -f get_peer_port
export -f get_external_host
export -f get_org_name
export -f verify_binary
export -f verify_binaries
export -f start_ca_server
export -f stop_ca_server
export -f create_ca_server_config
