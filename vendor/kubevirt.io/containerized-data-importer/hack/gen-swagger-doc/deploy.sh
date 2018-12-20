#!/usr/bin/env bash

set -e

GITHUB_FQDN=github.com
API_REF_REPO=kubevirt/cdi-api-reference
API_REF_DIR=/tmp/cdi-api-reference
GITHUB_IO_FQDN="https://kubevirt.io/cdi-api-reference"

TARGET_DIR="$TRAVIS_BRANCH"
if [ -n "${TRAVIS_TAG}" ]; then
    TARGET_DIR="$TRAVIS_TAG"
fi

rm -rf "${API_REF_DIR}"
git clone \
    "https://${API_REFERENCE_PUSH_TOKEN}@${GITHUB_FQDN}/${API_REF_REPO}.git" \
    "${API_REF_DIR}" >/dev/null 2>&1
mkdir -p ${API_REF_DIR}/${TARGET_DIR}
cp -f _out/apidocs/html/*.html "${API_REF_DIR}/${TARGET_DIR}/"

cd "${API_REF_DIR}"

# Generate README.md file
cat >README.md <<__EOF__
# KubeVirt Containerized Data Importer API Reference

Content of this repository is generated from OpenAPI specification of
[KubeVirt Containerized Data Importer project](https://github.com/kubevirt/containerized-data-importer) .

## KubeVirt Containerized Data Importer API References

* [master](${GITHUB_IO_FQDN}/master/index.html)
__EOF__
find * -type d -regex "^v[0-9.]*" \
    -exec echo "* [{}](${GITHUB_IO_FQDN}/{}/index.html)" \; >>README.md

git config --global user.email "travis@travis-ci.org"
git config --global user.name "Travis CI"

# NOTE: exclude index.html from match, becasue it is static except commit hash.
if git status --porcelain | grep -v "index[.]html" | grep --quiet "^ [AM]"; then
    git add -A README.md "${TARGET_DIR}"/*.html
    git commit --message "API Reference update by Travis Build ${TRAVIS_BUILD_NUMBER}"

    git push origin master >/dev/null 2>&1
    echo "API Reference updated for ${TARGET_DIR}."
else
    echo "API Reference hasn't changed."
fi
