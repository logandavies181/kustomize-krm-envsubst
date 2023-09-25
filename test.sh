#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo "Compiling"
go build

# Set some env vars used in the test kustomization
export PORT=58008
export WORKERS=96
export LEAF_PEM=$(cat test/leaf.pem)
export INTER_PEM=$(cat test/inter.pem)
export ROOT_PEM=$(cat test/root.pem)

echo "Kustomizing"
time kustomize build --enable-alpha-plugins --enable-exec test > test/expected.yaml

echo "Running kubeconform"
kubeconform \
  -schema-location default \
  -schema-location \
  'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
  test/expected.yaml
