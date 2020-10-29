#!/usr/bin/env bash
set -euo pipefail

release_base_url="https://gcsweb.apps.ovirt.org/gcs/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt"
release_date=$(curl -L "${release_base_url}/latest")
release_url="${release_base_url}/${release_date}"
commit=$(curl -L "${release_url}/commit")

echo "${release_date}_$(echo ${commit} | cut -c 1-9)"
