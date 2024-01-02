#!/bin/bash
# Usage:
#
#     git difftool -y --extcmd=hacks/diff-csv.sh
#
# currently used for make sanity
diff --unified=1 --ignore-matching-lines='^\s*createdAt:' --ignore-matching-lines='^\s*\* Copyright [0-9]\{4\} Red Hat, Inc\.' $@
