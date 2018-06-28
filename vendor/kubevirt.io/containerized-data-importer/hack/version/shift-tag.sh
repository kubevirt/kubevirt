#!/usr/bin/env bash

# shift-tag.sh

# shiftTag deletes the human defined tag in the remote repo
# and sets it again for the current commit. After the CI has updated
# the version values in the project, a new commit will be pushed to origin.
# This will cause the human defined tag to fall 1 behind the commit where the
# values are changed.  Thus, it is necessary to "shift" the tag by one.
# Note: this initiates a 2nd run of the CI, which is short circuited below
# by comparing tags.
function shiftTag(){
    local versionTag=$1

    git push --delete origin $versionTag # delete the remote stale tag
    git tag -f -a -m "[ci skip] shift existing tag to HEAD" $versionTag
    git push origin $versionTag
}


REPO_ROOT="$(readlink -f $(dirname $0)/../../)"

source $REPO_ROOT/hack/version/lib.sh

CUR_VERSION=$(getCurrentVersion)
verifyVersionFormat "$CUR_VERSION"
shiftTag "$CUR_VERSION"
