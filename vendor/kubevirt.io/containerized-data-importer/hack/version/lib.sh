#!/usr/bin/env bash

# lib.sh

# doIncrement replaces the oldVersion value with the newVersion in the given file
function setNewVersion() {
    local file=$1
    local oldVersion=$2
    local newVersion=$3

    sed -i "s#$oldVersion#$newVersion#g" $file
}

# pushNewVersion indexes only the files where versions are known to be specified,
# commits the changes, and pushes to master
# Parameters:
#   $1: the new version tag
#   $@: Known files containing an updated version value
function commitAndTag() {
    local new_tag_name="$1"
    shift
    local files="$@"
    printf "Adding changed files\n"
    for f in "$files"; do
        printf "Adding: %s\n" $files
        git add $f
    done
    printf "Commiting changed files\n"
    git commit -m "Update Version"
    printf "Creating new tag for commit (%s)\n" $new_tag_name
    git tag -f -a -m "Update Version" $new_tag_name
}

function verifyOnMaster() {
    local branch="master"
    printf "Verifying current branch is %s\n" "$branch"
    if [ "$(git rev-parse --abbrev-ref HEAD)" != "$branch" ]; then
        printf "Please checkout %s branch before continuing.\n" $branch
        exit 1
    fi
}

function verifyNoDiff() {
    printf "Checking commit diff between local and upstream master branches\n"
    local upstream="$(git remote -v | grep 'kubevirt/containerized-data-importer' | awk 'NR==1{print $1}')"
    local curBranch="$(git rev-parse --abbrev-ref HEAD)"

    if [ -z "$upstream" ]; then
        printf "No upstream remote repository detected, cannot verify commit differences\n"
        exit 1
    fi
    if [ -z "$curBranch" ]; then
        printf "No current branch was found, exiting\n"
        exit 1
    fi

    if ! git fetch "$upstream" master; then
        exit 1
    fi

    if [ -n "$(git rev-list --left-right "$upstream/master"..."$curBranch")" ]; then
        printf "Detected commit difference between %s and current branch (%s).  Merge/rebase %s and retry.\n" "$upstream" "$curBranch" "$upstream"
        exit 1
    fi
    printf "Verified local master matches %s\n" "$upstream"
}

function verifyVersionFormat() {
    printf "Validating version format\n"
    local newVersion="$1"
    # TODO improve regex to handle *-alpha.# suffixes
    if ! [[ "$newVersion" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
        printf "User defined version '%s' does not match semantic version format (v#.#.#*)\n" "$newVersion"
        exit 1
    fi
}

function getCurrentVersion() {
    local cv="$(git describe --tags --abbrev=0 HEAD)"
    if [ -z $cv ]; then
        exit 1
    fi
    printf $cv
}

function getVersionedFiles() {
    local curVersion=$1
    local repoRoot=$2
    local verFiles="$(grep -rl --exclude-dir={vender,bin,.git} --exclude={glide.*,.git*} "$curVersion" $repoRoot/ 2>/dev/null)"
    if [ -z "$verFiles" ]; then
        printf "" # this func exec'd inside subshell, so return null string
        exit 1
    fi
    printf "$(echo $verFiles | tr '\n' ' ')"
}

function acceptChanges() {
    local curVersion=$1
    local newVersion=$2
    local targetFiles=$3

    printf "\nThe version will be changed from '%s' to '%s' in:\n" "$curVersion" "$newVersion"
    printf "%s\n" "$(echo $targetFiles | tr ' ' '\n')"
    read -n 1 -p "Do you accept these changes [N|y]: " key
    printf "\n"
    case "$key" in
    y | Y)
        printf "Continuing with version update\n"
        ;;

    ?)
        printf "Aborting\n"
        exit 1
        ;;
    esac
}
