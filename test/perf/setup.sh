#!/bin/bash
# test/perf/setup.sh

PARENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$PARENT_DIR/../.."

echo "Setting up Perf environment..."

helm upgrade --install k6-operator grafana/k6-operator

kubectl create configmap k6-test-assets \
  --from-file=load.js="$PARENT_DIR/scripts/load.js" \
  --from-file=gpu.proto="$REPO_ROOT/api/proto/device/v1alpha1/gpu.proto"

kubectl apply --recursive -f "$PARENT_DIR/infra/haproxy-bridge.yaml"
