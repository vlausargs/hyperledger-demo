# K8s Operator Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run the existing supply-chain Fabric network on a local **kind** cluster via **hlf-operator**, with the existing `api` and `web` connected end-to-end.

**Architecture:** hlf-operator watches CRDs in namespace `hlf` and reconciles CAs, an orderer, and two peers (each with CouchDB), generating all crypto itself. A channel is created and joined; the Go chaincode is deployed as a chaincode-as-a-service (ccaas) pod. The existing `api`/`web` Helm charts are pointed at the in-cluster peers via a regenerated connection profile.

**Tech Stack:** kind, kubectl, helm, krew + `kubectl-hlf` plugin, hlf-operator v1.13.x, Hyperledger Fabric 2.5.15, Fabric CA 1.5.5, CouchDB 3.3, Go chaincode (`packages/chaincode`).

## Global Constraints

- Namespace: `hlf` for all Fabric + app resources.
- Fabric image version pinned: `2.5.15` (peer, orderer). Fabric CA: `1.5.5`. CouchDB: `3.3`.
- MSP IDs (unchanged from current network): `Org1MSP`, `Org2MSP`, `OrdererMSP`.
- Channel name: `mychannel`. Chaincode name: `basic`.
- Endorsement policy: `AND('Org1MSP.member','Org2MSP.member')`.
- Orderer node count on kind: **1** (production uses 3–5 — out of scope here).
- No Istio on kind; expose `api`/`web` via NodePort/port-forward only.
- All new manifests/scripts live under `infra/k8s/operator/`.
- Canonical command reference: `github.com/kfsoftware/meetup-k8s-hlf-2024` (mirror exact flags/versions from there when a command below is ambiguous).
- Each task commits its manifests/scripts. Do not commit generated crypto or `*.block` artifacts — add them to `.gitignore`.

---

## File Structure

- `infra/k8s/operator/kind-config.yaml` — kind cluster definition (port mappings).
- `infra/k8s/operator/README.md` — human runbook (the command sequence, in order).
- `infra/k8s/operator/00-namespace.yaml` — `hlf` namespace.
- `infra/k8s/operator/scripts/10-cas.sh` — create 3 CAs.
- `infra/k8s/operator/scripts/20-orderer.sh` — create orderer node.
- `infra/k8s/operator/scripts/30-peers.sh` — create 2 peers.
- `infra/k8s/operator/scripts/40-channel.sh` — create + join channel.
- `infra/k8s/operator/scripts/50-chaincode.sh` — build image, ccaas sync, install/approve/commit.
- `infra/k8s/operator/scripts/60-apps.sh` — connection profile + api/web deploy.
- `infra/k8s/operator/.gitignore` — ignore generated crypto/blocks/resources.json.
- `Makefile` — new `kind-*` target group (accretes per task).
- Delete: `infra/k8s/helm/fabric/` (superseded — final task).

Each `scripts/*.sh` starts with `#!/usr/bin/env bash` + `set -euo pipefail` and is idempotent where practical.

---

### Task 1: Host prereqs, kind cluster, operator install

**Files:**
- Create: `infra/k8s/operator/kind-config.yaml`
- Create: `infra/k8s/operator/00-namespace.yaml`
- Create: `infra/k8s/operator/README.md`
- Create: `infra/k8s/operator/.gitignore`
- Modify: `Makefile` (add `kind-up`, `kind-down`, `kind-operator`)

**Interfaces:**
- Produces: a running kind cluster named `hlf`, namespace `hlf`, hlf-operator deployment ready, `kubectl-hlf` plugin available. Later tasks assume `kubectl` context = `kind-hlf`.

- [ ] **Step 1: Install tools (host, one-time)**

```bash
# kind
go install sigs.k8s.io/kind@v0.23.0   # or: brew install kind
# krew (kubectl plugin manager) — follow https://krew.sigs.k8s.io/docs/user-guide/setup/install/
kubectl krew install hlf
kubectl hlf version   # confirm plugin resolves
```
Expected: `kubectl hlf version` prints a version, no "unknown command".

- [ ] **Step 2: Write kind config**

`infra/k8s/operator/kind-config.yaml`:
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: hlf
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 30949   # web NodePort
        hostPort: 30949
      - containerPort: 30948   # api NodePort
        hostPort: 30948
```

- [ ] **Step 3: Create cluster**

Run:
```bash
kind create cluster --config infra/k8s/operator/kind-config.yaml
kubectl cluster-info --context kind-hlf
```
Expected: `Kubernetes control plane is running at https://127.0.0.1:...`.

- [ ] **Step 4: Namespace manifest + apply**

`infra/k8s/operator/00-namespace.yaml`:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: hlf
```
Run: `kubectl apply -f infra/k8s/operator/00-namespace.yaml`
Expected: `namespace/hlf created`.

- [ ] **Step 5: Install operator**

Run:
```bash
helm repo add kfs https://kfsoftware.github.io/hlf-helm-charts
helm repo update
helm install hlf-operator --version=1.13.0 kfs/hlf-operator -n hlf
```

- [ ] **Step 6: Verify operator ready + CRDs present**

Run:
```bash
kubectl -n hlf rollout status deploy/hlf-operator-controller-manager --timeout=180s
kubectl get crds | grep hlf.kungfusoftware.es
```
Expected: rollout "successfully rolled out"; CRDs listed include `fabriccas`, `fabricpeers`, `fabricordnodes`, `fabricmainchannels`, `fabricfollowerchannels`, `fabricchaincodes`.

- [ ] **Step 7: Write `.gitignore` + README skeleton**

`infra/k8s/operator/.gitignore`:
```
resources.json
*.block
*.tar.gz
crypto-config/
connection-profile*.yaml
```
`infra/k8s/operator/README.md`: a runbook listing Task order (1→7) with the commands from each task. Start it now; append per task.

- [ ] **Step 8: Add Makefile targets**

Append to `Makefile`:
```makefile
# ─── Kubernetes (kind + hlf-operator) ────────────────────────────────
kind-up:
	kind create cluster --config infra/k8s/operator/kind-config.yaml
	kubectl apply -f infra/k8s/operator/00-namespace.yaml

kind-operator:
	helm repo add kfs https://kfsoftware.github.io/hlf-helm-charts
	helm repo update
	helm install hlf-operator --version=1.13.0 kfs/hlf-operator -n hlf
	kubectl -n hlf rollout status deploy/hlf-operator-controller-manager --timeout=180s

kind-down:
	kind delete cluster --name hlf
```

- [ ] **Step 9: Commit**

```bash
git add infra/k8s/operator/ Makefile
git commit -m "feat(k8s): kind cluster config + hlf-operator install"
```

---

### Task 2: Certificate Authorities (org1, org2, orderer)

**Files:**
- Create: `infra/k8s/operator/scripts/10-cas.sh`
- Modify: `Makefile` (add `kind-cas`), `infra/k8s/operator/README.md`

**Interfaces:**
- Consumes: running operator (Task 1).
- Produces: three CAs reachable in-cluster as `org1-ca.hlf`, `org2-ca.hlf`, `ord-ca.hlf`, each with enroll id/pw `enroll`/`enrollpw`. Later tasks enroll peer/orderer/admin identities against these.

- [ ] **Step 1: Write CA creation script**

`infra/k8s/operator/scripts/10-cas.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
NS=hlf
for org in org1 org2 ord; do
  kubectl hlf ca create \
    --storage-class=standard \
    --capacity=1Gi \
    --name="${org}-ca" \
    --namespace="$NS" \
    --enroll-id=enroll \
    --enroll-pw=enrollpw \
    --hosts="${org}-ca.localho.st" \
    --version=1.5.5
done
```

- [ ] **Step 2: Run it**

Run: `bash infra/k8s/operator/scripts/10-cas.sh`
Expected: three `CA ... created` messages, no errors.

- [ ] **Step 3: Verify CAs running**

Run:
```bash
kubectl hlf ca list -n hlf
kubectl -n hlf get pods -l app=hlf-ca
```
Expected: `org1-ca`, `org2-ca`, `ord-ca` each `Running` and `1/1`. Wait/re-run until Running (operator provisions asynchronously).

- [ ] **Step 4: Add Makefile target + README**

`Makefile`:
```makefile
kind-cas:
	bash infra/k8s/operator/scripts/10-cas.sh
```
Append the CA commands to `README.md`.

- [ ] **Step 5: Commit**

```bash
git add infra/k8s/operator/scripts/10-cas.sh Makefile infra/k8s/operator/README.md
git commit -m "feat(k8s): create org1/org2/orderer Fabric CAs"
```

---

### Task 3: Orderer node

**Files:**
- Create: `infra/k8s/operator/scripts/20-orderer.sh`
- Modify: `Makefile` (add `kind-orderer`), `README.md`

**Interfaces:**
- Consumes: `ord-ca` (Task 2).
- Produces: one orderer node MSP `OrdererMSP`, in-cluster + admin TLS material registered. Later channel task references `OrdererMSP` and the orderer endpoint.

- [ ] **Step 1: Write orderer script**

`infra/k8s/operator/scripts/20-orderer.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
NS=hlf
# register orderer identity at ord-ca
kubectl hlf ca register \
  --name=ord-ca --namespace=$NS \
  --user=orderer --secret=ordererpw \
  --type=orderer --enroll-id=enroll --enroll-secret=enrollpw \
  --mspid=OrdererMSP

kubectl hlf ordnode create \
  --image=hyperledger/fabric-orderer --version=2.5.15 \
  --storage-class=standard --capacity=1Gi \
  --enroll-id=orderer --enroll-pw=ordererpw \
  --mspid=OrdererMSP \
  --name=orderer0 --namespace=$NS \
  --ca-name=ord-ca.$NS \
  --hosts=orderer0.localho.st
```

- [ ] **Step 2: Run + verify**

Run:
```bash
bash infra/k8s/operator/scripts/20-orderer.sh
kubectl -n hlf get pods -l app=hlf-ordnode
```
Expected: `orderer0` pod `Running` `1/1` (may take ~1 min).

- [ ] **Step 3: Makefile + README + commit**

```makefile
kind-orderer:
	bash infra/k8s/operator/scripts/20-orderer.sh
```
```bash
git add infra/k8s/operator/scripts/20-orderer.sh Makefile infra/k8s/operator/README.md
git commit -m "feat(k8s): create orderer node (OrdererMSP)"
```

---

### Task 4: Peers (peer0-org1, peer0-org2) with CouchDB

**Files:**
- Create: `infra/k8s/operator/scripts/30-peers.sh`
- Modify: `Makefile` (add `kind-peers`), `README.md`

**Interfaces:**
- Consumes: `org1-ca`, `org2-ca` (Task 2).
- Produces: `peer0-org1` (`Org1MSP`) and `peer0-org2` (`Org2MSP`), each with CouchDB state DB, reachable in-cluster. Later channel/chaincode tasks target these MSPs + peers.

- [ ] **Step 1: Write peers script**

`infra/k8s/operator/scripts/30-peers.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
NS=hlf
create_peer () {
  local org=$1 mspid=$2 ca=$3
  kubectl hlf ca register --name=$ca --namespace=$NS \
    --user=peer --secret=peerpw --type=peer \
    --enroll-id=enroll --enroll-secret=enrollpw --mspid=$mspid
  kubectl hlf peer create \
    --image=hyperledger/fabric-peer --version=2.5.15 \
    --storage-class=standard --capacity=2Gi \
    --enroll-id=peer --enroll-pw=peerpw --mspid=$mspid \
    --name=peer0-$org --namespace=$NS \
    --ca-name=$ca.$NS \
    --statedb=couchdb \
    --hosts=peer0-$org.localho.st
}
create_peer org1 Org1MSP org1-ca
create_peer org2 Org2MSP org2-ca
```

- [ ] **Step 2: Run + verify**

Run:
```bash
bash infra/k8s/operator/scripts/30-peers.sh
kubectl -n hlf get pods -l app=hlf-peer
```
Expected: `peer0-org1` and `peer0-org2` pods `Running` (each pod includes a CouchDB container; `2/2` or peer+couchdb both ready). Re-run `get pods` until ready.

- [ ] **Step 3: Makefile + README + commit**

```makefile
kind-peers:
	bash infra/k8s/operator/scripts/30-peers.sh
```
```bash
git add infra/k8s/operator/scripts/30-peers.sh Makefile infra/k8s/operator/README.md
git commit -m "feat(k8s): create peer0-org1 + peer0-org2 with CouchDB"
```

---

### Task 5: Channel `mychannel` + peer joins

**Files:**
- Create: `infra/k8s/operator/scripts/40-channel.sh`
- Modify: `Makefile` (add `kind-channel`), `README.md`

**Interfaces:**
- Consumes: orderer (Task 3), both peers (Task 4).
- Produces: channel `mychannel` created via `FabricMainChannel`, both peers joined via `FabricFollowerChannel`. Later chaincode + app tasks assume the channel exists and both peers are members.

- [ ] **Step 1: Register admin identities for org1, org2, orderer**

`infra/k8s/operator/scripts/40-channel.sh` (part A):
```bash
#!/usr/bin/env bash
set -euo pipefail
NS=hlf
# admin identities used to sign channel config
kubectl hlf ca register --name=org1-ca --namespace=$NS --user=admin --secret=adminpw --type=admin --enroll-id=enroll --enroll-secret=enrollpw --mspid=Org1MSP
kubectl hlf ca register --name=org2-ca --namespace=$NS --user=admin --secret=adminpw --type=admin --enroll-id=enroll --enroll-secret=enrollpw --mspid=Org2MSP
kubectl hlf ca register --name=ord-ca  --namespace=$NS --user=admin --secret=adminpw --type=admin --enroll-id=enroll --enroll-secret=enrollpw --mspid=OrdererMSP
```

- [ ] **Step 2: Create the FabricMainChannel + FollowerChannels**

`40-channel.sh` (part B) — apply CRD manifests. Follow the exact `FabricMainChannel`/`FabricFollowerChannel` spec shape from the canonical meetup repo (`kubectl hlf channel` helpers, or apply YAML). The channel must list `Org1MSP` + `Org2MSP` as peer orgs and `OrdererMSP` as the orderer org, with the orderer endpoint `orderer0.hlf:7050`. Then a `FabricFollowerChannel` per peer to join.
```bash
# Pattern (verify field names against operator v1.13 CRD reference):
kubectl hlf channel generate --output=mychannel.block --name=mychannel \
  --organizations Org1MSP --organizations Org2MSP --ordererOrganizations OrdererMSP
# then apply FabricMainChannel + FabricFollowerChannel manifests (committed alongside script)
kubectl apply -f infra/k8s/operator/manifests/mainchannel.yaml
kubectl apply -f infra/k8s/operator/manifests/follower-org1.yaml
kubectl apply -f infra/k8s/operator/manifests/follower-org2.yaml
```
Create the three manifests under `infra/k8s/operator/manifests/` with the concrete CRD specs (name `mychannel`, the MSPs above, orderer `orderer0.hlf:7050`).

- [ ] **Step 3: Run + verify both peers joined**

Run:
```bash
bash infra/k8s/operator/scripts/40-channel.sh
kubectl hlf channel inspect --channel mychannel --namespace hlf --config <(kubectl hlf inspect --output resources.json)
```
Expected: channel height ≥ 1; both `peer0-org1` and `peer0-org2` appear as members. (Use `kubectl hlf peer channels` per peer if `inspect` flags differ in your operator version.)

- [ ] **Step 4: Makefile + README + commit**

```makefile
kind-channel:
	bash infra/k8s/operator/scripts/40-channel.sh
```
```bash
git add infra/k8s/operator/scripts/40-channel.sh infra/k8s/operator/manifests/ Makefile infra/k8s/operator/README.md
git commit -m "feat(k8s): create mychannel and join both peers"
```

---

### Task 6: Chaincode `basic` as ccaas

**Files:**
- Create: `infra/k8s/operator/scripts/50-chaincode.sh`
- Create: `packages/chaincode/Dockerfile` (if not already suitable for ccaas)
- Modify: `Makefile` (add `kind-chaincode`), `README.md`

**Interfaces:**
- Consumes: joined channel (Task 5), Go source in `packages/chaincode`.
- Produces: chaincode `basic` committed on `mychannel`, endorsement `AND(Org1MSP,Org2MSP)`, running as a ccaas pod. Apps invoke it afterward.

- [ ] **Step 1: Confirm/author ccaas Dockerfile**

Verify `packages/chaincode` builds a ccaas server binary (a chaincode that listens, launched by `CHAINCODE_SERVER_ADDRESS`). If `packages/chaincode/Dockerfile` is missing or peer-launched-style, add a ccaas Dockerfile:
```dockerfile
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /chaincode .
FROM gcr.io/distroless/static
COPY --from=build /chaincode /chaincode
ENV CHAINCODE_SERVER_ADDRESS=0.0.0.0:9999
EXPOSE 9999
ENTRYPOINT ["/chaincode"]
```
(Confirm `main.go` starts a `contractapi` chaincode server when `CHAINCODE_SERVER_ADDRESS` is set; adjust per `packages/chaincode/main.go`.)

- [ ] **Step 2: Build image + load into kind**

```bash
docker build -t hlf-basic-cc:1.0 packages/chaincode
kind load docker-image hlf-basic-cc:1.0 --name hlf
```
Expected: `Image: "hlf-basic-cc:1.0" ... loaded`.

- [ ] **Step 3: Write chaincode deploy script**

`infra/k8s/operator/scripts/50-chaincode.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
NS=hlf
CC=basic
# package id derived from label + image; follow meetup repo pattern:
PACKAGE_ID=$(kubectl hlf chaincode calculatepackageid --path=/dev/null --language=golang --label=$CC 2>/dev/null || echo "$CC:PLACEHOLDER")

kubectl hlf externalchaincode sync \
  --image=hlf-basic-cc:1.0 \
  --name=$CC --namespace=$NS \
  --package-id="$PACKAGE_ID" \
  --tls-required=false --replicas=1

# install on both peers, approve for each org, commit
kubectl hlf chaincode install --path=./chaincode.tgz --language=golang --label=$CC \
  --user=admin --peer=peer0-org1.$NS
kubectl hlf chaincode install --path=./chaincode.tgz --language=golang --label=$CC \
  --user=admin --peer=peer0-org2.$NS

kubectl hlf chaincode approveformyorg --config=resources.json --user=admin --peer=peer0-org1.$NS \
  --package-id="$PACKAGE_ID" --version=1.0 --sequence=1 --name=$CC --channel=mychannel \
  --policy="AND('Org1MSP.member','Org2MSP.member')"
kubectl hlf chaincode approveformyorg --config=resources.json --user=admin --peer=peer0-org2.$NS \
  --package-id="$PACKAGE_ID" --version=1.0 --sequence=1 --name=$CC --channel=mychannel \
  --policy="AND('Org1MSP.member','Org2MSP.member')"

kubectl hlf chaincode commit --config=resources.json --user=admin --mspid=Org1MSP \
  --version=1.0 --sequence=1 --name=$CC --channel=mychannel \
  --policy="AND('Org1MSP.member','Org2MSP.member')"
```
Note: exact package-id derivation for ccaas follows the meetup repo (`connection.json` + `metadata.json` → tgz). Mirror that sequence; the placeholder above must be replaced with the real computed id before approve/commit.

- [ ] **Step 4: Run + verify with a query**

Run:
```bash
bash infra/k8s/operator/scripts/50-chaincode.sh
kubectl -n hlf get pods -l app.kubernetes.io/name=$CC   # ccaas pod Running
kubectl hlf chaincode query --config=resources.json --user=admin --peer=peer0-org1.hlf \
  --chaincode=basic --channel=mychannel --fcn=GetAllAssets -a '[]' || true
```
Expected: ccaas pod `Running`; commit succeeds; a query returns (empty list or seeded data) without a "chaincode not found" error.

- [ ] **Step 5: Makefile + README + commit**

```makefile
kind-chaincode:
	docker build -t hlf-basic-cc:1.0 packages/chaincode
	kind load docker-image hlf-basic-cc:1.0 --name hlf
	bash infra/k8s/operator/scripts/50-chaincode.sh
```
```bash
git add infra/k8s/operator/scripts/50-chaincode.sh packages/chaincode/Dockerfile Makefile infra/k8s/operator/README.md
git commit -m "feat(k8s): deploy basic chaincode as ccaas and commit on mychannel"
```

---

### Task 7: Connect `api` + `web`

**Files:**
- Create: `infra/k8s/operator/scripts/60-apps.sh`
- Modify: `infra/k8s/helm/api/values.yaml`, `infra/k8s/helm/web/values.yaml` (NodePort + connection profile mount), `Makefile` (add `kind-apps`), `README.md`

**Interfaces:**
- Consumes: committed chaincode (Task 6), operator-issued peer/orderer TLS material.
- Produces: `api` (2 replicas) + `web` deployed, reachable via NodePort, driving transactions against the in-cluster network.

- [ ] **Step 1: Generate connection profile from operator material**

`infra/k8s/operator/scripts/60-apps.sh` (part A):
```bash
#!/usr/bin/env bash
set -euo pipefail
NS=hlf
kubectl hlf inspect --output resources.json
# build a Gateway connection profile for Org1 pointing at in-cluster DNS:
kubectl hlf networkconfig generate --output connection-profile.yaml \
  --config resources.json --organizations Org1MSP --internal
kubectl -n $NS create secret generic api-connection \
  --from-file=connection-profile.yaml --dry-run=client -o yaml | kubectl apply -f -
```
(`--internal` yields in-cluster hostnames like `peer0-org1.hlf:7051`. Verify flag name against operator version; fall back to `kubectl hlf inspect` output if `networkconfig` differs.)

- [ ] **Step 2: Point api chart at the secret + NodePort**

Edit `infra/k8s/helm/api/values.yaml`: set `service.type: NodePort` with `nodePort: 30948`; add a volume mounting secret `api-connection` at the path the api reads its connection profile from; set env for MSP `Org1MSP` and the identity to use. Edit `infra/k8s/helm/web/values.yaml`: `service.type: NodePort`, `nodePort: 30949`, and API base URL → `http://localhost:30948`.

- [ ] **Step 3: Build + load app images, deploy charts**

`60-apps.sh` (part B):
```bash
docker build -t hlf-api:local -f infra/docker/Dockerfiles/api.Dockerfile .
docker build -t hlf-web:local -f infra/docker/Dockerfiles/web.Dockerfile .
kind load docker-image hlf-api:local hlf-web:local --name hlf
helm install api infra/k8s/helm/api -n hlf --set image.repository=hlf-api --set image.tag=local
helm install web infra/k8s/helm/web -n hlf --set image.repository=hlf-web --set image.tag=local
```

- [ ] **Step 4: Run + verify end-to-end**

Run:
```bash
bash infra/k8s/operator/scripts/60-apps.sh
kubectl -n hlf rollout status deploy/api --timeout=120s
curl -s http://localhost:30948/health          # api health
curl -s http://localhost:30948/api/v1/network/organizations   # reaches network
```
Expected: health OK; organizations endpoint returns `Org1MSP`/`Org2MSP` — proves the api reached the in-cluster peers.

- [ ] **Step 5: Makefile + README + commit**

```makefile
kind-apps:
	bash infra/k8s/operator/scripts/60-apps.sh
```
```bash
git add infra/k8s/operator/scripts/60-apps.sh infra/k8s/helm/api/values.yaml infra/k8s/helm/web/values.yaml Makefile infra/k8s/operator/README.md
git commit -m "feat(k8s): deploy api+web against operator network via connection profile"
```

---

### Task 8: Full-run verification, retire skeleton chart, docs

**Files:**
- Delete: `infra/k8s/helm/fabric/`
- Modify: `infra/k8s/helm/README.md`, `Makefile` (add composite `kind-network`, `kind-all`), root docs
- Create: (nothing new)

**Interfaces:**
- Consumes: all prior tasks.
- Produces: one-command bring-up, clean repo (skeleton chart gone), passing e2e.

- [ ] **Step 1: Add composite Makefile targets**

```makefile
kind-network: kind-cas kind-orderer kind-peers kind-channel
kind-all: kind-up kind-operator kind-network kind-chaincode kind-apps
```

- [ ] **Step 2: Clean-cluster full run**

Run:
```bash
make kind-down || true
make kind-all
kubectl get pods -n hlf
```
Expected: `kubectl get pods -n hlf` → CAs, orderer, 2 peers (+couchdb), chaincode, api, web all `Running`. Fix any task whose pods are not Running before proceeding.

- [ ] **Step 3: Run existing e2e against the api**

Run the Playwright suite pointed at `http://localhost:30949` (web) / `http://localhost:30948` (api). Consult `e2e/playwright` config for the base-URL env var.
```bash
cd packages/web && PLAYWRIGHT_BASE_URL=http://localhost:30949 pnpm test:e2e
```
Expected: suite passes (or the same subset that passes today on bare-metal).

- [ ] **Step 4: Retire skeleton fabric chart + update docs**

```bash
git rm -r infra/k8s/helm/fabric
```
Edit `infra/k8s/helm/README.md`: remove the `fabric/` chart row; add a note that the Fabric network is now provisioned by hlf-operator (`infra/k8s/operator/`). Ensure `infra/k8s/operator/README.md` is a complete runbook (Tasks 1→7 in order).

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore(k8s): composite kind targets, retire skeleton fabric chart, runbook"
```

---

## Self-Review

**Spec coverage:**
- kind + operator install → Task 1 ✓
- 3 CAs → Task 2 ✓
- Orderer (1 node) → Task 3 ✓
- 2 peers + CouchDB → Task 4 ✓
- Channel + joins → Task 5 ✓
- Chaincode ccaas from `packages/chaincode` → Task 6 ✓
- api/web connected via regenerated connection profile → Task 7 ✓
- Verification (pods, channel, tx, e2e) → Tasks 6, 7, 8 ✓
- Skip Istio → honored (NodePort in Task 7) ✓
- Retire skeleton fabric chart → Task 8 ✓
- Endorsement `AND(Org1MSP,Org2MSP)` → Task 6 ✓
- Pin Fabric 2.5.15 → Tasks 3,4 ✓

**Known soft spots (verify against operator v1.13 CRD reference during execution — flag names drift between operator versions):** exact `FabricMainChannel`/`FabricFollowerChannel` field names (Task 5), ccaas package-id derivation + `connection.json`/`metadata.json` (Task 6), `kubectl hlf networkconfig`/`inspect` flags for the connection profile (Task 7). The canonical `kfsoftware/meetup-k8s-hlf-2024` repo is the authority for these exact sequences; mirror it. These are execution-time confirmations, not design gaps.

**Placeholder scan:** command sequences are concrete; the only literal placeholder (`PACKAGE_ID`) is explicitly called out with instructions to replace before approve/commit.

**Type consistency:** MSP IDs (`Org1MSP`/`Org2MSP`/`OrdererMSP`), names (`org1-ca`/`peer0-org1`/`orderer0`/`mychannel`/`basic`), and ports (30948 api, 30949 web) are consistent across all tasks.
