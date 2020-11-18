#!/usr/bin/env bash

set -ex

source ./hack/config

while IFS= read -r line; do
  if [[ $line != "" && $line != \#* ]]; then
    V="${line//=*/}"
    export "${V}=${!V}"
  fi
done < ./hack/config

./tools/digester/digester
