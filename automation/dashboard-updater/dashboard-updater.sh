#!/usr/bin/env bash

set -euo pipefail

if [[ "$#" -ne 2 ]]; then
    echo "Illegal number of parameters"
    echo "Usage: ./dashboard-updater.sh <source-of-json-files> <target-for-configmaps>"
fi

cat << EOF > /tmp/fake-config
apiVersion: v1
clusters:
- cluster:
    server: fake:6443
  name: fake-cluster
contexts:
- context:
    cluster: fake-cluster
    namespace: no-ns
  name: fake-context
current-context: fake-context
kind: Config
preferences: {}
EOF

# See https://github.com/kubernetes/kubernetes/issues/51475
export KUBECONFIG=/tmp/fake-config

json_files_dir="$1"
configmaps_files_dir="$2"

rm -Rf "$configmaps_files_dir"
mkdir -p "$configmaps_files_dir"

echo "Generating configmaps for dashboards"

for file in "${json_files_dir}"/*.json; do
    echo "Generating configmap for $file"
    file_name=${file##*/}
    file_name_without_extension=${file_name::${#file_name}-5}

    configmap_name="grafana-dashboard-$file_name_without_extension"
    kubectl create configmap "$configmap_name" \
      --namespace openshift-config-managed \
      --dry-run=client -o yaml \
      --from-file="$file_name=$file" | \
    kubectl label -f - --dry-run=client  -o yaml \
      --local 'console.openshift.io/dashboard=true' | \
    grep -v "creationTimestamp" > "${configmaps_files_dir}/${configmap_name}.yaml"
done
