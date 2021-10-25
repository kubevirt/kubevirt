#!/usr/bin/env bash

set -euo pipefail

curl -H 'Cache-Control: no-cache' https://raw.githubusercontent.com/fossas/fossa-cli/master/install.sh | bash
FOSSA_API_KEY="$(cat $FOSSA_TOKEN_FILE)" fossa analyze
FOSSA_API_KEY="$(cat $FOSSA_TOKEN_FILE)" fossa test --timeout=1800
