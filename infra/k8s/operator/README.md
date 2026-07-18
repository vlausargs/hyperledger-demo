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

## Task 5: Channel `mychannel` + peer joins (pending)

## Task 6: Chaincode `basic` as ccaas (pending)

## Task 7: Connect `api` + `web` (pending)

## Task 8: Full-run verification, retire skeleton chart, docs (pending)
