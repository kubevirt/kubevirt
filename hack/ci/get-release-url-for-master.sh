#!/usr/bin/env bash
set -euo pipefail

release_base_url="https://gcsweb.apps.ovirt.org/gcs/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt"
release_date=$(curl -L "${release_base_url}/latest")
echo "${release_base_url}/${release_date}"

