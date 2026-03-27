# Environment Variable Cleanup Summary

## Overview

This document summarizes the cleanup of redundant environment variables in the Hyperledger Fabric client application. The changes prioritize using only the ConnectionProfile YAML file for network configuration, removing duplicate configuration sources.

## Problem Statement

The client application and deployment scripts were using multiple sources for the same configuration:

- **Environment variables**: `PEER_ENDPOINT`, `GATEWAY_PEER`, `MSP_ID`, `ORG_NAME`, `USER_ID`
- **Connection profile YAML**: All network-related information (peer endpoints, MSP IDs, organization names, etc.)

This duplication caused:
- Configuration inconsistencies
- Maintenance overhead (changes needed in multiple places)
- Confusion about which source takes precedence

## Solution

The solution was to remove redundant environment variables and rely exclusively on the ConnectionProfile for network configuration. The ConnectionProfile already contains all necessary network information:

```yaml
organizations:
  Org1:
    mspid: Org1MSP
    peers:
    - peer0.org1.example.com
    
client:
  organization: Org1

peers:
  peer0.org1.example.com:
    url: grpcs://peer0.org1.example.com:8051
```

## Changes Made

### 1. `/fabric-network/client/main.go`

**Removed from Config struct:**
- `PeerEndpoint` - Now extracted from `connectionProfile.peers[peerName].url`
- `GatewayPeer` - Now extracted from `connectionProfile.client.organization` and `connectionProfile.organizations[orgName].peers[0]`
- `MSPID` - Now extracted from `connectionProfile.organizations[orgName].mspid`
- `UserID` - Not used in the actual connector code
- `OrgName` - Now extracted from `connectionProfile.client.organization`

**Removed default constants:**
- `defaultPeerEndpoint`
- `defaultGatewayPeer`

**Simplified validation:**
- Removed MSPID validation (no longer required as it comes from ConnectionProfile)

### 2. `/fabric-network/scripts/deployment/009-start-client.sh`

**Removed environment variable exports:**
```bash
# REMOVED:
export PEER_ENDPOINT="peer0.${ORG1_DOMAIN}:${PEER0_ORG1_PORT}"
export GATEWAY_PEER="peer0.${ORG1_DOMAIN}"
export MSP_ID="${ORG1_NAME}"
export USER_ID="user1"
export ORG_NAME="Org1"
```

### 3. `/fabric-network/client/start.sh`

**Removed environment variable exports:**
```bash
# REMOVED:
export PEER_ENDPOINT="peer0.org1.example.com:8051"
export GATEWAY_PEER="peer0.org1.example.com"
export MSP_ID="Org1MSP"
export USER_ID="user1"
export ORG_NAME="Org1"
```

### 4. `/fabric-network/client/fabric/connector.go`

**Removed unused struct:**
- Removed the entire `Config` struct (lines 88-96) that contained the removed fields

## Remaining Environment Variables

The following environment variables are **still required** as they are not available in the ConnectionProfile:

| Variable | Purpose | Example |
|----------|---------|---------|
| `WALLET_PATH` | Path to store/load identity files | `./wallet` |
| `CHANNEL_ID` | Channel to connect to | `mychannel` |
| `CHAINCODE_ID` | Chaincode to invoke | `basic` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `GIN_MODE` | Gin framework mode | `debug` |
| `TLS_CERT_PATH` | Path to crypto materials | `./crypto` |
| `CONNECTION_PROFILE` | Path to connection profile YAML | `./crypto/connection-profile.yaml` |

## Benefits

1. **Single Source of Truth**: All network configuration is now centralized in the ConnectionProfile YAML file

2. **Reduced Configuration Complexity**: Fewer environment variables to manage and document

3. **Improved Maintainability**: Network changes only need to be made in one place (the ConnectionProfile)

4. **Better Organization**: Clear separation between:
   - Network configuration (ConnectionProfile)
   - Application configuration (Environment variables)

5. **Prevents Inconsistencies**: No risk of environment variables conflicting with ConnectionProfile values

6. **Simplified Deployment**: Deployment scripts are cleaner and easier to understand

## How It Works

The `fabric.NewGateway()` function now extracts all necessary information from the ConnectionProfile:

```go
// From connector.go
orgName := gw.connectionProfile.Client.Organization
org := gw.connectionProfile.Organizations[orgName]
mspid := org.MSPID
peerName := org.Peers[0]
peer := gw.connectionProfile.Peers[peerName]
peerURL := peer.URL
```

No environment variables are used for network configuration beyond the path to the ConnectionProfile itself.

## Testing

The changes have been verified by:
- Successfully building the client application
- Confirming that the `connector.go` code extracts all required information from the ConnectionProfile
- Verifying that no compilation errors exist

## Migration Notes

If you have existing deployments using the old environment variables:

1. **Update deployment scripts**: Remove references to `PEER_ENDPOINT`, `GATEWAY_PEER`, `MSP_ID`, `USER_ID`, `ORG_NAME`

2. **Verify ConnectionProfile**: Ensure your ConnectionProfile YAML contains:
   - Valid organization definitions with MSP IDs
   - Valid peer definitions with URLs
   - Client organization reference

3. **Test deployment**: Run the deployment scripts to confirm the client connects successfully

## Conclusion

This cleanup aligns the client application with Hyperledger Fabric best practices, where the ConnectionProfile is the recommended method for managing network connection details. The application is now simpler, more maintainable, and less error-prone.