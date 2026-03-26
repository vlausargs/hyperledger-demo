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
export PEER_ENDPOINT="peer0.org1.example.com:8051"
export GATEWAY_PEER="peer0.org1.example.com"
export MSP_ID="Org1MSP"
export USER_ID="user1"
export CONNECTION_PROFILE="./crypto/connection-profile.yaml"
export ORG_NAME="Org1"

# Start application
./fabric-client
