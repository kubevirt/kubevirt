#!/usr/bin/env bash
set -euo pipefail

DOCKER_TAG="$1"

echo "Downloading kubevirt tests binary from release ${DOCKER_TAG}"
curl -Lo "/bin/tests.test" "https://github.com/kubevirt/kubevirt/releases/download/${DOCKER_TAG}/tests.test"
chmod +x "/bin/tests.test"

echo "Deploying kubevirt from release ${DOCKER_TAG}"
tagged_release_url="https://github.com/kubevirt/kubevirt/releases/download/${DOCKER_TAG}"
curl -L "${tagged_release_url}/kubevirt-operator.yaml" | oc create -f -
curl -L "${tagged_release_url}/kubevirt-cr.yaml" | oc create -f -

echo "Deploying test infrastructure"
testing_infra_url="https://gcsweb.apps.ovirt.org/gcs/kubevirt-prow/devel/release/kubevirt/kubevirt/${DOCKER_TAG}/manifests/testing/"
for testinfra_file in $(curl -L "${testing_infra_url}" | grep -oE 'https://[^"]*\.yaml'); do
    curl -L ${testinfra_file} | oc create -f -
done
