#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2022 Red Hat, Inc.
#
# This script checks the defaulting mechanism

set -ex

clean_nmap_output () {
  sed -i '/^$/d' $1
  sed -i '/^Starting Nmap.*$/d' $1
  sed -i '/^Nmap scan report for.*$/d' $1
  sed -i '/^Starting Nmap.*$/d' $1
  sed -i '/^Host is up.*$/d' $1
  sed -i '/^Nmap done:.*$/d' $1
  sed -i '/^MAC Address: .* (Unknown)$/d' $1
}

run_nmap () {
  nmap -T2 --max-parallelism 1 --max-retries 5 --script +ssl-enum-ciphers -p 4343 ${IPADDR} > $1
}

if ${KUBECTL_BINARY} get service -n ${INSTALLED_NAMESPACE} hyperconverged-cluster-webhook-service ; then
    SERVICE=service/hyperconverged-cluster-webhook-service
elif ${KUBECTL_BINARY} get service -n ${INSTALLED_NAMESPACE} hco-webhook-service ; then
    SERVICE=service/hco-webhook-service
else
    echo "Unable tp identify the service fot HCO webhook "
    exit -1
fi

if ! which nmap ; then
    echo "Try to install nmap"
    rpm -vhU --nodeps https://nmap.org/dist/nmap-7.92-1.x86_64.rpm
    rpm -vhU https://nmap.org/dist/ncat-7.92-1.x86_64.rpm
    rpm -vhU https://nmap.org/dist/nping-0.7.92-1.x86_64.rpm
fi

if [ -n "${OPENSHIFT_BUILD_NAMESPACE:-}" ]; then
  # on openshift-ci we are building with rhel-8-release-golang-1.18-openshift-4.11
  # which is FIPS compliant so only a subset of the allowed ciphers are available
  FIPS=".fips"
  IPADDR=$(${KUBECTL_BINARY} get pods -n "${INSTALLED_NAMESPACE}" -l name=hyperconverged-cluster-webhook -o jsonpath='{.items[0].status.podIP}')
  PF_PID=""
else
  FIPS=""
  IPADDR=127.0.0.1
  echo "Enable portforwarding to HCO webhook"
  ${KUBECTL_BINARY} port-forward -n ${INSTALLED_NAMESPACE} ${SERVICE} 4343:4343 &
  PF_PID=$!
  sleep 5
fi

${KUBECTL_BINARY} patch hco -n ${INSTALLED_NAMESPACE} --type=json kubevirt-hyperconverged -p '[{"op": "replace", "path": /spec/tlsSecurityProfile, "value": {old: {}, type: "Old"} }]'
sleep 2
run_nmap old.txt
clean_nmap_output old.txt
diff old.txt hack/tlsprofiles/old.expected${FIPS}

# nothing should happen in dry-run mode
${KUBECTL_BINARY} patch hco --dry-run=client -n ${INSTALLED_NAMESPACE} --type=json kubevirt-hyperconverged -p '[{"op": "replace", "path": /spec/tlsSecurityProfile, "value": {modern: {}, type: "Modern"} }]'
sleep 2
run_nmap old.txt
clean_nmap_output old.txt
diff old.txt hack/tlsprofiles/old.expected${FIPS}

${KUBECTL_BINARY} patch hco -n ${INSTALLED_NAMESPACE} --type=json kubevirt-hyperconverged -p '[{"op": "replace", "path": /spec/tlsSecurityProfile, "value": {intermediate: {}, type: "Intermediate"} }]'
sleep 2
run_nmap intermediate.txt
clean_nmap_output intermediate.txt
diff intermediate.txt hack/tlsprofiles/intermediate.expected${FIPS}

${KUBECTL_BINARY} patch hco -n ${INSTALLED_NAMESPACE} --type=json kubevirt-hyperconverged -p '[{"op": "replace", "path": /spec/tlsSecurityProfile, "value": {modern: {}, type: "Modern"} }]'
sleep 2
run_nmap modern.txt
clean_nmap_output modern.txt
diff modern.txt hack/tlsprofiles/modern.expected${FIPS}

${KUBECTL_BINARY} patch hco -n ${INSTALLED_NAMESPACE} --type=json kubevirt-hyperconverged -p '[{"op": "replace", "path": /spec/tlsSecurityProfile, "value": {custom: {minTLSVersion: "VersionTLS12", ciphers: ["ECDHE-ECDSA-CHACHA20-POLY1305", "ECDHE-ECDSA-AES256-GCM-SHA384", "AES256-GCM-SHA384", "AES128-SHA256"]}, type: "Custom"} }]'
run_nmap custom.txt
clean_nmap_output custom.txt
diff custom.txt hack/tlsprofiles/custom.expected${FIPS}

${KUBECTL_BINARY} patch hco -n ${INSTALLED_NAMESPACE} --type=json kubevirt-hyperconverged -p '[{"op": "remove", "path": /spec/tlsSecurityProfile }]'
sleep 2
run_nmap default.txt
clean_nmap_output default.txt
diff default.txt hack/tlsprofiles/intermediate.expected${FIPS}

if [ -n "$PF_PID" ]; then
  echo "Terminating port forwarding"
  kill ${PF_PID}
fi
