#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

dir="$PWD"

rm --recursive --force fieldType/schemas
mkdir --parents fieldType/schemas/{native,crds}

tempdir=$(mktemp --directory)
trap 'rm --recursive --force $tempdir' EXIT
cd "$tempdir"

# https://askubuntu.com/a/1074185
git clone \
    --no-checkout \
    --depth 1 \
    --filter=tree:0 \
    https://github.com/yannh/kubernetes-json-schema.git
cd kubernetes-json-schema
git sparse-checkout set --no-cone master-standalone
git checkout

cd "$dir"

mv "${tempdir}/kubernetes-json-schema/master-standalone" fieldType/schemas/native

cd "$tempdir"
git clone \
    --depth 1 \
    https://github.com/datreeio/CRDs-catalog.git
cd "$dir"

find "${tempdir}/CRDs-catalog" -mindepth 1 -maxdepth 1 -type d -not -path '.git/*' \
    -exec mv {} fieldType/schemas/crds \;
