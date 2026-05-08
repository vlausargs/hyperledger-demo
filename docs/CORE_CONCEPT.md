# Hyperledger Fabric — Core Concepts

## What Is Hyperledger Fabric?

Permissioned blockchain framework. Unlike Bitcoin/Ethereum (public, anonymous), Fabric requires identity for every participant. Only known, verified members can join the network. Built for enterprise use cases where you need privacy, control, and compliance.

---

## Key Difference From Public Blockchains

| | Public (Bitcoin/Ethereum) | Hyperledger Fabric |
|---|---|---|
| Access | Anyone | Invited members only |
| Identity | Anonymous | Known (PKI-based) |
| Consensus | Proof of Work/Stake | Pluggable (etcdraft) |
| Speed | Slow (thousands of ms) | Fast (sub-second) |
| Privacy | All see all | Channels + private data |
| Token | Required | Not required |

---

## Core Components

### 1. Organizations
Real-world entities (companies, departments) that participate in the network. Each org controls its own members and infrastructure.

In this project: `Org1`, `Org2`, `OrdererOrg`

### 2. Peers
Servers that hold copies of the ledger and run chaincode. Owned by organizations. Two roles:
- **Endorsing peer** — simulates transactions, signs them
- **Committing peer** — validates and writes blocks to ledger

Every peer is both endorsing and committing.

### 3. Orderer
Does NOT hold ledger state. Single job: receive endorsed transactions from all orgs, order them into blocks, distribute blocks to peers. Uses `etcdraft` consensus (like Raft — leader-based, crash fault tolerant).

Think of orderer as: **postal service** — collects letters, bundles into packages, delivers in order.

### 4. Certificate Authority (CA)
Issues cryptographic identities (X.509 certs) to every participant — admins, users, peers, orderers. Identity = your cert signed by your org's CA. No cert = no access.

`fabric-ca-client` tool registers and enrolls identities against Fabric CA.

### 5. Channel
A private subnet within the network. Only channel members see transactions on that channel. Multiple channels can exist on same network — orgs on Channel A cannot see Channel B's data.

This project uses one channel: `mychannel`

### 6. Ledger
Two parts:
- **World State** — current key-value snapshot (stored in CouchDB here)
- **Blockchain** — immutable log of all transactions that produced current state

CouchDB enables rich JSON queries on world state. The blockchain log never changes.

### 7. Chaincode (Smart Contract)
Business logic deployed to peers. Written in Go, Java, or Node.js. Defines what transactions are valid and how ledger state changes.

This project's chaincode (`basic`): manages assets — create, read, update, delete, transfer ownership.

### 8. MSP (Membership Service Provider)
Translates certs → identities → roles. Each org has an MSP that says "this cert belongs to Org1, role=member". Network uses MSPs to enforce who can do what.

MSP IDs in this project: `Org1MSP`, `Org2MSP`, `OrdererMSP`

---

## Transaction Flow (How a TX Gets Committed)

```
Client App
    │
    ▼
1. PROPOSE — Client sends transaction proposal to endorsing peers
    │
    ▼
2. ENDORSE — Each endorsing peer simulates the tx (runs chaincode),
              signs result. Does NOT write to ledger yet.
    │
    ▼
3. COLLECT — Client collects enough endorsements (per endorsement policy)
    │
    ▼
4. SUBMIT — Client sends endorsed tx to Orderer
    │
    ▼
5. ORDER — Orderer batches txs into a block, signs it, broadcasts to all peers
    │
    ▼
6. VALIDATE & COMMIT — Each peer validates signatures, checks endorsement
                        policy, checks for conflicts, writes block to ledger
    │
    ▼
7. EVENT — Peers emit commit events back to client
```

**Endorsement policy** controls step 3. This project requires: `AND('Org1MSP.member', 'Org2MSP.member')` — both orgs must endorse every transaction. Neither org can act unilaterally.

---

## Endorsement Policy

Rule that says: "who must sign this transaction for it to be valid?"

Examples:
- `AND('Org1MSP.member', 'Org2MSP.member')` — both must sign
- `OR('Org1MSP.member', 'Org2MSP.member')` — either can sign
- `OutOf(2, 'Org1MSP.member', 'Org2MSP.member', 'Org3MSP.member')` — 2 of 3

This project uses AND — mutual trust requirement, neither party controls alone.

---

## Chaincode Lifecycle (Fabric 2.x)

Deploying chaincode is a multi-step governance process:

```
1. Package   — Create .tar.gz of chaincode source
2. Install   — Install package on each org's peer (each org does this)
3. Approve   — Each org votes to approve the chaincode definition
4. Commit    — Once enough orgs approved, commit definition to channel
5. Ready     — Chaincode now executable
```

Both orgs must approve before commit. This prevents one org from deploying malicious logic.

---

## Identity & PKI

Every actor has an X.509 certificate issued by their org's CA:

```
Root CA (org)
  └── TLS CA
  └── Peer certs
  └── Orderer certs
  └── Admin certs
  └── User certs
```

TLS secures all communication (peer↔peer, peer↔orderer, client↔peer). MSP maps certs to org membership. No cert = rejected at network boundary.

---

## World State vs Blockchain

```
Blockchain (immutable log):
  Block 1 → Block 2 → Block 3 → Block 4
  [create A] [update A] [transfer A] [delete A]

World State (CouchDB, current snapshot):
  Asset A: { owner: "Bob", value: 500 }  ← result of all above txs
```

Chaincode reads/writes World State. History queries walk the Blockchain. Both are part of the Ledger.

---

## How This Project Maps to Concepts

| Concept | This Project |
|---------|-------------|
| Organizations | Org1, Org2, OrdererOrg |
| Peers | peer0.org1, peer0.org2 |
| Orderer | orderer1.orderer.example.com |
| CAs | ca.orderer, ca.org1, ca.org2 |
| Channel | mychannel |
| World State DB | CouchDB (one per peer) |
| Chaincode | basic (asset management) |
| Client SDK | Fabric Gateway v1.5.1 |
| Client App | Go REST API (Gin, port 8080) |
| Endorsement Policy | AND(Org1MSP, Org2MSP) |

---

## Summary

Fabric = **permissioned blockchain where known parties transact privately, with business logic enforced by smart contracts, and no single party controls the ledger.**

Trust model: cryptographic identity + endorsement policy + immutable log. No party can forge a transaction or change history. All parties see the same state. Disputes resolved by the ledger, not lawyers.
