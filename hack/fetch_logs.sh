#!/bin/bash

set -e

# Script to fetch KubeVirt PR logs from Google Cloud Storage
# Usage: ./fetch_logs.sh <url>

# Determine container runtime (podman or docker)
determine_cri_bin() {
    if [ "${KUBEVIRTCI_RUNTIME}" = "podman" ]; then
        echo podman
    elif [ "${KUBEVIRTCI_RUNTIME}" = "docker" ]; then
        echo docker
    else
        if curl --unix-socket "${XDG_RUNTIME_DIR}/podman/podman.sock" http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
            echo podman
        elif docker ps >/dev/null 2>&1; then
            echo docker
        else
            >&2 echo "no working container runtime found. Neither docker nor podman seems to work."
            exit 1
        fi
    fi
}

# Check if all required arguments are provided
if [ $# -ne 1 ]; then
    echo "Error: Missing required arguments"
    echo "Usage: $0 <url>"
    echo ""
    echo "Example:"
    echo "  $0 https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15955/pull-kubevirt-e2e-k8s-1.34-ipv6-sig-network/1983266997290405888"
    exit 1
fi

PROW_URL=$1

# Parse the URL to extract PR number, job name, and instance
# Expected format: https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/{PR}/{JOB_NAME}/{INSTANCE}
URL_PATH=$(echo "$PROW_URL" | sed 's|https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/||')

# Extract components
PR_NUMBER=$(echo "$URL_PATH" | cut -d'/' -f1)
JOB_NAME=$(echo "$URL_PATH" | cut -d'/' -f2)
INSTANCE=$(echo "$URL_PATH" | cut -d'/' -f3)

# Validate that we successfully parsed all components
if [ -z "$PR_NUMBER" ] || [ -z "$JOB_NAME" ] || [ -z "$INSTANCE" ]; then
    echo "Error: Failed to parse URL. Please provide a valid Prow dashboard URL."
    echo "Expected format: https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/{PR}/{JOB_NAME}/{INSTANCE}"
    exit 1
fi

# Construct the GCS path
GCS_PATH="gs://kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/${PR_NUMBER}/${JOB_NAME}/${INSTANCE}"

# Create output directory structure
OUTPUT_DIR="logs/${PR_NUMBER}/${JOB_NAME}/${INSTANCE}"
COPY_DEST="logs/${PR_NUMBER}/${JOB_NAME}"

# Remove existing directory if it exists
if [ -d "$OUTPUT_DIR" ]; then
    echo "error: folder already exists: $OUTPUT_DIR"
    exit 1
fi

mkdir -p "$COPY_DEST"

# Detect container runtime
CRI_BIN=$(determine_cri_bin)

echo "========================================"
echo "Fetching run logs"
echo "========================================"
echo "PR Number:  $PR_NUMBER"
echo "Job Name:   $JOB_NAME"
echo "Instance:   $INSTANCE"
echo "GCS Path:   $GCS_PATH"
echo "Output Dir: $OUTPUT_DIR"
echo "========================================"
echo ""

if [ -n "$CONTAINERIZED" ]; then
    ${CRI_BIN} run --rm \
        -v "$PWD:/data:Z" \
        docker.io/google/cloud-sdk:latest \
        gsutil -m cp -r "$GCS_PATH" "/data/$COPY_DEST/"
else
    if ! command -v gsutil &>/dev/null; then
        echo "Error: gsutil is not installed. Please install it or set CONTAINERIZED=1 to use containerized version."
        exit 1
    fi
    gsutil -m cp -r "$GCS_PATH" "$COPY_DEST/"
fi

echo ""
echo "========================================"
echo "Logs downloaded successfully!"
echo "Location: $OUTPUT_DIR"
echo "========================================"
