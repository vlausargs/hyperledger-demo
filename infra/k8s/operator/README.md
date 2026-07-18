# hlf-operator on kind — runbook

Rehearses moving the Fabric supply-chain network onto Kubernetes via
[hlf-operator](https://github.com/kfsoftware/hlf-operator), on a local
[kind](https://kind.sigs.k8s.io/) cluster before targeting a real cluster.

Plan: `docs/superpowers/plans/2026-07-17-k8s-operator-deployment.md`

Cluster name: `hlf` (kubectl context `kind-hlf`). Namespace: `hlf`.

Task order and commands, appended as each task lands:

## Task 1: Host prereqs, kind cluster, operator install (DONE)

Tools (host, one-time): `kubectl`, `kind`, `helm`, and the `kubectl-hlf` krew
plugin. Confirm with `kubectl hlf version`.

Create the cluster and namespace:

```bash
kind create cluster --config infra/k8s/operator/kind-config.yaml
kubectl cluster-info --context kind-hlf
kubectl apply -f infra/k8s/operator/00-namespace.yaml
```

Install the operator (chart version pinned to match the installed
`kubectl-hlf` plugin's minor line — see Makefile `kind-operator` target for
the exact `--version`):

```bash
helm repo add kfs https://kfsoftware.github.io/hlf-helm-charts
helm repo update
helm install hlf-operator --version=1.14.0 kfs/hlf-operator -n hlf
```

Verify:

```bash
kubectl -n hlf rollout status deploy/hlf-operator-controller-manager --timeout=180s
kubectl get crds | grep hlf.kungfusoftware.es
```

Expected CRDs include `fabriccas`, `fabricpeers`, `fabricorderernodes`,
`fabricmainchannels`, `fabricfollowerchannels`, `fabricchaincodes`.

Makefile targets: `make kind-up`, `make kind-operator`, `make kind-down`.

## Task 2: Certificate Authorities (org1, org2, orderer) (DONE)

Creates `org1-ca`, `org2-ca`, `ord-ca` via the operator, enroll id/pw
`enroll`/`enrollpw`, Fabric CA version 1.5.5, storage class `standard`, 1Gi:

```bash
bash infra/k8s/operator/scripts/10-cas.sh
```

Verify (operator provisions asynchronously — wait/re-run until `Running`):

```bash
kubectl -n hlf get fabriccas.hlf.kungfusoftware.es
kubectl -n hlf get pods -l app=hlf-ca
kubectl -n hlf wait --for=condition=ready pod -l app=hlf-ca --timeout=300s
```

Expected: `org1-ca`, `org2-ca`, `ord-ca` each `RUNNING`/`Running` and `1/1`.
Each CA is reachable in-cluster via its Kubernetes Service, e.g.
`org1-ca.hlf.svc.cluster.local:7054` (short form `org1-ca.hlf`).

Note: the plugin's `--hosts` flag (Istio ingress `Gateway`/`VirtualService`)
is intentionally omitted from the script — this kind cluster has no Istio
CRDs installed, and using `--hosts` makes the CA's chart install fail
(`FabricCA` stuck in `FAILED` state) since the operator can't create those
Istio resource kinds. In-cluster reachability, which is all later
enroll/register tasks need, comes from the plain Kubernetes `Service` the
operator creates regardless of `--hosts`.

Also note: this plugin version has no `kubectl hlf ca list` subcommand; use
`kubectl -n hlf get fabriccas.hlf.kungfusoftware.es` instead.

Makefile target: `make kind-cas`.

## Task 3: Orderer node (DONE)

Registers an `orderer` identity (type `orderer`, MSP `OrdererMSP`) at
`ord-ca`, then creates `orderer0` (Fabric orderer 2.5.15, storage class
`standard`, 1Gi):

```bash
bash infra/k8s/operator/scripts/20-orderer.sh
```

Verify:

```bash
kubectl -n hlf wait --for=condition=ready pod -l app=hlf-ordnode --timeout=300s
kubectl -n hlf get pods -l app=hlf-ordnode
kubectl -n hlf get fabricorderernodes.hlf.kungfusoftware.es
```

Expected: `orderer0` pod `Running` `1/1`, `FabricOrdererNode` `orderer0` in
state `RUNNING`.

Notes:

- Same as Task 2, the brief's `--hosts=orderer0.localho.st` flag on
  `ordnode create` is dropped — no Istio on this cluster, and `--hosts`
  would leave the resource `FAILED`. In-cluster reachability
  (`orderer0.hlf:7050`, needed by later channel tasks) comes from the
  operator's plain Kubernetes `Service` regardless.
- `ord-ca`'s TLS serving certificate only carries SANs
  `localhost`/`ord-ca`/`ord-ca.hlf`/`127.0.0.1` (Task 2's dropped `--hosts`
  controls Istio provisioning only, not the cert SAN list). Both
  `kubectl hlf ca register` (run from the host) and the in-cluster operator
  reconciling `ordnode create` auto-discover the CA via its NodePort + the
  kind node's external IP, which is **not** in that SAN list, so both fail
  TLS hostname verification (`x509: certificate is valid for 127.0.0.1,
  not <node-ip>`) if left to auto-discovery. The script works around this
  by: (1) port-forwarding `ord-ca`'s ClusterIP service to
  `127.0.0.1:17054` and passing `--ca-url=https://127.0.0.1:17054` to `ca
  register`; (2) passing `--ca-host=ord-ca.hlf --ca-port=7054` explicitly
  to `ordnode create` so the operator enrolls the orderer's certs via the
  in-cluster ClusterIP DNS name (a valid SAN) instead of the NodePort.
- The script tolerates `ca register` returning "already registered" (e.g.
  from a prior partial run) and continues to `ordnode create`.

Makefile target: `make kind-orderer`.

## Task 4: Peers (peer0-org1, peer0-org2) with CouchDB (DONE)

Registers a `peer` identity (type `peer`) at `org1-ca` / `org2-ca` for
`Org1MSP` / `Org2MSP` respectively, then creates `peer0-org1` and
`peer0-org2` (Fabric peer 2.5.15, storage class `standard`, 2Gi, CouchDB
state database):

```bash
bash infra/k8s/operator/scripts/30-peers.sh
```

Verify:

```bash
kubectl -n hlf wait --for=condition=ready pod -l app=hlf-peer --timeout=400s
kubectl -n hlf get pods -l app=hlf-peer
kubectl -n hlf get fabricpeers.hlf.kungfusoftware.es
```

Expected: `peer0-org1` and `peer0-org2` pods `Running` `2/2` (peer +
CouchDB containers both ready).

Notes:

- Same as Tasks 2-3, the brief's `--hosts=peer0-$org.localho.st` flag on
  `peer create` is dropped — no Istio on this cluster, and `--hosts` would
  leave the resource `FAILED`. In-cluster reachability (`peer0-org1.hlf`,
  `peer0-org2.hlf`, needed by later channel/chaincode tasks) comes from the
  operator's plain Kubernetes `Service` regardless.
- Same TLS SAN limitation as Task 3: each CA's serving certificate only
  carries SANs `localhost`/`<ca>`/`<ca>.hlf`/`127.0.0.1`, so
  NodePort/node-IP auto-discovery fails TLS hostname verification. The
  script works around this per org by: (1) port-forwarding the CA's
  ClusterIP service to a local port (`org1-ca` → `127.0.0.1:17055`,
  `org2-ca` → `127.0.0.1:17056`, distinct per org so the port-forwards
  don't collide) and passing `--ca-url=https://127.0.0.1:<port>` to `ca
  register`; (2) passing `--ca-host=<ca>.hlf --ca-port=7054` explicitly to
  `peer create` so the operator enrolls the peer's certs via the
  in-cluster ClusterIP DNS name (a valid SAN) instead of the NodePort.
- The script tolerates `ca register` returning "already registered" (e.g.
  from a prior partial run) and continues to `peer create`. `peer create`
  itself is not idempotent (a re-run errors with `already exists` on
  `peer0-org1` before reaching org2) — same behavior as `ordnode create`
  in Task 3; delete the `FabricPeer` CR and its PVC before recreating.

Makefile target: `make kind-peers`.

## Task 5a: Istio + CoreDNS (DONE)

Tasks 2-4 dropped the operator's `--hosts=*.localho.st` flag because this
kind cluster had no Istio, so the CA/orderer/peer resources are only
reachable in-cluster via plain Kubernetes Services. The channel task (join
via the orderer's channel-participation/admin API) needs the operator's
`FabricMainChannel` to reach the orderer through an Istio ingress host
(`orderer0.localho.st`); without Istio that host doesn't resolve and the
admin endpoint falls back to `node-IP:0`, which fails. This task installs
Istio (istiod + `istio-ingressgateway`, and the `networking.istio.io`
`Gateway`/`VirtualService` CRDs the operator creates when `--hosts` is
used) and adds a CoreDNS rewrite so `*.localho.st` resolves in-cluster to
the ingress gateway:

```bash
bash infra/k8s/operator/scripts/05-istio.sh
```

What it does (idempotent — safe to re-run):

1. Downloads `istioctl` 1.23.2 (k8s 1.31-compatible) to `~/.local/bin/istioctl`
   if not already on `PATH`.
2. Runs `istioctl install --set profile=default -y`. This installs `istiod`
   and an `istio-ingressgateway` `Service`/`Deployment` into a new
   `istio-system` namespace, plus the `networking.istio.io` CRDs
   (`gateways`, `virtualservices`, etc.). Sidecar injection is **not**
   enabled on the `hlf` namespace — the operator manages its own
   `Gateway`/`VirtualService` objects directly, so only the ingress gateway
   + CRDs are needed cluster-side.
3. Patches the `coredns` `ConfigMap` in `kube-system`: reads the current
   Corefile out of the cluster (so it doesn't clobber any other changes),
   inserts one line —
   `rewrite name regex (.*)\.localho\.st istio-ingressgateway.istio-system.svc.cluster.local`
   — inside the `.:53` server block (right after `ready`, before the
   `kubernetes` plugin so the rewrite is applied before cluster-service
   resolution), then re-applies the ConfigMap and restarts `coredns`. Skips
   the patch if the line is already present.

Verify:

```bash
kubectl -n istio-system get pods
kubectl get crd | grep networking.istio.io
kubectl -n istio-system get svc istio-ingressgateway -o jsonpath='{.spec.clusterIP}'
kubectl run dnstest --rm -i --restart=Never --image=busybox:1.36 -- nslookup orderer0.localho.st
```

Expected: `istiod-*` and `istio-ingressgateway-*` pods `Running` `1/1`;
`gateways.networking.istio.io` and `virtualservices.networking.istio.io`
(among others) listed; the `nslookup` resolves `orderer0.localho.st` to
`istio-ingressgateway.istio-system.svc.cluster.local` at the same address
as the ingress gateway's `CLUSTER-IP` (not NXDOMAIN). The gateway's
`EXTERNAL-IP` stays `<pending>` in kind (no cloud LoadBalancer
controller) — that's expected and fine, since only in-cluster resolution
via the `ClusterIP` is needed for the operator's admin-API calls.

Makefile target: `make kind-istio`.

## Task 5: Channel `mychannel` + peer joins (DONE)

Registers admin identities at each CA, enrolls them as `FabricIdentity` CRs
(both an MSP identity and a TLS identity per org), generates
`FabricMainChannel`/`FabricFollowerChannel` manifests from the live cluster
state, patches in the deltas below, and applies+waits for `RUNNING`:

```bash
bash infra/k8s/operator/scripts/40-channel.sh
```

Verify:

```bash
kubectl -n hlf get fabricmainchannels,fabricfollowerchannels
```

Expected: `mychannel` (`FabricMainChannel`) and `follower-mychannel-org1` /
`follower-mychannel-org2` (`FabricFollowerChannel`) all `RUNNING`.

Three deltas the manifests need beyond a naive `channelcrd main/follower
create` generation (all encoded in the script, all present in the committed
manifests):

1. **TLS admin identities.** The operator's osnadmin
   (channel-participation) call needs a TLS *client* identity, not just the
   MSP identity, or the join fails with `tls: certificate required`. The
   script enrolls a second `FabricIdentity` per org against the CA's
   `tlsca` profile (`caname: tlsca`), named `<org>-admin-tls`.
2. **`<MSP>-tls` entries in the channel `identities` map.** The
   `FabricMainChannel`'s `spec.identities` map needs `Org1MSP-tls`,
   `Org2MSP-tls`, `OrdererMSP-tls` keys pointing at those TLS identities'
   secrets (`secretKey: user.yaml`), alongside the plain `Org1MSP` /
   `Org2MSP` / `OrdererMSP` MSP-identity entries.
3. **Internal orderer join.** `ordererOrganizations[0].orderersToJoin`
   (in-cluster `{name, namespace}` ref) works once the TLS identity above
   is present — no Istio/external endpoint needed. Keep
   `externalOrderersToJoin: []`. (The followers' peer join stays on
   `externalPeersToJoin` — that path already worked and wasn't changed.)

Also note: `spec.orderers[0].tlsCert` (mainchannel) and each follower's
`spec.orderers[].certificate` must equal the live orderer's
`FabricOrdererNode.status.tlsCert`. The script re-stamps both from the live
value on every run; this is a no-op on a normal clean run (the orderer is
created once, before the channel is generated) and only matters if
`orderer0` was ever recreated (which would rotate its TLS cert) after the
manifests were last generated.

Makefile target: `make kind-channel`.

## Task 6: Chaincode `basic` as ccaas

Deploys `basic` (Go source: `packages/chaincode`) as chaincode-as-a-service
(ccaas) — a standalone Deployment/Service running a pre-built image, rather
than the peer spawning/compiling the chaincode itself — and commits it on
`mychannel` with endorsement `AND(Org1MSP.member, Org2MSP.member)`.

```bash
make kind-chaincode
```

Verify:

```bash
kubectl -n hlf get fabricchaincode,pod -l chaincode=basic
kubectl hlf chaincode querycommitted --config=<config> --user=admin --peer=peer0-org1.hlf --channel=mychannel
```

Expected: pod `basic-...` `Running`; `basic` listed committed on `mychannel`
at the script's `CC_VERSION`/`CC_SEQUENCE`; a live query against
`InventoryContract:GetInventory` returns (`[]` on an empty ledger) rather
than a "chaincode not found" error.

**ccaas support confirmed, no source change needed.** contract-api-go v2
(`contractapi.(*ContractChaincode).Start()`, in
`fabric-contract-api-go/v2@v2.0.0/contractapi/contract_chaincode.go`)
already checks `CHAINCODE_SERVER_ADDRESS` + `CORE_CHAINCODE_ID_NAME` and
branches into a real ccaas gRPC server (`shim.ChaincodeServer`) when both are
set, falling back to the peer-launched `shim.Start(cc)` stdio path
otherwise. `packages/chaincode/main.go`'s existing `cc.Start()` call is
sufficient; only `packages/chaincode/Dockerfile` needed to be added/set
`CHAINCODE_SERVER_ADDRESS`.

Four soft spots hit deploying this on the running cluster, all now encoded
in `scripts/50-chaincode.sh`:

1. **The ccaas Service is always on port 7052**, regardless of
   `externalchaincode sync`'s `--port` flag (that flag only rewrites the
   Deployment's env/probe port) — `connection.json`'s `address` and the
   chaincode image's listen port both need to be `7052` to match.
2. **`externalchaincode sync`'s update path drops custom `--env`** — a
   second `sync` against an existing `FabricChaincode` silently loses
   `CORE_CHAINCODE_ID_NAME` even though a *fresh* object populates it
   correctly. The script always deletes the `FabricChaincode` before
   syncing rather than relying on `sync`'s update/`--force` path.
3. **Reproducible `chaincode.tgz`.** `calculatepackageid` hashes the raw
   tgz bytes; a naive `tar czf` embeds file mtimes, so a byte-identical
   `connection.json`/`metadata.json` still produces a different package-id
   every run. That desyncs the freshly-deployed ccaas pod's
   `CORE_CHAINCODE_ID_NAME` from whatever package-id is actually approved
   and committed on the channel, and a `query` against the mismatched pod
   hangs (peer-side timeout) rather than failing fast. The script builds
   with `--sort=name --mtime='UTC 1970-01-01' --owner=0 --group=0
   --numeric-owner` so the package-id is stable run-to-run.
4. **`kubectl hlf chaincode install/approveformyorg/commit/query` run from
   the host, not from inside the cluster**, so they need externally
   reachable endpoints, not in-cluster `*.hlf` DNS. The script generates an
   SDK config via `kubectl hlf inspect` (Istio `*.localho.st` hostnames,
   set up in Task 5a — these resolve publicly to loopback) and tunnels
   through `kubectl port-forward`. `approveformyorg`'s (and `commit`'s
   submitting-org leg's) TxStatus event registration independently
   hardcodes `peer0-orgN.localho.st:443` — ignoring `--config`'s peer
   port entirely — so a literal privileged `:443` tunnel is required
   (`sudo kubectl port-forward svc/istio-ingressgateway 443:443`); a second,
   unprivileged tunnel carries the non-submitting org's leg during `commit`
   to avoid two concurrent SNI connections racing on one `kubectl
   port-forward` listener (confirmed empirically: one times out
   otherwise).

`CC_VERSION=1.1` / `CC_SEQUENCE=2` (not `1.0`/`1`): an interactive dry run
while developing this script — before tar output was made reproducible —
already committed a stray `1.0`/sequence-1 definition under a package-id
that doesn't match this script's deterministic build. Fabric lifecycle
sequences are append-only once committed, so the script targets the next
sequence rather than fighting the stale one; this has no effect on a
from-scratch cluster.

Idempotent-tolerant: re-running `make kind-chaincode` recomputes the same
deterministic package-id, redeploys the ccaas pod, and skips
approve/commit if `CC_VERSION`/`CC_SEQUENCE` is already committed
(re-approving/re-committing an already-committed sequence fails on this
cluster — confirmed empirically) — but always re-runs `install` (which is
itself idempotent) and the verification query.

## Task 7: Connect `api` + `web` (pending)

## Task 8: Full-run verification, retire skeleton chart, docs (pending)
