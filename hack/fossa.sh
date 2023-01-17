#!/usr/bin/env bash

set -euo pipefail

curl -H 'Cache-Control: no-cache' https://raw.githubusercontent.com/fossas/fossa-cli/master/install-latest.sh | bash
FOSSA_OPTS=""
if [[ "${CI:-}" == "true" ]]; then
    FOSSA_OPTS="--branch=$PULL_BASE_REF"
fi
FOSSA_API_KEY="$(cat $FOSSA_TOKEN_FILE)" fossa analyze $FOSSA_OPTS
FOSSA_API_KEY="$(cat $FOSSA_TOKEN_FILE)" fossa test --timeout=1800
