# Hyperledger Fabric Network - Production Ready Setup

This project provides a production-ready Hyperledger Fabric network with the following characteristics:
- 2 Organizations with 1 peer each
- 1 Orderer service
- Fabric Certificate Authority (CA) with PostgreSQL for identity management
- CouchDB for world state database (1 CouchDB per peer)
- Go chaincode support
- Go client application with Gin framework
- Docker-based deployment
- Modular deployment across multiple machines
- Environment-based configuration

## Architecture Overview

```
Machine A (Orderer):
├── Orderer Service
└── PostgreSQL Database (for Orderer CA)

Machine B (Organization 1):
├── Peer1
├── CouchDB for Peer1
├── Org1 CA
└── PostgreSQL Database (for Org1 CA)

Machine C (Organization 2):
├── Peer2
├── CouchDB for Peer2
├── Org2 CA
└── PostgreSQL Database (for Org2 CA)
```

## Project Structure

```
fabric-network/
├── config/                    # Configuration files
│   ├── ca/                    # CA configurations
│   ├── orderer/               # Orderer configurations
│   ├── peer/                  # Peer configurations
│   └── blockchain/            # Network and channel configurations
├── organizations/             # Generated certificates and MSP
│   ├── org1/                  # Organization 1 certificates
│   ├── org2/                  # Organization 2 certificates
│   └── orderer/               # Orderer certificates
├── scripts/                   # Deployment and management scripts
│   ├── docker/                # Docker-related scripts
│   └── deployment/            # Deployment scripts (001-*.sh, 002-*.sh, etc.)
├── chaincode/                 # Go chaincode
│   └── basic/                 # Basic chaincode example
├── client/                    # Go client application
│   ├── main.go               # Gin server entry point
│   └── rest/                  # REST API handlers
├── docker-compose/            # Docker Compose files
│   ├── orderer/               # Orderer docker-compose
│   ├── org1/                  # Org1 docker-compose
│   └── org2/                  # Org2 docker-compose
└── .env                       # Environment configuration
```

## Prerequisites

- Docker (20.10+)
- Docker Compose (2.0+)
- Go (1.19+)
- PostgreSQL (13+)
- OpenSSL
- Curl
- JQ (for JSON processing)

## Installation & Deployment

### 1. Initial Setup

Clone or navigate to the project directory:

```bash
cd fabric-network
```

### 2. Configure Environment

Edit the `.env` file to configure your environment:

```bash
nano .env
```

Key configuration options:
- Network settings (domains, ports)
- PostgreSQL credentials
- Fabric version
- Organization and orderer details

### 3. Deploy Network Step-by-Step

The deployment is organized into numbered scripts for clarity and control:

```bash
# Step 1: Deploy PostgreSQL databases
./scripts/deployment/001-deploy-postgres.sh

# Step 2: Deploy Fabric CA services
./scripts/deployment/002-deploy-ca.sh

# Step 3: Register and enroll identities
./scripts/deployment/003-setup-ca.sh

# Step 4: Generate channel configuration
./scripts/deployment/004-generate-configtx.sh

# Step 5: Deploy Orderer
./scripts/deployment/005-deploy-orderer.sh

# Step 6: Deploy Peers
./scripts/deployment/006-deploy-peers.sh

# Step 7: Create and join channels
./scripts/deployment/007-create-channel.sh

# Step 8: Deploy chaincode
./scripts/deployment/008-deploy-chaincode.sh

# Step 9: Build and start client application
./scripts/deployment/009-start-client.sh
```

### 4. Modular Deployment Across Machines

For deploying components across different machines:

#### On Machine A (Orderer):
1. Copy the project to Machine A
2. Edit `.env` and set `DEPLOY_ORDERER=true`, `DEPLOY_ORG1=false`, `DEPLOY_ORG2=false`
3. Run scripts 001, 002, 004, and 005

#### On Machine B (Org1):
1. Copy the project to Machine B
2. Edit `.env` and set `DEPLOY_ORDERER=false`, `DEPLOY_ORG1=true`, `DEPLOY_ORG2=false`
3. Run scripts 001, 002, 003, 006, and 008
4. Copy the channel configuration from the machine where it was created

#### On Machine C (Org2):
1. Copy the project to Machine C
2. Edit `.env` and set `DEPLOY_ORDERER=false`, `DEPLOY_ORG1=false`, `DEPLOY_ORG2=true`
3. Run scripts 001, 002, 003, 006, and 008
4. Copy the channel configuration from the machine where it was created

## Components

### Fabric Certificate Authority (CA)

Each organization has its own CA backed by PostgreSQL for persistent identity storage.

### Orderer

Uses Raft consensus mechanism with a single orderer node. Can be expanded to multiple orderers for high availability.

### Peers

Each peer has:
- CouchDB for state database
- Secure communication using TLS
- Access to the application channel

### Chaincode

Developed in Go and deployed via Lifecycle Chaincode (LLC).

### Client Application

Go-based REST API server using Gin framework for interacting with the blockchain network.

## Network Operations

### Check Status

```bash
# Check containers
docker-compose ps

# Check logs
docker-compose logs -f [service_name]

# Check peer status
docker exec peer0.org1.example.com peer status
```

### Channel Operations

```bash
# List channels
docker exec peer0.org1.example.com peer channel list

# Get channel info
docker exec peer0.org1.example.com peer channel getinfo -c mychannel
```

### Chaincode Operations

```bash
# Invoke chaincode
docker exec peer0.org1.example.com peer chaincode invoke \
  -o orderer.example.com:7050 \
  -C mychannel \
  -n mycc \
  -c '{"Args":["init","a","100","b","200"]}'

# Query chaincode
docker exec peer0.org1.example.com peer chaincode query \
  -C mychannel \
  -n mycc \
  -c '{"Args":["query","a"]}'
```

## Troubleshooting

### Common Issues

1. **CA not starting**: Check PostgreSQL connection and credentials
2. **Peer cannot join channel**: Verify peer certificates and channel configuration
3. **Chaincode not working**: Check chaincode version and endorsement policy
4. **Network connectivity**: Ensure proper domain resolution between machines

### Logs

Check logs for specific services:

```bash
# Orderer logs
docker-compose logs -f orderer.example.com

# Peer logs
docker-compose logs -f peer0.org1.example.com

# CA logs
docker-compose logs -f ca.org1.example.com

# CouchDB logs
docker-compose logs -f couchdb.org1.example.com
```

### Reset Network

To completely reset the network:

```bash
./scripts/deployment/999-teardown.sh
rm -rf organizations/
```

## Security Considerations

- All communications use TLS
- Certificates are generated using Fabric CA (not cryptogen)
- PostgreSQL databases use strong passwords
- Environment variables contain sensitive information - protect `.env` file
- Use production-grade certificates for production deployments

## Scalability

This setup can be scaled by:
- Adding more peers per organization
- Adding more orderers for Raft consensus
- Adding more organizations
- Load balancing peers

## Production Checklist

Before deploying to production:

- [ ] Use strong PostgreSQL passwords
- [ ] Obtain valid TLS certificates from a CA
- [ ] Configure proper backup strategies for databases
- [ ] Set up monitoring and logging
- [ ] Configure firewall rules
- [ ] Implement proper identity management
- [ ] Test disaster recovery procedures
- [ ] Set up log aggregation
- [ ] Configure resource limits for Docker containers
- [ ] Review and adjust channel policies

## Support & Maintenance

For updates and maintenance:
- Keep Fabric versions updated
- Regularly review security advisories
- Monitor performance metrics
- Backup channel configuration and certificates
- Keep chaincode versioned

## License

This project is provided as-is for educational and production purposes.