#!/usr/bin/bash

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
    local GHRELURL="https://github.com/kubevirt/kubevirt/releases/tag/"
    local RELURL="$GHRELURL$FUTURERELREF"
    cat <<EOF
The source code and selected binaries are available for download at:
<$RELURL>.

The primary release artifact of KubeVirt is the git tree. The release tag is
signed and can be verified using [git-evtag][git-evtag].

Pre-built containers are published on Docker Hub and can be viewed at:
<https://hub.docker.com/u/kubevirt/>.
EOF
}

shortlog() {
    git shortlog -sne $RELSPANREF | sed "s/^/    /"
}

functest() {
    log "Running functional tests - can take a while."
    cat .release-functest | tail -n5 |
        sed -r "s/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[m|K]//g" |
        egrep "(Ran|PASS)" |
        fold -sw 74 | sed -n "{ s/^/> / ; p }"
}

usage() {
    echo "Usage: $0 [FUTURE_RELEASE_REF] [PREV_RELEASE_REF]"
}

main() {
    log "Span: $RELSPANREF"

    fold -s <<EOF | tee release-announce
---
$(summary)

$(downloads)


$(section "Notable changes")

$(release_notes)


$(section "Contributors")

$(git shortlog -sne $RELSPANREF | wc -l) people contributed to this release:

$(shortlog)


Test Results
------------

$(functest)


Additional Resources
--------------------
- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/kubevirt/demo>
- [How to contribute][contributing]
- [License][license]


[git-evtag]: https://github.com/cgwalters/git-evtag#using-git-evtag
[contributing]: https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md
[license]: https://github.com/kubevirt/kubevirt/blob/master/LICENSE
---
git evtag sign $FUTURERELREF
EOF
}

#
# Let's get the party started
#
FUTURERELREF="$1"
RELREF="HEAD"
PREREF="$2"
RELREF=${RELREF:-$(git describe --abbrev=0 --tags)}
PREREF=${PREREF:-$(git describe --abbrev=0 --tags $RELREF^)}
RELSPANREF=$PREREF..$RELREF

main

# vim: sw=2 et
