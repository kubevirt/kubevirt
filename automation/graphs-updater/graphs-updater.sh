#!/usr/bin/env bash

set -euo pipefail

if [[ "$#" -ne 1 ]]; then
    echo "Illegal number of parameters"
    echo "Usage: <target-for-graphs>"
fi

PREFIX=https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test/logs
JOB_NAME=periodic-ci-kubevirt-hyperconverged-cluster-operator-main-hco-e2e-deploy-nightly-main-aws
SUFFIX=artifacts/hco-e2e-deploy-nightly-main-aws/test/artifacts

declare -a filelist=("component.gv" "component.gv.svg" "managed-by.gv" "managed-by.gv.svg")
graphs_files_dir="$1"

LATEST_BUILD=$(curl -L ${PREFIX}/${JOB_NAME}/latest-build.txt)
ARTIFACTS_FOLDER=${PREFIX}/${JOB_NAME}/${LATEST_BUILD}/${SUFFIX}

for f in "${filelist[@]}"
do
   curl "${ARTIFACTS_FOLDER}/${f}" -f -s -o "${graphs_files_dir}/${f}"
done
