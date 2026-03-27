#!/bin/bash
# Quick start script for Fabric Client

cd "$(dirname "$0")"

# Set environment variables
export WALLET_PATH="./wallet"
export CHANNEL_ID="mychannel"
export CHAINCODE_ID="basic"
export SERVER_PORT="8080"
export GIN_MODE="debug"
export TLS_CERT_PATH="./crypto"
export CONNECTION_PROFILE="./crypto/connection-profile.yaml"

# Start application
./fabric-client
