#!/bin/bash

# Script 002: Deploy Fabric CA Services
# This script deploys Fabric CA services for the orderer and organizations

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Set default CSR values for CA certificates
CA_CSR_COUNTRY=${CA_CSR_COUNTRY:-"US"}
CA_CSR_STATE=${CA_CSR_STATE:-"California"}
CA_CSR_LOCALITY=${CA_CSR_LOCALITY:-"San Francisco"}
CA_CSR_ORGANIZATION=${CA_CSR_ORGANIZATION:-"Hyperledger"}
CA_CSR_ORGANIZATIONAL_UNIT=${CA_CSR_ORGANIZATIONAL_UNIT:-"Fabric"}

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

print_status $YELLOW "CSR Configuration:"
echo "  Country: $CA_CSR_COUNTRY"
echo "  State: $CA_CSR_STATE"
echo "  Locality: $CA_CSR_LOCALITY"
echo "  Organization: $CA_CSR_ORGANIZATION"
echo "  Organizational Unit: $CA_CSR_ORGANIZATIONAL_UNIT"
echo ""

# Function to generate CA server configuration file
generate_ca_server_config() {
    local ca_name=$1
    local ca_hostname=$2
    local config_dir=$3
    local db_host=$4
    local db_port=$5
    local db_user=$6
    local db_pass=$7
    local db_name=$8

    mkdir -p "${config_dir}"

    cat > "${config_dir}/fabric-ca-server-config.yaml" << EOF
ca:
  name: ${ca_name}
  keyfile: priv_sk
  certfile: ca.${ca_hostname}-cert.pem
  chainfile: ca-chain.pem

address: 0.0.0.0
port: 7054

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
          hf.Registrar.Roles: "client,user,peer,orderer,admin"
          hf.Registrar.DelegateRoles: "client,user,peer,orderer,admin"
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
      - C: ${CA_CSR_COUNTRY}
        ST: ${CA_CSR_STATE}
        L: ${CA_CSR_LOCALITY}
        O: ${CA_CSR_ORGANIZATION}
        OU: ${CA_CSR_ORGANIZATIONAL_UNIT}
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

mtime: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
EOF

    print_status $GREEN "✓ Generated CA server config: ${config_dir}/fabric-ca-server-config.yaml"
}

# Function to create CA docker-compose file
create_ca_compose() {
    local ca_name=$1
    local ca_hostname=$2
    local ca_port=$3
    local config_dir=$4
    local compose_file=$5

    cat > "$compose_file" << EOF
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
    ports:
      - "${ca_port}:7054"
    volumes:
      - ${ca_name}_hlf_data:/etc/hyperledger/fabric-ca-server
      - ${config_dir}/fabric-ca-server-config.yaml:/etc/hyperledger/fabric-ca-server/fabric-ca-server-config.yaml
    networks:
      - ${NETWORK_NAME}
    command: sh -c "fabric-ca-server start -b ${ca_hostname}-admin:${ca_hostname}-adminpw -c /etc/hyperledger/fabric-ca-server/fabric-ca-server-config.yaml -d"
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

    print_status $GREEN "✓ Generated docker-compose: ${compose_file}"
}

# Create directories
CONFIG_DIR="${PROJECT_ROOT}/config"
COMPOSE_DIR="${PROJECT_ROOT}/docker-compose/ca"

mkdir -p "${CONFIG_DIR}/ca/orderer"
mkdir -p "${CONFIG_DIR}/ca/org1"
mkdir -p "${CONFIG_DIR}/ca/org2"
mkdir -p "${COMPOSE_DIR}"

# Deploy Orderer CA
if [ "$DEPLOY_ORDERER" = true ]; then
    print_status $YELLOW "Deploying Orderer CA..."

    config_dir="${CONFIG_DIR}/ca/orderer"
    compose_file="${COMPOSE_DIR}/orderer-ca.yml"

    # Generate CA server config
    generate_ca_server_config \
        "${CA_ORDERER_NAME}" \
        "${CA_ORDERER_HOSTNAME}" \
        "${config_dir}" \
        "${POSTGRES_ORDERER_HOST}" \
        "${POSTGRES_ORDERER_PORT}" \
        "${POSTGRES_ORDERER_USER}" \
        "${POSTGRES_ORDERER_PASSWORD}" \
        "${POSTGRES_ORDERER_DB}"

    # Generate docker-compose file
    create_ca_compose \
        "${CA_ORDERER_NAME}" \
        "${CA_ORDERER_HOSTNAME}" \
        "${CA_ORDERER_PORT}" \
        "${config_dir}" \
        "${compose_file}"

    docker-compose -f "$compose_file" up -d
    print_status $GREEN "✓ Orderer CA deployed successfully"
fi

# Deploy Org1 CA
if [ "$DEPLOY_ORG1" = true ]; then
    print_status $YELLOW "Deploying Org1 CA..."

    config_dir="${CONFIG_DIR}/ca/org1"
    compose_file="${COMPOSE_DIR}/org1-ca.yml"

    # Generate CA server config
    generate_ca_server_config \
        "${CA_ORG1_NAME}" \
        "${CA_ORG1_HOSTNAME}" \
        "${config_dir}" \
        "${POSTGRES_ORG1_HOST}" \
        "${POSTGRES_ORG1_PORT}" \
        "${POSTGRES_ORG1_USER}" \
        "${POSTGRES_ORG1_PASSWORD}" \
        "${POSTGRES_ORG1_DB}"

    # Generate docker-compose file
    create_ca_compose \
        "${CA_ORG1_NAME}" \
        "${CA_ORG1_HOSTNAME}" \
        "${CA_ORG1_PORT}" \
        "${config_dir}" \
        "${compose_file}"

    docker-compose -f "$compose_file" up -d
    print_status $GREEN "✓ Org1 CA deployed successfully"
fi

# Deploy Org2 CA
if [ "$DEPLOY_ORG2" = true ]; then
    print_status $YELLOW "Deploying Org2 CA..."

    config_dir="${CONFIG_DIR}/ca/org2"
    compose_file="${COMPOSE_DIR}/org2-ca.yml"

    # Generate CA server config
    generate_ca_server_config \
        "${CA_ORG2_NAME}" \
        "${CA_ORG2_HOSTNAME}" \
        "${config_dir}" \
        "${POSTGRES_ORG2_HOST}" \
        "${POSTGRES_ORG2_PORT}" \
        "${POSTGRES_ORG2_USER}" \
        "${POSTGRES_ORG2_PASSWORD}" \
        "${POSTGRES_ORG2_DB}"

    # Generate docker-compose file
    create_ca_compose \
        "${CA_ORG2_NAME}" \
        "${CA_ORG2_HOSTNAME}" \
        "${CA_ORG2_PORT}" \
        "${config_dir}" \
        "${compose_file}"

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
print_status $BLUE "Configuration files generated in: ${CONFIG_DIR}/ca/"
print_status $BLUE "Docker compose files generated in: ${COMPOSE_DIR}"
print_status $YELLOW "Next step: Run 003-setup-ca.sh"
