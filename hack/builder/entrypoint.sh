#!/usr/bin/env bash
set -e
set -o pipefail

source /etc/profile.d/gimme.sh
export GOPATH="/root/go"
eval "$@"
