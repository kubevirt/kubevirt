#!/usr/bin/env bash

set -euo pipefail

curl -H 'Cache-Control: no-cache' https://raw.githubusercontent.com/fossas/fossa-cli/master/install-latest.sh | bash
FOSSA_OPTS=""
if [[ "${CI:-}" == "true" ]]; then
    if [[ "${JOB_TYPE:-}" == "presubmit" ]]; then
        FOSSA_OPTS="--branch=pr/${PULL_NUMBER}"
    else
        FOSSA_OPTS="--branch=${PULL_BASE_REF}"
    fi
fi
export ARTIFACTS="${ARTIFACTS:-_out/fossa}"
mkdir -p "${ARTIFACTS}"
FOSSA_API_KEY="$(<"$FOSSA_TOKEN_FILE")" fossa analyze "${FOSSA_OPTS}" \
  --tee-output > "${ARTIFACTS}/analysis.json"
FOSSA_API_KEY="$(<"$FOSSA_TOKEN_FILE")" fossa test --timeout=1800
