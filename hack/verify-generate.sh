#!/usr/bin/env bash

if [[ -n $(git status --porcelain 2>/dev/null) ]]; then
    echo "ERROR: git tree state is not clean!"
    echo "You probably need to run 'make generate' or 'make' and commit the changes"
    git status
    exit 1
fi
