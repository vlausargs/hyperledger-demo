# Hyperledger Fabric Network Deployment Guide

This guide provides step-by-step instructions for deploying a production-ready Hyperledger Fabric network with the following features:

- 2 Organizations (Org1, Org2) with 1 peer each
- 1 Orderer service
- Fabric Certificate Authority (CA) with PostgreSQL for identity management
- CouchDB for world state database (1 CouchDB per peer)
- Go chaincode support
- Go client application with Gin framework
- Docker-based deployment
- Modular deployment across multiple machines

## Table of Contents

1. [System Overview](#system-overview)
2. [Prerequisites](#prerequisites)
3. [Initial Setup](#initial-setup)
4. [Single Machine Deployment](#single-machine-deployment)
5. [Modular Multi-Machine Deployment](#modular-multi-machine-deployment)
6. [Network Operations](#network-operations)
7. [Client Application Usage](#client-application-usage)
8. [Troubleshooting](#troubleshooting)
9. [Maintenance](#maintenance)

## System Overview

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   Hyperledger Fabric Network                  │
├─────────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐│
│  │   Machine A │      │   Machine B │      │   Machine C ││
│  │  (Orderer)  │      │   (Org1)    │      │   (Org2)    ││
│  └─────────────┘      └─────────────┘      └─────────────┘│
│         │                   │                   │           │
│         │                   │                   │           │
│  ┌──────▼────────┐  ┌──▼──────┐     ┌─────▼────────┐│
│  │    Orderer    │  │  Org1   │     │    Org2      ││
│  │   Service     │  │   Peer  │     │    Peer      ││
│  │    + CA      │  │   + CA  │     │    + CA      ││
│  │  + PostgreSQL│  │+CouchDB │     │  + CouchDB   ││
│  │              │  │+PostgreSQL│   │   + PostgreSQL││
│  └───────────────┘  └─────────┘     └──────────────┘│
│         │                   │                   │           │
│         └───────────────────┴───────────────────┘           │
│                    Fabric Network Channel                    │
└─────────────────────────────────────────────────────────────────┘
```

### Components

#### Orderer Machine (Machine A)
- Orderer Service (Raft consensus)
- Orderer CA
- PostgreSQL Database (for Orderer CA)
- Network: Orderer Organization

#### Organization 1 Machine (Machine B)
- Peer0
- CouchDB (world state database)
- Org1 CA
- PostgreSQL Database (for Org1 CA)
- Network: Org1 Organization

#### Organization 2 Machine (Machine C)
- Peer0
- CouchDB (world state database)
- Org2 CA
- PostgreSQL Database (for Org2 CA)
- Network: Org2 Organization

## Prerequisites

### Required Software

Before deploying the network, ensure you have the following software installed:

#### Docker & Docker Compose
```bash
# Check Docker version
docker --version  # Should be 20.10 or higher

# Check Docker Compose version
docker-compose --version  # Should be 2.0 or higher
```

#### Go (for client application)
```bash
# Check Go version
go version  # Should be 1.19 or higher
```

#### Additional Tools
```bash
# OpenSSL (for certificate operations)
openssl version

# Curl (for API testing)
curl --version

# JQ (for JSON processing)
jq --version
```

### System Requirements

- **Minimum**:
  - CPU: 2 cores
  - RAM: 4 GB
  - Disk: 20 GB
  - Network: 100 Mbps

- **Recommended** (Production):
  - CPU: 4 cores or more
  - RAM: 8 GB or more
  - Disk: 50 GB or more
  - Network: 1 Gbps
  - SSD storage

### Network Configuration

Ensure that the following ports are available on each machine:

#### Orderer Machine
- 7050: Orderer service
- 7051: Orderer cluster communication
- 7053: Orderer admin port
- 7054: Orderer CA
- 5432: PostgreSQL for Orderer CA

#### Org1 Machine
- 7051: Peer service
- 7052: Peer chaincode
- 8054: Org1 CA
- 5984: CouchDB for Org1
- 5432: PostgreSQL for Org1 CA

#### Org2 Machine
- 9051: Peer service
- 9052: Peer chaincode
- 9054: Org2 CA
- 7984: CouchDB for Org2
- 5432: PostgreSQL for Org2 CA

#### Client Application
- 8080: REST API server

### Firewall Configuration

Open the following ports for inter-machine communication:

```bash
# Orderer Machine
sudo ufw allow 7050/tcp
sudo ufw allow 7051/tcp
sudo ufw allow 7053/tcp
sudo ufw allow 7054/tcp

# Org1 Machine
sudo ufw allow 7051/tcp
sudo ufw allow 7052/tcp
sudo ufw allow 8054/tcp
sudo ufw allow 5984/tcp

# Org2 Machine
sudo ufw allow 9051/tcp
sudo ufw allow 9052/tcp
sudo ufw allow 9054/tcp
sudo ufw allow 7984/tcp
```

## Initial Setup

### 1. Clone or Copy Project Files

```bash
# Navigate to project directory
cd /path/to/your/workspace

# Copy project files
cp -r /path/to/fabric-network ./

cd fabric-network
```

### 2. Configure Environment

Edit the `.env` file to configure your deployment:

```bash
nano .env
```

#### Key Configuration Sections

**Deployment Control**
```bash
# For single machine deployment (all components)
DEPLOY_ORDERER=true
DEPLOY_ORG1=true
DEPLOY_ORG2=true

# For modular deployment (customize per machine)
# Machine A (Orderer):
# DEPLOY_ORDERER=true
# DEPLOY_ORG1=false
# DEPLOY_ORG2=false
```

**External Host Configuration**
```bash
# Update these with your actual hostnames/IPs
ORDERER_EXTERNAL_HOST=orderer.yourdomain.com
ORG1_EXTERNAL_HOST=org1.yourdomain.com
ORG2_EXTERNAL_HOST=org2.yourdomain.com
```

**Security Configuration**
```bash
# Change all passwords to strong values
POSTGRES_ORDERER_PASSWORD=your_strong_password_here
POSTGRES_ORG1_PASSWORD=your_strong_password_here
POSTGRES_ORG2_PASSWORD=your_strong_password_here

CA_ORDERER_ENROLLMENT_SECRET=your_ca_password_here
CA_ORG1_ENROLLMENT_SECRET=your_ca_password_here
CA_ORG2_ENROLLMENT_SECRET=your_ca_password_here
```

### 3. Verify Configuration

```bash
# Check environment variables
source .env
echo "Orderer: $DEPLOY_ORDERER"
echo "Org1: $DEPLOY_ORG1"
echo "Org2: $DEPLOY_ORG2"
```

### 4. Prepare Scripts

```bash
# Make all deployment scripts executable
chmod +x scripts/deployment/*.sh

# Verify scripts are present
ls -la scripts/deployment/
```

## Single Machine Deployment

For deploying all components on a single machine (development/testing):

### Step 1: Deploy PostgreSQL Databases

```bash
./scripts/deployment/001-deploy-postgres.sh
```

**Expected Output:**
```
=== Starting PostgreSQL Deployment ===
✓ Orderer PostgreSQL deployed successfully
✓ Org1 PostgreSQL deployed successfully
✓ Org2 PostgreSQL deployed successfully
✓ postgres-orderer is ready
✓ postgres-org1 is ready
✓ postgres-org2 is ready
```

**Verification:**
```bash
# Check running PostgreSQL containers
docker ps | grep postgres

# Verify databases
docker exec postgres-orderer psql -U postgres -d fabric_ca_orderer -c "SELECT version();"
```

### Step 2: Deploy Fabric CA Services

```bash
./scripts/deployment/002-deploy-ca.sh
```

**Expected Output:**
```
=== Starting Fabric CA Deployment ===
✓ Orderer CA deployed successfully
✓ Org1 CA deployed successfully
✓ Org2 CA deployed successfully
✓ ca-orderer is ready
✓ ca-org1 is ready
✓ ca-org2 is ready
```

**Verification:**
```bash
# Check running CA containers
docker ps | grep ca

# Test CA endpoints
curl http://localhost:7054/cainfo
curl http://localhost:8054/cainfo
curl http://localhost:9054/cainfo
```

### Step 3: Setup CA Identities

```bash
./scripts/deployment/003-setup-ca.sh
```

**Expected Output:**
```
=== Starting CA Identity Setup ===
=== Setting up Orderer CA Identities ===
✓ Identity ca-admin setup completed
✓ Identity orderer1 setup completed
=== Setting up Org1 CA Identities ===
✓ Identity admin setup completed
✓ Identity peer0 setup completed
✓ Identity user1 setup completed
=== Setting up Org2 CA Identities ===
✓ Identity admin setup completed
✓ Identity peer0 setup completed
✓ Identity user1 setup completed
```

**Verification:**
```bash
# Check generated certificates
ls -la organizations/ordererOrganizations/orderer.example.com/orderers/
ls -la organizations/peerOrganizations/org1.example.com/peers/
ls -la organizations/peerOrganizations/org2.example.com/peers/
```

### Step 4: Generate Channel Configuration

```bash
./scripts/deployment/004-generate-configtx.sh
```

**Expected Output:**
```
=== Starting Channel Configuration Generation ===
✓ configtx.yaml created successfully
✓ Genesis block generated successfully
✓ Channel configuration transaction generated successfully
✓ Org1 anchor peer update generated successfully
✓ Org2 anchor peer update generated successfully
```

**Verification:**
```bash
# Check generated artifacts
ls -lh config/channel-artifacts/

# Verify genesis block
file config/channel-artifacts/genesis.block
```

### Step 5: Deploy Orderer

```bash
./scripts/deployment/005-deploy-orderer.sh
```

**Expected Output:**
```
=== Starting Orderer Deployment ===
✓ Orderer configuration created
✓ Orderer service deployed
✓ Orderer is ready
✓ Orderer is running (Version: 2.4.7)
✓ Orderer listening on port 7050
✓ Orderer listening on port 7051 (cluster)
```

**Verification:**
```bash
# Check orderer container
docker ps | grep orderer

# Check orderer logs
docker logs orderer.example.com

# Check orderer metrics
curl http://localhost:8443/metrics
```

### Step 6: Deploy Peers

```bash
./scripts/deployment/006-deploy-peers.sh
```

**Expected Output:**
```
=== Starting Peer Deployment ===
=== Deploying peer0.org1.example.com ===
✓ peer0.org1.example.com deployed
✓ peer0.org1.example.com is ready
✓ peer0.org1.example.com is running (Version: 2.4.7)
✓ peer0.org1.example.com connected to CouchDB
=== Deploying peer0.org2.example.com ===
✓ peer0.org2.example.com deployed
✓ peer0.org2.example.com is ready
✓ peer0.org2.example.com is running (Version: 2.4.7)
✓ peer0.org2.example.com connected to CouchDB
```

**Verification:**
```bash
# Check peer containers
docker ps | grep peer

# Check CouchDB containers
docker ps | grep couchdb

# Check CouchDB UI
curl http://localhost:5984/_utils
curl http://localhost:7984/_utils
```

### Step 7: Create and Join Channel

```bash
./scripts/deployment/007-create-channel.sh
```

**Expected Output:**
```
=== Starting Channel Creation and Join ===
Creating channel 'mychannel'...
✓ Channel 'mychannel' created successfully
Joining peer0.org1.example.com to channel 'mychannel'...
✓ peer0.org1.example.com joined channel 'mychannel' successfully
Joining peer0.org2.example.com to channel 'mychannel'...
✓ peer0.org2.example.com joined channel 'mychannel' successfully
```

**Verification:**
```bash
# List channels
docker exec peer0.org1.example.com peer channel list
docker exec peer0.org2.example.com peer channel list

# Get channel info
docker exec peer0.org1.example.com peer channel getinfo -c mychannel
```

### Step 8: Deploy Chaincode

```bash
./scripts/deployment/008-deploy-chaincode.sh
```

**Expected Output:**
```
=== Starting Chaincode Deployment ===
Copying chaincode to peer containers...
Packaging chaincode on peer0.org1.example.com...
✓ Chaincode packaged successfully on peer0.org1.example.com
Installing chaincode on peer0.org1.example.com...
✓ Chaincode installed successfully on peer0.org1.example.com
Package ID: basic_1.0:abc123def456...
✓ Chaincode approved successfully for Org1MSP
✓ Chaincode approved successfully for Org2MSP
✓ Chaincode committed successfully to channel mychannel
Initializing chaincode ledger...
✓ Chaincode initialized successfully
```

**Verification:**
```bash
# Query all assets (should return initial assets)
docker exec peer0.org1.example.com peer chaincode query \
  -C mychannel -n basic -c '{"Args":["GetAllAssets"]}' \
  --tls --cafile /etc/hyperledger/fabric/tls/ca.crt
```

### Step 9: Start Client Application

```bash
./scripts/deployment/009-start-client.sh
```

**Expected Output:**
```
=== Starting Fabric Client Application ===
✓ Go version: go1.19
✓ Prerequisites verified
✓ Dependencies downloaded
✓ Client application built successfully
✓ Client application started successfully (PID: 12345)
✓ Server is ready and listening on port 8080
✓ Health check passed
```

**Verification:**
```bash
# Test health endpoint
curl http://localhost:8080/health

# Test API endpoints
curl http://localhost:8080/api/v1/assets
```

## Modular Multi-Machine Deployment

For deploying components across multiple machines (production):

### Machine A: Orderer Deployment

#### 1. Configure Environment

```bash
cd fabric-network
nano .env
```

```bash
# Only deploy orderer
DEPLOY_ORDERER=true
DEPLOY_ORG1=false
DEPLOY_ORG2=false

# Update external host configuration
ORDERER_EXTERNAL_HOST=orderer.yourdomain.com
ORG1_EXTERNAL_HOST=org1.yourdomain.com
ORG2_EXTERNAL_HOST=org2.yourdomain.com
```

#### 2. Deploy Orderer Components

```bash
# Deploy PostgreSQL for Orderer CA
./scripts/deployment/001-deploy-postgres.sh

# Deploy Orderer CA
./scripts/deployment/002-deploy-ca.sh

# Setup Orderer CA identities
./scripts/deployment/003-setup-ca.sh

# Generate channel configuration
./scripts/deployment/004-generate-configtx.sh

# Deploy Orderer
./scripts/deployment/005-deploy-orderer.sh
```

#### 3. Verify Orderer Deployment

```bash
# Check all orderer containers
docker ps | grep -E "orderer|postgres"

# Verify orderer is listening
netstat -tlnp | grep 7050
```

### Machine B: Org1 Deployment

#### 1. Configure Environment

```bash
cd fabric-network
nano .env
```

```bash
# Only deploy Org1
DEPLOY_ORDERER=false
DEPLOY_ORG1=true
DEPLOY_ORG2=false

# Update external host configuration
ORDERER_EXTERNAL_HOST=orderer.yourdomain.com
ORG1_EXTERNAL_HOST=org1.yourdomain.com
ORG2_EXTERNAL_HOST=org2.yourdomain.com
```

#### 2. Copy Channel Configuration

```bash
# From Machine A (Orderer machine)
scp config/channel-artifacts/* user@machine-b:/path/to/fabric-network/config/channel-artifacts/
```

#### 3. Deploy Org1 Components

```bash
# Deploy PostgreSQL for Org1 CA
./scripts/deployment/001-deploy-postgres.sh

# Deploy Org1 CA
./scripts/deployment/002-deploy-ca.sh

# Setup Org1 CA identities
./scripts/deployment/003-setup-ca.sh

# Deploy Org1 peer
./scripts/deployment/006-deploy-peers.sh
```

#### 4. Join Channel

```bash
# Join Org1 peer to channel
./scripts/deployment/007-create-channel.sh
```

#### 5. Verify Org1 Deployment

```bash
# Check all Org1 containers
docker ps | grep -E "peer|couchdb|postgres|ca"

# Verify peer is joined to channel
docker exec peer0.org1.example.com peer channel list
```

### Machine C: Org2 Deployment

#### 1. Configure Environment

```bash
cd fabric-network
nano .env
```

```bash
# Only deploy Org2
DEPLOY_ORDERER=false
DEPLOY_ORG1=false
DEPLOY_ORG2=true

# Update external host configuration
ORDERER_EXTERNAL_HOST=orderer.yourdomain.com
ORG1_EXTERNAL_HOST=org1.yourdomain.com
ORG2_EXTERNAL_HOST=org2.yourdomain.com
```

#### 2. Copy Channel Configuration

```bash
# From Machine A (Orderer machine)
scp config/channel-artifacts/* user@machine-c:/path/to/fabric-network/config/channel-artifacts/
```

#### 3. Deploy Org2 Components

```bash
# Deploy PostgreSQL for Org2 CA
./scripts/deployment/001-deploy-postgres.sh

# Deploy Org2 CA
./scripts/deployment/002-deploy-ca.sh

# Setup Org2 CA identities
./scripts/deployment/003-setup-ca.sh

# Deploy Org2 peer
./scripts/deployment/006-deploy-peers.sh
```

#### 4. Join Channel

```bash
# Join Org2 peer to channel
./scripts/deployment/007-create-channel.sh
```

#### 5. Verify Org2 Deployment

```bash
# Check all Org2 containers
docker ps | grep -E "peer|couchdb|postgres|ca"

# Verify peer is joined to channel
docker exec peer0.org2.example.com peer channel list
```

### Chaincode Deployment (From Any Machine)

After all peers are deployed and joined to the channel, deploy chaincode:

```bash
# Deploy chaincode (can be run from any machine with access to peers)
./scripts/deployment/008-deploy-chaincode.sh
```

### Client Application Deployment (Optional)

Deploy the client application on a separate machine or any machine with peer access:

```bash
# Start client application
./scripts/deployment/009-start-client.sh
```

## Network Operations

### Container Management

#### View All Containers

```bash
# All Fabric containers
docker ps | grep -E "orderer|peer|ca|couchdb|postgres"

# Specific service containers
docker ps | grep orderer
docker ps | grep peer
docker ps | grep ca
```

#### View Container Logs

```bash
# Orderer logs
docker logs -f orderer.example.com

# Peer logs
docker logs -f peer0.org1.example.com
docker logs -f peer0.org2.example.com

# CA logs
docker logs -f ca.orderer.example.com
docker logs -f ca.org1.example.com
docker logs -f ca.org2.example.com

# CouchDB logs
docker logs -f couchdb.org1.example.com
docker logs -f couchdb.org2.example.com
```

#### Restart Services

```bash
# Restart orderer
docker restart orderer.example.com

# Restart peer
docker restart peer0.org1.example.com

# Restart CA
docker restart ca.org1.example.com
```

### Channel Operations

#### List Channels

```bash
# From Org1
docker exec peer0.org1.example.com peer channel list \
  --tls --cafile /etc/hyperledger/fabric/tls/ca.crt

# From Org2
docker exec peer0.org2.example.com peer channel list \
  --tls --cafile /etc/hyperledger/fabric/tls/ca.crt
```

#### Get Channel Information

```bash
docker exec peer0.org1.example.com peer channel getinfo \
  -c mychannel \
  --tls --cafile /etc/hyperledger/fabric/tls/ca.crt
```

#### Fetch Channel Block

```bash
docker exec peer0.org1.example.com peer channel fetch 0 \
  mychannel.block \
  -c mychannel \
  --tls --cafile /etc/hyperledger/fabric/tls/ca.crt
```

### Chaincode Operations

#### Query Chaincode

```bash
# Get all assets
docker exec peer0.org1.example.com peer chaincode query \
  -C mychannel \
  -n basic \
  -c '{"Args":["GetAllAssets"]}' \
  --tls --cafile /etc/hyperledger/fabric/tls/ca.crt

# Read specific asset
docker exec peer0.org1.example.com peer chaincode query \
  -C mychannel \
  -n basic \
  -c '{"Args":["ReadAsset","asset1"]}' \
  --tls --cafile /etc/hyperledger/fabric/tls/ca.crt
```

#### Invoke Chaincode

```bash
# Create new asset
docker exec peer0.org1.example.com peer chaincode invoke \
  -o orderer.example.com:7050 \
  -C mychannel \
  -n basic \
  -c '{"Args":["CreateAsset","asset7","purple",20,"Owner",800]}' \
  --tls \
  --cafile /etc/hyperledger/fabric/tls/ca.crt \
  --peerAddresses peer0.org1.example.com:7051 \
  --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt \
  --peerAddresses peer0.org2.example.com:9051 \
  --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt

# Transfer asset
docker exec peer0.org1.example.com peer chaincode invoke \
  -o orderer.example.com:7050 \
  -C mychannel \
  -n basic \
  -c '{"Args":["TransferAsset","asset1","NewOwner"]}' \
  --tls \
  --cafile /etc/hyperledger/fabric/tls/ca.crt \
  --peerAddresses peer0.org1.example.com:7051 \
  --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt \
  --peerAddresses peer0.org2.example.com:9051 \
  --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt
```

#### Upgrade Chaincode

```bash
# Install new version
docker exec peer0.org1.example.com peer lifecycle chaincode install \
  /etc/hyperledger/fabric/chaincode/basic_v2.tar.gz \
  --tls \
  --cafile /etc/hyperledger/fabric/tls/ca.crt

# Approve new version
docker exec peer0.org1.example.com peer lifecycle chaincode approveformyorg \
  -o orderer.example.com:7050 \
  --channelID mychannel \
  --name basic \
  --version 2.0 \
  --package-id <new-package-id> \
  --sequence 2 \
  --tls \
  --cafile /etc/hyperledger/fabric/tls/ca.crt

# Commit new version
docker exec peer0.org1.example.com peer lifecycle chaincode commit \
  -o orderer.example.com:7050 \
  --channelID mychannel \
  --name basic \
  --version 2.0 \
  --sequence 2 \
  --tls \
  --cafile /etc/hyperledger/fabric/tls/ca.crt \
  --peerAddresses peer0.org1.example.com:7051 \
  --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt \
  --peerAddresses peer0.org2.example.com:9051 \
  --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt
```

### CouchDB Operations

#### Access CouchDB UI

```bash
# Org1 CouchDB
http://localhost:5984/_utils
Username: admin
Password: org1_couchdb_password_change-this

# Org2 CouchDB
http://localhost:7984/_utils
Username: admin
Password: org2_couchdb_password_change-this
```

#### Query CouchDB Database

```bash
# List all databases
curl -u admin:password http://localhost:5984/_all_dbs

# Get documents from chaincode database
curl -u admin:password http://localhost:5984/mychannel_basic/_all_docs
```

## Client Application Usage

### API Endpoints

The client application provides a REST API for interacting with the blockchain network.

#### Base URL

```
http://localhost:8080/api/v1
```

#### Health Check

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "Fabric Client API",
  "version": "1.0.0"
}
```

### Asset Operations

#### Get All Assets

```bash
curl http://localhost:8080/api/v1/assets
```

**Response:**
```json
{
  "assets": [
    {
      "ID": "asset1",
      "color": "blue",
      "size": 5,
      "owner": "Tomoko",
      "appraisedValue": 300
    }
  ],
  "count": 1
}
```

#### Get Specific Asset

```bash
curl http://localhost:8080/api/v1/assets/asset1
```

**Response:**
```json
{
  "ID": "asset1",
  "color": "blue",
  "size": 5,
  "owner": "Tomoko",
  "appraisedValue": 300
}
```

#### Create New Asset

```bash
curl -X POST http://localhost:8080/api/v1/assets \
  -H "Content-Type: application/json" \
  -d '{
    "ID": "asset10",
    "color": "green",
    "size": 10,
    "owner": "Alice",
    "appraisedValue": 500
  }'
```

**Response:**
```json
{
  "message": "Asset created successfully",
  "assetID": "asset10"
}
```

#### Update Asset

```bash
curl -X PUT http://localhost:8080/api/v1/assets/asset10 \
  -H "Content-Type: application/json" \
  -d '{
    "color": "red",
    "size": 15,
    "owner": "Bob",
    "appraisedValue": 600
  }'
```

**Response:**
```json
{
  "message": "Asset updated successfully",
  "assetID": "asset10"
}
```

#### Delete Asset

```bash
curl -X DELETE http://localhost:8080/api/v1/assets/asset10
```

**Response:**
```json
{
  "message": "Asset deleted successfully",
  "assetID": "asset10"
}
```

#### Transfer Asset

```bash
curl -X POST http://localhost:8080/api/v1/assets/asset1/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "newOwner": "Charlie"
  }'
```

**Response:**
```json
{
  "message": "Asset transferred successfully",
  "assetID": "asset1",
  "newOwner": "Charlie"
}
```

#### Get Asset History

```bash
curl http://localhost:8080/api/v1/assets/asset1/history
```

**Response:**
```json
{
  "assetID": "asset1",
  "history": [
    {
      "txId": "tx1",
      "timestamp": "2023-01-01T00:00:00Z",
      "isDelete": false,
      "record": {
        "ID": "asset1",
        "color": "blue",
        "size": 5,
        "owner": "Tomoko",
        "appraisedValue": 300
      }
    }
  ]
}
```

### Network Information

#### Get Channels

```bash
curl http://localhost:8080/api/v1/channels
```

**Response:**
```json
{
  "channels": [
    {
      "channel_id": "mychannel",
      "status": "active"
    }
  ]
}
```

#### Get Peers

```bash
curl http://localhost:8080/api/v1/network/peers
```

**Response:**
```json
{
  "peers": [
    {
      "name": "peer0.org1.example.com",
      "address": "peer0.org1.example.com:7051",
      "status": "online",
      "org": "Org1"
    },
    {
      "name": "peer0.org2.example.com",
      "address": "peer0.org2.example.com:9051",
      "status": "online",
      "org": "Org2"
    }
  ]
}
```

### Using Test Script

The project includes a comprehensive test script for API endpoints:

```bash
cd client
./test-api.sh
```

This script will:
1. Check health endpoint
2. Get all assets
3. Create a new asset
4. Get specific asset
5. Update asset
6. Transfer asset
7. Get asset history

## Troubleshooting

### Common Issues and Solutions

#### 1. CA Service Not Starting

**Symptoms:**
- CA container fails to start
- Error: "Failed to connect to database"

**Solutions:**
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Verify PostgreSQL credentials
docker exec postgres-orderer psql -U postgres -d fabric_ca_orderer

# Check CA logs
docker logs ca.orderer.example.com

# Restart CA
docker restart ca.orderer.example.com
```

#### 2. Peer Cannot Join Channel

**Symptoms:**
- Peer fails to join channel
- Error: "Proposal was rejected"

**Solutions:**
```bash
# Verify peer certificates
ls -la organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/

# Check channel configuration
ls -la config/channel-artifacts/

# Verify peer is running
docker ps | grep peer

# Check peer logs
docker logs peer0.org1.example.com

# Retry joining channel
docker exec peer0.org1.example.com peer channel join \
  -b /etc/hyperledger/fabric/channel-artifacts/mychannel.block \
  --tls \
  --cafile /etc/hyperledger/fabric/tls/ca.crt
```

#### 3. Chaincode Not Working

**Symptoms:**
- Chaincode invoke/query fails
- Error: "Chaincode definition not found"

**Solutions:**
```bash
# Check chaincode is installed
docker exec peer0.org1.example.com peer lifecycle chaincode queryinstalled

# Check chaincode is committed
docker exec peer0.org1.example.com peer lifecycle chaincode querycommitted \
  -C mychannel

# Verify chaincode container
docker ps | grep dev-peer

# Check chaincode logs
docker logs dev-peer0.org1.example.com-basic_1.0-*

# Redeploy chaincode
./scripts/deployment/008-deploy-chaincode.sh
```

#### 4. CouchDB Connection Issues

**Symptoms:**
- Peer fails to start
- Error: "Failed to connect to CouchDB"

**Solutions:**
```bash
# Check CouchDB is running
docker ps | grep couchdb

# Test CouchDB connection
curl http://localhost:5984/_up

# Check CouchDB logs
docker logs couchdb.org1.example.com

# Verify peer configuration
docker inspect peer0.org1.example.com | grep -A 5 CORE_LEDGER_STATE

# Restart peer
docker restart peer0.org1.example.com
```

#### 5. Network Connectivity Issues

**Symptoms:**
- Components on different machines cannot communicate
- Connection timeout errors

**Solutions:**
```bash
# Check firewall rules
sudo ufw status

# Test network connectivity
ping orderer.yourdomain.com
telnet orderer.yourdomain.com 7050

# Check DNS resolution
nslookup orderer.yourdomain.com

# Verify external host configuration
grep EXTERNAL_HOST .env

# Check container network
docker network inspect fabric_network
```

#### 6. Client Application Not Starting

**Symptoms:**
- Client application fails to build
- API server not responding

**Solutions:**
```bash
# Check Go installation
go version

# Download dependencies
cd client
go mod download

# Build application
go build -o fabric-client main.go

# Check logs
tail -f /tmp/fabric-client.log

# Test API manually
curl http://localhost:8080/health

# Restart client
pkill -f fabric-client
./scripts/deployment/009-start-client.sh
```

### Log Analysis

#### Analyzing Orderer Logs

```bash
# View recent logs
docker logs --tail 100 orderer.example.com

# Follow logs
docker logs -f orderer.example.com

# Search for errors
docker logs orderer.example.com 2>&1 | grep -i error

# Search for warnings
docker logs orderer.example.com 2>&1 | grep -i warning
```

#### Analyzing Peer Logs

```bash
# View recent logs
docker logs --tail 100 peer0.org1.example.com

# Search for specific transactions
docker logs peer0.org1.example.com 2>&1 | grep "TxId: abc123"

# Check for gRPC errors
docker logs peer0.org1.example.com 2>&1 | grep -i grpc
```

#### Analyzing Chaincode Logs

```bash
# Find chaincode container
docker ps | grep dev-peer

# View chaincode logs
docker logs dev-peer0.org1.example.com-basic_1.0-abc123

# Check for runtime errors
docker logs dev-peer0.org1.example.com-basic_1.0-abc123 2>&1 | grep panic
```

### Reset and Redeploy

#### Partial Reset (Keep Data)

```bash
# Stop containers
docker stop $(docker ps -q)

# Remove containers (keep volumes)
docker rm $(docker ps -aq)

# Restart services
./scripts/deployment/005-deploy-orderer.sh
./scripts/deployment/006-deploy-peers.sh
```

#### Full Reset (Remove All)

```bash
# Use teardown script
./scripts/deployment/999-teardown.sh --full

# Or manual cleanup
docker stop $(docker ps -q)
docker rm $(docker ps -aq)
docker volume rm $(docker volume ls -q)
docker network rm fabric_network

# Remove artifacts
rm -rf organizations/
rm -rf config/channel-artifacts/
```

## Maintenance

### Backup Procedures

#### Backup Certificates

```bash
# Create backup directory
mkdir -p backups/$(date +%Y%m%d)

# Backup certificates
tar -czf backups/$(date +%Y%m%d)/certificates.tar.gz organizations/

# Backup configuration
cp .env backups/$(date +%Y%m%d)/
cp config/channel-artifacts/*.block backups/$(date +%Y%m%d)/
cp config/channel-artifacts/*.tx backups/$(date +%Y%m%d)/
```

#### Backup Chaincode

```bash
# Backup chaincode source
tar -czf backups/$(date +%Y%m%d)/chaincode.tar.gz chaincode/

# Backup chaincode packages
find organizations/chaincode -name "*.tar.gz" -exec cp {} backups/$(date +%Y%m%d)/ \;
```

#### Backup Database Data

```bash
# Backup PostgreSQL data
docker exec postgres-orderer pg_dump -U postgres fabric_ca_orderer > backups/$(date +%Y%m%d)/orderer_ca.sql
docker exec postgres-org1 pg_dump -U postgres fabric_ca_org1 > backups/$(date +%Y%m%d)/org1_ca.sql
docker exec postgres-org2 pg_dump -U postgres fabric_ca_org2 > backups/$(date +%Y%m%d)/org2_ca.sql

# Backup CouchDB data
docker exec couchdb.org1.example.com curl -X GET http://admin:password@localhost:5984/_all_dbs > backups/$(date +%Y%m%d)/org1_couchdb_list.txt
```

### Monitoring

#### System Monitoring

```bash
# Monitor container resource usage
docker stats

# Monitor disk usage
df -h

# Monitor Docker volumes
docker system df
```

#### Network Monitoring

```bash
# Check network connections
netstat -tlnp | grep -E "7050|7051|9051|9052"

# Monitor network traffic
iftop -i docker0

# Test latency
ping -c 10 orderer.example.com
```

### Updates and Upgrades

#### Update Hyperledger Fabric Version

```bash
# Edit .env file
nano .env

# Update version
FABRIC_VERSION=2.5.0
CA_VERSION=1.5.6

# Rebuild network
./scripts/deployment/999-teardown.sh
./scripts/deployment/001-deploy-postgres.sh
# ... continue with other scripts
```

#### Update Chaincode

```bash
# Modify chaincode code
nano chaincode/basic/chaincode.go

# Increment version
# Update .env
CHAINCODE_VERSION=2.0
CHAINCODE_SEQUENCE=2

# Redeploy chaincode
./scripts/deployment/008-deploy-chaincode.sh
```

### Performance Tuning

#### Increase Docker Resources

```bash
# Edit Docker daemon configuration
sudo nano /etc/docker/daemon.json

# Add resource limits
{
  "default-runtime": "nvidia",
  "runtimes": {
    "nvidia": {
      "path": "nvidia-container-runtime",
      "runtimeArgs": []
    }
  },
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}

# Restart Docker
sudo systemctl restart docker
```

#### Optimize PostgreSQL

```bash
# Access PostgreSQL
docker exec -it postgres-orderer psql -U postgres fabric_ca_orderer

# Increase connection pool
ALTER SYSTEM SET max_connections = 200;

# Increase shared buffers
ALTER SYSTEM SET shared_buffers = '256MB';

# Reload configuration
SELECT pg_reload_conf();
```

#### Optimize CouchDB

```bash
# Access CouchDB configuration
curl -X PUT http://admin:password@localhost:5984/_node/_config/couchdb/max_dbs_open -d '"10000"'

# Increase cache size
curl -X PUT http://admin:password@localhost:5984/_node/_config/couchdb/max_document_size -d '"4294967296"'
```

## Best Practices

### Security

1. **Change Default Passwords**
   - Update all default passwords in `.env` file
   - Use strong, unique passwords for each component

2. **Use TLS**
   - Ensure TLS is enabled for all communications
   - Use valid certificates from a trusted CA in production

3. **Limit Network Access**
   - Configure firewall rules properly
   - Use network segmentation for production deployments

4. **Regular Updates**
   - Keep Hyperledger Fabric versions updated
   - Monitor security advisories
   - Apply patches promptly

### Scalability

1. **Add More Peers**
   - Scale horizontally by adding more peers per organization
   - Use load balancers for peer access

2. **Add More Orderers**
   - Implement Raft consensus with multiple orderers
   - Distribute orderers across different machines

3. **Database Optimization**
   - Tune PostgreSQL and CouchDB for production workloads
   - Implement database clustering for high availability

4. **Client Application Scaling**
   - Deploy multiple instances of client application
   - Use API gateway for load balancing

### Monitoring

1. **Implement Logging**
   - Centralize logs using ELK stack or similar
   - Set up log rotation to manage disk space

2. **Metrics Collection**
   - Collect and analyze metrics using Prometheus and Grafana
   - Monitor key performance indicators

3. **Alerting**
   - Set up alerts for critical failures
   - Monitor container health and resource usage

4. **Health Checks**
   - Implement regular health checks
   - Automate recovery procedures

## Conclusion

This guide provides comprehensive instructions for deploying and managing a production-ready Hyperledger Fabric network. By following these steps and best practices, you can successfully deploy a blockchain network that meets your business requirements.

For additional support or questions, refer to:
- Hyperledger Fabric Documentation: https://hyperledger-fabric.readthedocs.io/
- Fabric CA Documentation: https://hyperledger-fabric-ca.readthedocs.io/
- Project Issues: Check the project's issue tracker

Happy blockchain development!