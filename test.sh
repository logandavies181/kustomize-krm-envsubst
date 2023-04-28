#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo "Compiling"
go build

echo "Kustomizing"
PORT=58008 WORKERS=96 kustomize build --enable-alpha-plugins --enable-exec test > test/expected.yaml

echo "Running kubeconform"
kubeconform test/expected.yaml
