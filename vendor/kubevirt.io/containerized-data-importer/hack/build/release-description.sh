#!/usr/bin/env bash

set -eou pipefail

underline() {
    echo "$2"
    printf "%0.s$1" $(seq ${#2})
}

log() { echo "$@" >&2; }
title() { underline "=" "$@"; }
section() { underline "-" "$@"; }

#
# All sorts of content
#
release_notes() {
    log "Fetching release notes"
    cat manual-release-notes || echo "FIXME manual notes needed"
}

summary() {
    log "Building summary"
    echo "This release follows $PREREF and consists of $(git log --oneline $RELSPANREF | wc -l) changes, contributed by"
    echo -n "$(git shortlog -sne $RELSPANREF | wc -l) people, leading to"
    echo "$(git diff --shortstat $RELSPANREF)."
}

downloads() {
    log "Adding download urls"
    local GHRELURL="https://github.com/kubevirt/containerized-data-importer/releases/tag/"
    local RELURL="$GHRELURL$RELREF"
    cat <<EOF
The source code and selected binaries are available for download at:
<$RELURL>.

Pre-built CDI containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/r/kubevirt/cdi-controller/>
<https://hub.docker.com/r/kubevirt/cdi-importer/>
<https://hub.docker.com/r/kubevirt/cdi-cloner/>
<https://hub.docker.com/r/kubevirt/cdi-uploadproxy/>
<https://hub.docker.com/r/kubevirt/cdi-apiserver/>
<https://hub.docker.com/r/kubevirt/cdi-uploadserver/>
EOF
}

shortlog() {
    git shortlog -sne $RELSPANREF | sed "s/^/    /"
}

usage() {
    echo "Usage: $0 [RELEASE_REF] [PREV_RELEASE_REF]"
}

main() {
    log "Span: $RELSPANREF"
    cat <<EOF | tee release-announcement
$(summary)

$(downloads)


$(section "Notable changes")

$(release_notes)


$(section "Contributors")

$(git shortlog -sne $RELSPANREF | wc -l) people contributed to this release:

$(shortlog)

Additional Resources
--------------------
- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- [How to contribute][contributing]
- [License][license]

[contributing]: https://github.com/kubevirt/containerized-data-importer/blob/master/hack/README.md
[license]: https://github.com/kubevirt/containerized-data-importer/blob/master/LICENSE
EOF
}

#
# Let's get the party started
#
RELREF="$1"
PREREF="$2"
RELREF=${RELREF:-$(git describe --abbrev=0 --tags)}
PREREF=${PREREF:-$(git describe --abbrev=0 --tags $RELREF^)}
RELSPANREF=$PREREF..$RELREF

main

# vim: sw=2 et
