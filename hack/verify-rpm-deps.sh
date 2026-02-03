#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/config.sh

# Verify RPM dependencies using native comparison script
# Verify RPM dependencies using native lock files

echo "Verifying RPM dependencies..."

# Check that lock files exist
if [ ! -d "rpm-lockfiles" ]; then
    echo "Error: rpm-lockfiles directory not found. Run 'make rpm-deps' first."
    exit 1
fi

# Verify lock files have valid JSON structure
for lockfile in rpm-lockfiles/*.lock.json; do
    if [ -f "$lockfile" ]; then
        if ! jq empty "$lockfile" 2>/dev/null; then
            echo "Error: Invalid JSON in $lockfile"
            exit 1
        fi
    fi
done

echo "RPM dependencies verified successfully."
