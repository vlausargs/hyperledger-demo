#!/bin/bash
# Script to download Hyperledger Fabric binaries using versions from .env file
# This script downloads Fabric platform-specific binaries to the fabric-binaries folder

set -e

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Path to .env file
ENV_FILE="${PROJECT_ROOT}/.env"

# Check if .env file exists
if [ ! -f "${ENV_FILE}" ]; then
    echo "Error: .env file not found at ${ENV_FILE}"
    exit 1
fi

# Source the .env file to get version variables
echo "Loading environment variables from ${ENV_FILE}..."
set -a
source "${ENV_FILE}"
set +a

# Check if required version variables are set
if [ -z "${FABRIC_VERSION}" ]; then
    echo "Error: FABRIC_VERSION is not set in .env file"
    exit 1
fi

if [ -z "${CA_VERSION}" ]; then
    echo "Error: CA_VERSION is not set in .env file"
    exit 1
fi

# Display versions to be downloaded
echo "=========================================="
echo "Downloading Hyperledger Fabric Binaries"
echo "=========================================="
echo "Fabric Version: ${FABRIC_VERSION}"
echo "Fabric CA Version: ${CA_VERSION}"
echo "Target Directory: ${PROJECT_ROOT}/fabric-binaries"
echo "=========================================="

# Change to project root directory for download
cd "${PROJECT_ROOT}"

# Download Fabric binaries
echo "Starting download..."
# Check if download was successful
if [ ! -d "${PROJECT_ROOT}/fabric-binaries" ]; then
    echo "fabric-binaries directory not created"
    mkdir  -p "${PROJECT_ROOT}/fabric-binaries/bin"
    mkdir  -p "${PROJECT_ROOT}/fabric-binaries/builders"
    mkdir  -p "${PROJECT_ROOT}/fabric-binaries/config"
    # exit 1
fi
cd "${PROJECT_ROOT}/fabric-binaries"
curl -sSL https://raw.githubusercontent.com/hyperledger/fabric/master/scripts/bootstrap.sh | bash -s -- $FABRIC_VERSION $CA_VERSION -s -d


# Verify key binaries exist
REQUIRED_BINARIES=("configtxgen" "cryptogen" "peer" "orderer" "fabric-ca-client")
MISSING_BINARIES=0

for binary in "${REQUIRED_BINARIES[@]}"; do
    if [ ! -f "${PROJECT_ROOT}/fabric-binaries/bin/${binary}" ]; then
        echo "Warning: ${binary} not found in fabric-binaries/bin"
        MISSING_BINARIES=$((MISSING_BINARIES + 1))
    else
        echo "✓ Found ${binary}"
    fi
done

if [ ${MISSING_BINARIES} -gt 0 ]; then
    echo ""
    echo "Warning: ${MISSING_BINARIES} required binary/binaries may be missing"
else
    echo ""
    echo "=========================================="
    echo "✓ All required binaries downloaded successfully!"
    echo "=========================================="
fi

# Display download location
echo ""
echo "Binaries are located at: ${PROJECT_ROOT}/fabric-binaries/bin"
echo "Config files are located at: ${PROJECT_ROOT}/fabric-binaries/config"
echo ""
echo "You can now use these binaries with your Fabric network."
