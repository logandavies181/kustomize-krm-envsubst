#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if [[ $# -gt 0 ]]
then
  dirs=$*
else
  dirs=$(find test/* -type d)
fi

echo "Compiling"
go build

# Set some env vars used in the test kustomization
export PORT=58008
export WORKERS=96
export LEAF_PEM=$(cat test/leaf.pem)
export INTER_PEM=$(cat test/inter.pem)
export ROOT_PEM=$(cat test/root.pem)
export TO_LOWERCASE=WAS_UPPERCASE
export INCLUDED_VAR=INCLUDE_THIS

for dir in $dirs
do
  echo "Kustomizing ${dir}"
  time kustomize build --enable-alpha-plugins --enable-exec "${dir}" > "${dir}/expected.yaml"

  echo "Running kubeconform on result of ${dir}"
  kubeconform \
    -schema-location default \
    -schema-location \
    'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
    "${dir}/expected.yaml"
  echo Success
done
