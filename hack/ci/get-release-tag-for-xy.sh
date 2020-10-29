#!/usr/bin/env bash
set -euo pipefail

KUBEVIRT_RELEASE="$1"

curl --fail -s https://api.github.com/repos/kubevirt/kubevirt/releases |
      jq -r '(.[].tag_name | select( test("-(rc|alpha|beta)") | not ) )' |
      sort -rV | grep "v$KUBEVIRT_RELEASE" | head -1
