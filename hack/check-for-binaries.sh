#!/bin/bash

if diff <(git grep -c '') <(git grep -cI '') |
    grep -E -v -e 'docs/.*\.png|swagger-ui' -e 'vendor/*' -e 'assets/*' -e 'pkg/downwardmetrics/vhostmd/testdata/vhostmd0' |
    grep '^<'; then
    echo "Binary files are present in git repostory."
    exit 1
fi
