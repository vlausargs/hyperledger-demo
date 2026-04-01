#!/bin/bash

# Script 001: Deploy PostgreSQL Databases for Fabric CA
# This script deploys PostgreSQL databases for each CA in the network

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

print_status $GREEN "=== Starting PostgreSQL Deployment ==="
print_status $YELLOW "Deployment Mode:"
echo "  Orderer: $DEPLOY_ORDERER"
echo "  Org1: $DEPLOY_ORG1"
echo "  Org2: $DEPLOY_ORG2"
echo ""

# Function to create docker-compose file for PostgreSQL
create_postgres_compose() {
    local db_name=$1
    local db_port=$2
    local db_user=$3
    local db_pass=$4
    local db_database=$5
    local compose_file=$6

    cat > "$compose_file" << EOF
services:
  ${db_name}:
    image: ${POSTGRES_IMAGE}
    container_name: ${db_name}
    environment:
      POSTGRES_USER: ${db_user}
      POSTGRES_PASSWORD: ${db_pass}
      POSTGRES_DB: ${db_database}
      PGPORT: ${db_port}
    volumes:
      - ${db_name}_hlf_data:/var/lib/postgresql/data
    network_mode: host
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${db_user} -d ${db_database} -p ${db_port}"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: ${POSTGRES_MEMORY_LIMIT}
          cpus: '${POSTGRES_CPU_LIMIT}'
        reservations:
          memory: ${POSTGRES_MEMORY_RESERVE}
          cpus: '${POSTGRES_CPU_RESERVE}'

volumes:
  ${db_name}_hlf_data:
    driver: local
EOF
}

# Deploy Orderer PostgreSQL
if [ "$DEPLOY_ORDERER" = true ]; then
    print_status $YELLOW "Deploying PostgreSQL for Orderer CA..."

    compose_file="${PROJECT_ROOT}/docker-compose/postgres/orderer-postgres.yml"
    create_postgres_compose \
        "${POSTGRES_ORDERER_HOST}" \
        "${POSTGRES_ORDERER_PORT}" \
        "${POSTGRES_ORDERER_USER}" \
        "${POSTGRES_ORDERER_PASSWORD}" \
        "${POSTGRES_ORDERER_DB}" \
        "$compose_file"

    docker compose -f "$compose_file" up -d
    print_status $GREEN "✓ Orderer PostgreSQL deployed successfully"
fi

# Deploy Org1 PostgreSQL
if [ "$DEPLOY_ORG1" = true ]; then
    print_status $YELLOW "Deploying PostgreSQL for Org1 CA..."

    compose_file="${PROJECT_ROOT}/docker-compose/postgres/org1-postgres.yml"
    create_postgres_compose \
        "${POSTGRES_ORG1_HOST}" \
        "${POSTGRES_ORG1_PORT}" \
        "${POSTGRES_ORG1_USER}" \
        "${POSTGRES_ORG1_PASSWORD}" \
        "${POSTGRES_ORG1_DB}" \
        "$compose_file"

    docker compose -f "$compose_file" up -d
    print_status $GREEN "✓ Org1 PostgreSQL deployed successfully"
fi

# Deploy Org2 PostgreSQL
if [ "$DEPLOY_ORG2" = true ]; then
    print_status $YELLOW "Deploying PostgreSQL for Org2 CA..."

    compose_file="${PROJECT_ROOT}/docker-compose/postgres/org2-postgres.yml"
    create_postgres_compose \
        "${POSTGRES_ORG2_HOST}" \
        "${POSTGRES_ORG2_PORT}" \
        "${POSTGRES_ORG2_USER}" \
        "${POSTGRES_ORG2_PASSWORD}" \
        "${POSTGRES_ORG2_DB}" \
        "$compose_file"

    docker compose -f "$compose_file" up -d
    print_status $GREEN "✓ Org2 PostgreSQL deployed successfully"
fi

# Wait for databases to be ready
print_status $YELLOW "Waiting for PostgreSQL databases to be ready..."
sleep 10

# Verify databases
verify_database() {
    local db_name=$1
    local db_port=$2
    local db_user=$3
    local db_pass=$4
    local db_database=$5

    print_status $YELLOW "Verifying $db_name..."

    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if docker exec $db_name pg_isready -U $db_user -d $db_database > /dev/null 2>&1; then
            print_status $GREEN "✓ $db_name is ready"
            return 0
        fi

        echo "  Attempt $attempt/$max_attempts: Waiting for database..."
        sleep 2
        attempt=$((attempt + 1))
    done

    print_status $RED "✗ $db_name failed to start"
    return 1
}

# Verify all databases if they're deployed
if [ "$DEPLOY_ORDERER" = true ]; then
    verify_database "${POSTGRES_ORDERER_HOST}" "${POSTGRES_ORDERER_PORT}" "${POSTGRES_ORDERER_USER}" "${POSTGRES_ORDERER_PASSWORD}" "${POSTGRES_ORDERER_DB}"
fi

if [ "$DEPLOY_ORG1" = true ]; then
    verify_database "${POSTGRES_ORG1_HOST}" "${POSTGRES_ORG1_PORT}" "${POSTGRES_ORG1_USER}" "${POSTGRES_ORG1_PASSWORD}" "${POSTGRES_ORG1_DB}"
fi

if [ "$DEPLOY_ORG2" = true ]; then
    verify_database "${POSTGRES_ORG2_HOST}" "${POSTGRES_ORG2_PORT}" "${POSTGRES_ORG2_USER}" "${POSTGRES_ORG2_PASSWORD}" "${POSTGRES_ORG2_DB}"
fi

print_status $GREEN "=== PostgreSQL Deployment Completed Successfully ==="
print_status $YELLOW "Next step: Run 002-deploy-ca.sh"
