#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

go build
PORT=58008 kustomize build --enable-alpha-plugins --enable-exec test > test/expected.yaml

kubeconform test/expected.yaml
