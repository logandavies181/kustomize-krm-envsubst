#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset 

trap 'rm -rf /tmp/kubernetes-json-schema' EXIT

rm -rf fieldType/master-standalone-strict

(
  cd /tmp
  git clone \
    --depth 1\
    --filter=blob:none \
    --sparse git@github.com:yannh/kubernetes-json-schema.git

  cd kubernetes-json-schema
  git sparse-checkout set master-standalone-strict
)

mv /tmp/kubernetes-json-schema/master-standalone-strict fieldType
