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
# Copyright 2026 Red Hat, Inc.
#
# Manages Go test cache for CI. Inspired by:
# https://github.com/kubernetes/test-infra/pull/16623
#
# Uses the GCS JSON API directly (via curl + openssl + jq) so it works
# on all architectures including s390x where gsutil is unavailable.
#
# Usage:
#   hack/go-test-cache.sh save    - Upload cache to GCS after tests
#   hack/go-test-cache.sh restore - Download cache from GCS before testing
#
# Environment variables:
#   GO_TEST_CACHE_BUCKET - GCS bucket for storing the cache (required)
#                          e.g. "gs://kubevirt-prow/cache/go-test-cache"
#   GOOGLE_APPLICATION_CREDENTIALS - Path to GCS service account key (required
#                                    for GCS access)

set -euo pipefail

# --- Logging helpers ---
log_info() { echo "[go-test-cache] INFO:  $*"; }
log_warn() { echo "[go-test-cache] WARN:  $*" >&2; }
log_error() { echo "[go-test-cache] ERROR: $*" >&2; }
log_debug() { echo "[go-test-cache] DEBUG: $*"; }

CACHE_DIR="${GOCACHE:-$(go env GOCACHE)}"
CACHE_ARCHIVE="/tmp/go-test-cache.tar.gz"
GO_TEST_CACHE_BUCKET="${GO_TEST_CACHE_BUCKET:-}"

if [ -z "${GO_TEST_CACHE_BUCKET}" ]; then
    if [ "${1:-}" = "restore" ]; then
        log_info "GO_TEST_CACHE_BUCKET not set, skipping cache restore"
        exit 0
    fi
    log_error "GO_TEST_CACHE_BUCKET must be set"
    log_error "Example: export GO_TEST_CACHE_BUCKET=gs://kubevirt-prow/cache/go-test-cache"
    exit 1
fi

ARCH=$(uname -m)
case ${ARCH} in
x86_64* | amd64*) CACHE_ARCH="amd64" ;;
aarch64* | arm64*) CACHE_ARCH="arm64" ;;
s390x) CACHE_ARCH="s390x" ;;
*) CACHE_ARCH="${ARCH}" ;;
esac

GCS_BUCKET=$(echo "${GO_TEST_CACHE_BUCKET}" | sed 's|gs://||' | cut -d'/' -f1)
GCS_PREFIX=$(echo "${GO_TEST_CACHE_BUCKET}" | sed 's|gs://[^/]*/||')
GCS_OBJECT_NAME="${GCS_PREFIX}/cache-${CACHE_ARCH}.tar.gz"
GCS_BASE_URL="https://storage.googleapis.com"

# Print environment for debugging
log_debug "Command: $0 ${1:-}"
log_debug "Architecture: ${ARCH} -> ${CACHE_ARCH}"
log_debug "GOCACHE: ${CACHE_DIR}"
log_debug "GCS_BUCKET: ${GCS_BUCKET}"
log_debug "GCS_OBJECT_NAME: ${GCS_OBJECT_NAME}"
log_debug "GOOGLE_APPLICATION_CREDENTIALS: ${GOOGLE_APPLICATION_CREDENTIALS:-<not set>}"
if [ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]; then
    if [ -f "${GOOGLE_APPLICATION_CREDENTIALS}" ]; then
        log_debug "Credentials file exists: $(ls -la "${GOOGLE_APPLICATION_CREDENTIALS}")"
    else
        log_debug "Credentials file DOES NOT EXIST at: ${GOOGLE_APPLICATION_CREDENTIALS}"
    fi
fi

access_token=""
token_expiry=0

get_access_token() {
    if [ -z "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]; then
        log_error "GOOGLE_APPLICATION_CREDENTIALS is not set"
        return 1
    fi
    if [ ! -f "${GOOGLE_APPLICATION_CREDENTIALS}" ]; then
        log_error "Credentials file not found: ${GOOGLE_APPLICATION_CREDENTIALS}"
        return 1
    fi

    # Reuse token if still valid
    if [ -n "$access_token" ] && [ "$(date +%s)" -lt "$token_expiry" ]; then
        log_debug "Reusing cached access token (expires in $((token_expiry - $(date +%s)))s)"
        return 0
    fi

    log_debug "Generating new access token..."

    local sa_email sa_key jwt_header jwt_claim jwt_signature jwt response
    sa_email=$(jq -r '.client_email' "${GOOGLE_APPLICATION_CREDENTIALS}")
    if [ -z "${sa_email}" ] || [ "${sa_email}" = "null" ]; then
        log_error "Failed to extract client_email from credentials file"
        log_error "File contents (keys only): $(jq -r 'keys[]' "${GOOGLE_APPLICATION_CREDENTIALS}" 2>&1)"
        return 1
    fi
    log_debug "Service account: ${sa_email}"

    sa_key=$(jq -r '.private_key' "${GOOGLE_APPLICATION_CREDENTIALS}")
    if [ -z "${sa_key}" ] || [ "${sa_key}" = "null" ]; then
        log_error "Failed to extract private_key from credentials file (key field missing or empty)"
        return 1
    fi
    log_debug "Private key loaded (length: ${#sa_key} chars)"

    jwt_header=$(echo -n '{"alg":"RS256","typ":"JWT"}' | base64 -w 0 | tr '+/' '-_' | tr -d '=')
    jwt_claim=$(echo -n '{"iss":"'"${sa_email}"'","scope":"https://www.googleapis.com/auth/devstorage.read_write","aud":"https://oauth2.googleapis.com/token","exp":'"$(($(date +%s) + 3600))"',"iat":'"$(date +%s)"'}' | base64 -w 0 | tr '+/' '-_' | tr -d '=')
    jwt_signature=$(echo -n "${jwt_header}.${jwt_claim}" | openssl dgst -binary -sha256 -sign <(echo "${sa_key}") | base64 -w 0 | tr '+/' '-_' | tr -d '=')
    jwt="${jwt_header}.${jwt_claim}.${jwt_signature}"

    response=$(curl -s -X POST "https://oauth2.googleapis.com/token" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=${jwt}")

    access_token=$(echo "${response}" | jq -r '.access_token')
    token_expiry=$(($(date +%s) + 3500))

    if [ -z "${access_token}" ] || [ "${access_token}" = "null" ]; then
        log_error "Failed to obtain access token from Google OAuth2"
        log_error "Service account: ${sa_email}"
        log_error "OAuth2 error: $(echo "${response}" | jq -r '.error // "unknown"')"
        log_error "OAuth2 error_description: $(echo "${response}" | jq -r '.error_description // "none"')"
        return 1
    fi

    log_debug "Access token obtained successfully (length: ${#access_token})"
}

urlencode_path() {
    echo "$1" | sed 's/\//%2F/g'
}

gcs_upload() {
    local file="$1"
    local file_size
    file_size=$(du -sh "${file}" | cut -f1)
    log_info "Uploading ${file} (${file_size}) to gs://${GCS_BUCKET}/${GCS_OBJECT_NAME}"

    local encoded_object
    encoded_object=$(urlencode_path "${GCS_OBJECT_NAME}")

    get_access_token || return 1

    # Initiate a resumable upload — returns a session URI in the Location header
    local initiate_url="${GCS_BASE_URL}/upload/storage/v1/b/${GCS_BUCKET}/o?uploadType=resumable&name=${encoded_object}"
    log_debug "Initiating resumable upload: ${initiate_url}"

    local upload_uri
    upload_uri=$(curl -s -i -X POST \
        -H "Authorization: Bearer ${access_token}" \
        -H "Content-Type: application/gzip" \
        -H "Content-Length: 0" \
        "${initiate_url}" | grep -i "^location:" | tr -d '\r' | sed 's/^[Ll]ocation: //')

    if [ -z "${upload_uri}" ]; then
        log_error "Failed to initiate resumable upload (no Location header returned)"
        return 1
    fi
    log_debug "Resumable upload URI obtained"

    # Stream the file to GCS using PUT -T (no memory buffering)
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -X PUT \
        -T "${file}" \
        -H "Content-Type: application/gzip" \
        "${upload_uri}")

    if [ "${http_code}" -ge 200 ] && [ "${http_code}" -lt 300 ]; then
        log_info "Upload successful (HTTP ${http_code})"
        return 0
    else
        log_error "GCS upload failed for gs://${GCS_BUCKET}/${GCS_OBJECT_NAME}"
        log_error "HTTP status: ${http_code}"
        return 1
    fi
}

gcs_download() {
    local file="$1"
    local encoded_object
    encoded_object=$(urlencode_path "${GCS_OBJECT_NAME}")

    get_access_token || return 1

    local download_url="${GCS_BASE_URL}/storage/v1/b/${GCS_BUCKET}/o/${encoded_object}?alt=media"
    log_debug "Download URL: ${download_url}"

    local http_code curl_exit
    http_code=$(curl -s -o "${file}" -w "%{http_code}" \
        --retry 3 \
        --retry-delay 5 \
        -H "Authorization: Bearer ${access_token}" \
        "${download_url}") || curl_exit=$?
    curl_exit=${curl_exit:-0}

    log_debug "Download HTTP response code: ${http_code}, curl exit code: ${curl_exit}"

    if [ "${curl_exit}" -ne 0 ]; then
        log_error "curl failed with exit code ${curl_exit} (transfer incomplete or network error)"
        rm -f "${file}"
        return 1
    fi

    if [ "${http_code}" = "404" ]; then
        log_warn "Cache object not found (HTTP 404): gs://${GCS_BUCKET}/${GCS_OBJECT_NAME}"
        rm -f "${file}"
        return 1
    elif [ "${http_code}" -ge 200 ] && [ "${http_code}" -lt 300 ]; then
        # Verify the archive is not truncated
        if ! gzip -t "${file}" 2>/dev/null; then
            log_error "Downloaded file is corrupted or truncated (gzip integrity check failed)"
            local actual_size
            actual_size=$(du -sh "${file}" | cut -f1)
            log_error "Downloaded size: ${actual_size}"
            rm -f "${file}"
            return 1
        fi
        local file_size
        file_size=$(du -sh "${file}" | cut -f1)
        log_info "Download successful (${file_size}, integrity verified)"
        return 0
    else
        log_error "GCS download failed with HTTP ${http_code}"
        log_error "Object: gs://${GCS_BUCKET}/${GCS_OBJECT_NAME}"
        rm -f "${file}"
        return 1
    fi
}

cache_save() {
    log_info "=== Go Test Cache Save ==="

    if [ ! -d "${CACHE_DIR}" ]; then
        log_warn "Cache directory ${CACHE_DIR} does not exist, nothing to save"
        return 0
    fi

    local cache_size num_files
    cache_size=$(du -sh "${CACHE_DIR}" 2>/dev/null | cut -f1)
    num_files=$(find "${CACHE_DIR}" -type f | wc -l)
    log_info "Cache directory: ${CACHE_DIR}"
    log_info "Cache size (uncompressed): ${cache_size} (${num_files} files)"

    log_info "Compressing cache..."
    time tar -czf "${CACHE_ARCHIVE}" -C "${CACHE_DIR}" .
    local archive_size
    archive_size=$(du -sh "${CACHE_ARCHIVE}" | cut -f1)
    log_info "Cache size (compressed): ${archive_size}"

    log_info "Uploading cache..."
    time gcs_upload "${CACHE_ARCHIVE}"

    rm -f "${CACHE_ARCHIVE}"
    log_info "=== Cache save completed successfully ==="
}

cache_restore() {
    log_info "=== Go Test Cache Restore ==="

    log_info "Downloading cache..."
    if ! time gcs_download "${CACHE_ARCHIVE}"; then
        log_warn "Cache restore failed — tests will run without cache (slower but functional)"
        return 0
    fi

    local archive_size
    archive_size=$(du -sh "${CACHE_ARCHIVE}" | cut -f1)
    log_info "Downloaded cache archive: ${archive_size}"

    log_info "Decompressing cache to ${CACHE_DIR}..."
    mkdir -p "${CACHE_DIR}"
    time tar -xzf "${CACHE_ARCHIVE}" -C "${CACHE_DIR}"

    rm -f "${CACHE_ARCHIVE}"

    local cache_size num_files
    cache_size=$(du -sh "${CACHE_DIR}" 2>/dev/null | cut -f1)
    num_files=$(find "${CACHE_DIR}" -type f | wc -l)
    log_info "Cache restored: ${cache_size} (${num_files} files)"
    log_info "=== Cache restore completed successfully ==="
}

case "${1:-}" in
save)
    cache_save
    ;;
restore)
    cache_restore
    ;;
*)
    log_error "Usage: $0 {save|restore}"
    exit 1
    ;;
esac
