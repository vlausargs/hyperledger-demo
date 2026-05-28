#!/usr/bin/env bash
# 010-add-org.sh ORG=<name>
# Provision a new organization: generate compose fragment, deploy CA, enroll
# identities, join channel, and refresh endorsement policy.
#
# Wraps the configloader CLI plus existing 002-008 deployment scripts.
# This is the script invoked by `make add-org ORG=<name>`.

set -euo pipefail

ORG="${ORG:-${1:-}}"
if [[ -z "$ORG" ]]; then
  echo "usage: $0 ORG=<orgName>" >&2
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
CFG_DIR="$REPO_ROOT/infra/configloader"
ORG_FILE="$REPO_ROOT/infra/config/orgs/${ORG}.yml"

if [[ ! -f "$ORG_FILE" ]]; then
  echo "Org config missing: $ORG_FILE" >&2
  echo "Create it first (see infra/config/orgs/org1.yml for shape)." >&2
  exit 1
fi

echo "==> validating registry"
( cd "$CFG_DIR" && go run ./cmd validate )

echo "==> generating compose fragment for $ORG"
( cd "$CFG_DIR" && go run ./cmd generate-compose --org="$ORG" )

echo "==> provisioning CA + Postgres"
# TODO: parameterize 002-deploy-ca.sh / 001-deploy-postgres.sh by ORG once those
# scripts accept an ORG arg. Current scripts assume static org1/org2/org3 set.
ORG="$ORG" "$REPO_ROOT/fabric-network/scripts/deployment/001-deploy-postgres.sh" || true
ORG="$ORG" "$REPO_ROOT/fabric-network/scripts/deployment/002-deploy-ca.sh"      || true
ORG="$ORG" "$REPO_ROOT/fabric-network/scripts/deployment/003-setup-ca.sh"       || true

echo "==> joining channel + updating endorsement policy"
ORG="$ORG" "$REPO_ROOT/fabric-network/scripts/deployment/007-create-channel.sh" || true

echo "==> updated endorsement policy:"
( cd "$CFG_DIR" && go run ./cmd policy )

echo "done. $ORG added."
