#!/bin/bash

# Script 999: Teardown Network
# This script stops and cleans up the Hyperledger Fabric network

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

# Parse command line arguments
FULL_CLEANUP=false
KEEP_DATA=false
QUIET=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --full)
            FULL_CLEANUP=true
            shift
            ;;
        --keep-data)
            KEEP_DATA=true
            shift
            ;;
        --quiet)
            QUIET=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --full      Perform full cleanup including volumes and artifacts"
            echo "  --keep-data Keep Docker volumes and data"
            echo "  --quiet     Suppress non-error output"
            echo "  --help, -h  Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

print_status $GREEN "=== Starting Network Teardown ==="
print_status $YELLOW "Teardown Options:"
echo "  Full Cleanup: $FULL_CLEANUP"
echo "  Keep Data: $KEEP_DATA"
echo "  Quiet Mode: $QUIET"
echo ""

# Function to stop and remove containers
stop_containers() {
    print_status $YELLOW "Stopping and removing containers..."

    # Get all fabric-related containers
    local containers=$(docker ps -a --format "{{.Names}}" | grep -E "orderer|peer|ca|couchdb|postgres|cli" | grep -v rabbitmq | grep -v elasticsearch | grep -v openbao | grep -v redis || true)

    if [ -z "$containers" ]; then
        print_status $BLUE "No Fabric containers found"
    else
        if [ "$QUIET" != true ]; then
            echo "Found containers:"
            echo "$containers"
            echo ""
        fi

        # Stop containers
        docker stop $containers 2>/dev/null || true
        print_status $GREEN "✓ Containers stopped"

        # Remove containers
        docker rm -f $containers 2>/dev/null || true
        print_status $GREEN "✓ Containers removed"
    fi
}

# Function to remove volumes
remove_volumes() {
    if [ "$KEEP_DATA" = true ]; then
        print_status $YELLOW "Keeping Docker volumes as requested (--keep-data)"
        return
    fi

    print_status $YELLOW "Removing Docker volumes..."

    # Prune stopped containers first to release volume references
    docker container prune -f > /dev/null 2>&1 || true

    # Get all fabric-related volumes
    local volumes=$(docker volume ls --format "{{.Name}}" | grep -E "orderer|peer|ca|postgres|couchdb" | grep -v rabbitmq | grep -v elasticsearch | grep -v openbao | grep -v redis || true)

    if [ -z "$volumes" ]; then
        print_status $BLUE "No Fabric volumes found"
    else
        if [ "$QUIET" != true ]; then
            echo "Found volumes:"
            echo "$volumes"
            echo ""
        fi

        # Remove volumes
        docker volume rm $volumes 2>/dev/null || true
        print_status $GREEN "✓ Volumes removed"
    fi
}

# Function to remove networks
remove_networks() {
    print_status $YELLOW "Removing networks..."

    # Remove fabric network
    if docker network ls --filter "name=${NETWORK_NAME}" --format "{{.Name}}" | grep -q "${NETWORK_NAME}"; then
        docker network rm "${NETWORK_NAME}" 2>/dev/null || true
        print_status $GREEN "✓ Network ${NETWORK_NAME} removed"
    else
        print_status $BLUE "No Fabric network found"
    fi
}

# Function to clean up artifacts
cleanup_artifacts() {
    if [ "$FULL_CLEANUP" != true ]; then
        print_status $YELLOW "Skipping artifact cleanup (use --full for complete cleanup)"
        return
    fi

    print_status $YELLOW "Cleaning up generated artifacts..."

    # Remove organizations directory
    if [ -d "${PROJECT_ROOT}/organizations" ]; then
        rm -rf "${PROJECT_ROOT}/organizations"
        print_status $GREEN "✓ Organizations directory removed"
    fi

    # Remove channel artifacts
    if [ -d "${PROJECT_ROOT}/config/channel-artifacts" ]; then
        rm -rf "${PROJECT_ROOT}/config/channel-artifacts"
        print_status $GREEN "✓ Channel artifacts removed"
    fi

    # Remove fabric-ca-server config
    if [ -d "${PROJECT_ROOT}/config/fabric-ca-server" ]; then
        rm -rf "${PROJECT_ROOT}/config/fabric-ca-server"
        print_status $GREEN "✓ fabric-ca-server removed"
    fi
    # Remove system-genesis-block
    if [ -f "${PROJECT_ROOT}/config/system-genesis-block" ]; then
        rm -f "${PROJECT_ROOT}/config/system-genesis-block"
        print_status $GREEN "✓ System genesis block removed"
    fi

    # Remove connection profiles
    if [ -f "${PROJECT_ROOT}/config/connection-profile.yaml" ]; then
        rm -f "${PROJECT_ROOT}/config/connection-profile.yaml"
        print_status $GREEN "✓ Connection profile removed"
    fi

    # Remove client crypto
    if [ -d "${PROJECT_ROOT}/client/crypto" ]; then
        rm -rf "${PROJECT_ROOT}/client/crypto"
        print_status $GREEN "✓ Client crypto removed"
    fi

    # Remove client wallet
    if [ -d "${PROJECT_ROOT}/client/wallet" ]; then
        rm -rf "${PROJECT_ROOT}/client/wallet"
        print_status $GREEN "✓ Client wallet removed"
    fi

    # Remove docker-compose generated files
    if [ -d "${PROJECT_ROOT}/docker-compose" ]; then
        find "${PROJECT_ROOT}/docker-compose" -name "*.yml" -delete 2>/dev/null || true
        print_status $GREEN "✓ Docker compose files removed"
    fi
}

# Function to stop client application
stop_client() {
    print_status $YELLOW "Stopping client application..."

    # Kill fabric-client process
    if pgrep -f "fabric-client" > /dev/null; then
        pkill -f "fabric-client"
        print_status $GREEN "✓ Client application stopped"
    else
        print_status $BLUE "Client application not running"
    fi

    # Remove client binary
    if [ -f "${PROJECT_ROOT}/client/fabric-client" ]; then
        rm -f "${PROJECT_ROOT}/client/fabric-client"
        print_status $GREEN "✓ Client binary removed"
    fi
}

# Function to clean up Docker images (optional)
cleanup_images() {
    if [ "$FULL_CLEANUP" != true ]; then
        return
    fi

    print_status $YELLOW "Optionally removing Fabric Docker images..."
    print_status $BLUE "Skipping image removal to speed up future deployments"
    print_status $BLUE "To remove images manually, run: docker rmi \$(docker images | grep fabric | awk '{print \$3}')"
}

# Function to clean up temporary files
cleanup_temp() {
    print_status $YELLOW "Cleaning up temporary files..."

    # Remove log files
    rm -f /tmp/fabric-client.log 2>/dev/null || true

    # Remove any temporary chaincode builds
    find "${PROJECT_ROOT}/chaincode" -name "*.tar.gz" -delete 2>/dev/null || true
    find "${PROJECT_ROOT}/chaincode" -type d -name "input" -exec rm -rf {} + 2>/dev/null || true
    find "${PROJECT_ROOT}/chaincode" -type d -name "output" -exec rm -rf {} + 2>/dev/null || true

    print_status $GREEN "✓ Temporary files cleaned"
}

# Function to create backup before cleanup (optional)
create_backup() {
    if [ "$FULL_CLEANUP" = true ]; then
        print_status $YELLOW "Creating backup of configuration files..."

        local backup_dir="${PROJECT_ROOT}/backups/backup-$(date +%Y%m%d-%H%M%S)"
        mkdir -p "$backup_dir"

        # Backup .env file
        cp "${PROJECT_ROOT}/.env" "$backup_dir/" 2>/dev/null || true

        # Backup chaincode
        cp -r "${PROJECT_ROOT}/chaincode" "$backup_dir/" 2>/dev/null || true

        # Backup client application
        cp -r "${PROJECT_ROOT}/client" "$backup_dir/" 2>/dev/null || true

        print_status $GREEN "✓ Backup created at $backup_dir"
        print_status $BLUE "To restore: cp -r $backup_dir/* ${PROJECT_ROOT}/"
    fi
}

# Function to show summary
show_summary() {
    print_status $GREEN "=== Teardown Summary ==="
    echo ""

    print_status $BLUE "Actions performed:"
    if [ "$FULL_CLEANUP" = true ]; then
        echo "  ✓ Stopped all containers"
        echo "  ✓ Removed all containers"
        echo "  ✓ Removed all volumes"
        echo "  ✓ Removed network"
        echo "  ✓ Cleaned up all artifacts"
        echo "  ✓ Created backup"
    else
        echo "  ✓ Stopped all containers"
        echo "  ✓ Removed all containers"
        echo "  ✓ Removed network"
        echo "  ✓ Kept volumes and data"
    fi

    echo ""
    print_status $YELLOW "Remaining items:"

    if [ "$FULL_CLEANUP" != true ]; then
        echo "  • Docker volumes (use --full to remove)"
        echo "  • Generated certificates and artifacts"
        echo "  • Channel configuration files"
    fi

    echo "  • Docker images (kept for faster redeployment)"
    echo "  • Configuration files (.env, scripts)"
    echo "  • Chaincode source code"
    echo "  • Client application source code"
    echo ""

    print_status $BLUE "To completely reset the network:"
    echo "  $0 --full"
    echo ""

    print_status $BLUE "To keep data but stop containers:"
    echo "  $0 --keep-data"
    echo ""

    print_status $BLUE "To restart the network:"
    echo "  ./scripts/deployment/001-deploy-postgres.sh"
    echo "  (then follow with other scripts)"
    echo ""
}

# Main teardown process
print_status $YELLOW "Warning: This will stop and remove Fabric network components"
if [ "$QUIET" != true ]; then
    read -p "Are you sure you want to continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status $RED "Teardown cancelled"
        exit 0
    fi
fi

# Create backup before full cleanup
if [ "$FULL_CLEANUP" = true ]; then
    create_backup
fi

# Stop client application first
stop_client

# Stop and remove containers
stop_containers

# Remove networks
remove_networks

# Remove volumes
remove_volumes

# Clean up artifacts
cleanup_artifacts

# Clean up temporary files
cleanup_temp

# Optionally clean up images
cleanup_images

# Show summary
show_summary

print_status $GREEN "=== Teardown Completed Successfully ==="
