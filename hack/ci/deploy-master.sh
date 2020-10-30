#!/usr/bin/env bash
set -euo pipefail

release_url="$(/get-release-url-for-master.sh)"

echo "Downloading kubevirt tests binary from nightly build"
curl -L -o /bin/tests.test "${release_url}/testing/tests.test"
chmod +x /bin/tests.test

echo "Deploying kubevirt from nightly build"
oc create -f "${release_url}/kubevirt-operator.yaml"
oc create -f "${release_url}/kubevirt-cr.yaml"

echo "Deploying test infrastructure"
for testinfra_file in $(curl -L "${release_url}/testing/" | grep -oE 'https://[^"]*\.yaml'); do
    oc create -f "${testinfra_file}"
done
