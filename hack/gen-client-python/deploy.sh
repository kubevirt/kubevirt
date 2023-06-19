#!/usr/bin/env bash

set -exo pipefail

source $(dirname "$0")/../common.sh

GITHUB_FQDN=github.com
CLIENT_PYTHON_REPO=${CLIENT_PYTHON_REPO:-kubevirt/client-python}
CLIENT_PYTHON_DIR=/tmp/kubevirt-client-python

git clone \
    "https://${GIT_USER_NAME}@${GITHUB_FQDN}/${CLIENT_PYTHON_REPO}.git" \
    "${CLIENT_PYTHON_DIR}" >/dev/null 2>&1

# Remove content under kubevirt, docs and test directories
rm -rf "${CLIENT_PYTHON_DIR}"/{kubevirt,docs,test}
# Copy client-python into repository
cp -rf "${PYTHON_CLIENT_OUT_DIR}"/* "${CLIENT_PYTHON_DIR}/"

cd "${CLIENT_PYTHON_DIR}"

git config user.email "${GIT_AUTHOR_EMAIL:-kubevirtbot@redhat.com}"
git config user.name "${GIT_AUTHOR_NAME:-kubevirt-bot}"

CLIENT_UPDATED="false"
# Check api_client.py and configuration.py whether there are other changes
# except a 'version', which is getting updated regardless of changes in API.
for i in api_client.py configuration.py; do
    if [ "$(git diff --numstat -- "kubevirt/${i}" | cut -f 1)" != "1" ] &&
        [ -n "$(git diff --numstat -- "kubevirt/${i}" | cut -f 1)" ]; then
        CLIENT_UPDATED="true"
    fi
done
# Check if there are changes to commit, ignoring api_client.py & configuration.py
# which were tested above.
if git status --porcelain |
    grep 'kubevirt/' |
    grep -v 'kubevirt/\(api_client[.]py\|configuration[.]py\)' |
    grep --quiet "^ [AM]"; then

    CLIENT_UPDATED="true"
fi
# Push only in case something got changed in code.
if [ "${CLIENT_UPDATED}" = "true" ]; then
    git add -A .
    git commit --message "Client Python update by KubeVirt Prow build ${BUILD_ID}"

    git push origin master >/dev/null 2>&1
    echo "Client Python updated."
else
    echo "Client Python hasn't changed."
fi
