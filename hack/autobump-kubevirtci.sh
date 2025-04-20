#!/bin/bash
# Copyright 2018 The Kubernetes Authors.
# Copyright The KubeVirt Authors.
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
# Taken from https://github.com/kubernetes/test-infra/blob/4d7f26e59a5e186eef3a7de55486b7a40bbd79d7/hack/autodeps.sh
# and modified for kubevirt.

set -o errexit
set -o pipefail

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

cd $(git rev-parse --show-toplevel)

if [[ $# -lt 2 ]]; then
    echo "Usage: $(basename "$0") <github-login> </path/to/github/token> [git-name] [git-email]" >&2
    exit 1
fi
user=$1
token=$2
shift 2
if [[ $# -ge 2 ]]; then
    echo "git config user.name=$1 user.email=$2..." >&2
    git config user.name "$1"
    git config user.email "$2"
    shift 2
fi
if ! git config user.name &>/dev/null && git config user.email &>/dev/null; then
    echo "ERROR: git config user.name, user.email unset. No defaults provided" >&2
    exit 1
fi

hack/bump-kubevirtci.sh

git add -A
if git diff --name-only --exit-code HEAD; then
    echo "Nothing changed" >&2
    exit 0
fi

title="Run hack/bump-kubevirtci.sh, updating to ${kubevirtci_git_hash:0:8}..."
git commit -s -m "${title}"
git push -f "https://${user}@github.com/${user}/kubevirt.git" HEAD:kubevirtci

echo "Creating PR to merge ${user}:kubevirtci into main..." >&2
pr-creator \
    --github-token-path="${token}" \
    --org=kubevirt --repo=kubevirt --branch=main \
    --title="${title}" --match-title="Run hack/bump-kubevirtci.sh" \
    --body="Automatic kubevirtci update to ${kubevirtci_git_hash}. Please review" \
    --source="${user}":kubevirtci \
    --confirm
