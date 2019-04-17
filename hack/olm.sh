#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
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
# Copyright 2019 Red Hat, Inc.
#

set -e

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

if [ $# -ne 1 ]; then
    echo "usage: ${0} verify | push"
fi

BUNDLE_DIR=${OUT_DIR}/manifests/release/olm/bundle

echo "operator-courier version:"
operator-courier --version

echo "using these manifests:"
ls ${BUNDLE_DIR}

case ${1} in

verify)

    operator-courier verify --ui_validate_io ${BUNDLE_DIR}

    echo "olm verify success!"
    exit 0

    ;;

push)

    if [[ -z "$QUAY_USERNAME" ]] || [[ -z "$QUAY_USERNAME" ]] || [[ -z "$QUAY_REPOSITORY" ]]; then
        echo "please set QUAY_USERNAME, QUAY_PASSWORD and QUAY_REPOSITORY"
        exit 1
    fi

    echo "getting auth token from Quay"
    AUTH_TOKEN=$(curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
        {
            "user": {
                "username": "'"${QUAY_USERNAME}"'",
                "password": "'"${QUAY_PASSWORD}"'"
            }
        }' | jq -r '.token')

    echo "pushing bundle"
    operator-courier push "$BUNDLE_DIR" "$QUAY_REPOSITORY" "$package_name" "$csv_version" "$AUTH_TOKEN"

    ;;
esac
