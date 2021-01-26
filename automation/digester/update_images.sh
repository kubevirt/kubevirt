#!/usr/bin/env bash

set -ex

(
  cd ./tools/digester
  go build .
)

source ./hack/config

while IFS= read -r line; do
  if [[ $line != "" && $line != \#* ]]; then
    V="${line//=*/}"
    export "${V}=${!V}"
  fi
done < ./hack/config

export HCO_VERSION=${HCO_VERSION:-${CSV_VERSION}}
./tools/digester/digester
