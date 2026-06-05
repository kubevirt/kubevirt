# Overview

This document describes the technical procedure for cutting releases.
Refer to [release.md](release.md) in order to learn about the high-level release process.

# Creating Releases

The actual releases are all cut using the kubevirt release tool. This tool
automates the entire process of creating branches, signing tags, generating
prow configs, and more. To use the release tool, you need to prepare your
credentials.

## Release Tool Credentials

You must have 2 items before you can create a release.

1. **GPG key and passphrase files used for creating signed releases.**

[Instructions for adding GPG key to your github account](https://docs.github.com/en/authentication/managing-commit-signature-verification/adding-a-gpg-key-to-your-github-account)

After adding the GPG key to Gitub, export both the key and the passphrase to 
separate files.
Be aware that this results in both the key and passphrase being placed into
plain text files on your machine. Make sure you don't place this in shared
storage.

**Example of exporting key to file**

    gpg --export-secret-key -a <your-email-address> > ${HOME}/gpg-private

**Example of putting passphrase in file**

    echo "<insert your passphrase here>" > ${HOME}/gpg-passphrase

2. **Github API token file used for accessing the github api**

When you create this token the only permission you need to give it is the
ability to access github repositories. That's it.

[Instructions for creating _classic_ access tokens](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token)

**Place your token in a text file such as** `${HOME}/github-api-token`

## Release Tool Usage Examples

Once you have your credentials in files, the kubevirt release tool handles
all the rest. All you need to do is provide your credentials and tell the tool
what release you want to make.

Place the paths to your credential files in the following environment variables.

    export GPG_PRIVATE_KEY_FILE="${HOME}/gpg-private"
    export GPG_PASSPHRASE_FILE="${HOME}/gpg-passphrase"
    export GITHUB_API_TOKEN_FILE="${HOME}/github-api-token"

> **Note**
> The release tool is running in a container. SELinux can lead to file
> access issues. Thus look at your audit log if you experience file
> access issues.

Now you can use the release tool to do whatever you'd like. Note that you can
use the `--dry-run=true` (used by default) argument to test a change before executing it.

**Example: creating a new release branch with the initial release candidate v0.31.0-rc.0**

    hack/release.sh --new-branch release-0.31 --new-release v0.31.0-rc.0 --dry-run=false

**Example: Creating a new rc v0.31.0-rc.0**

    hack/release.sh --new-release v0.31.0-rc.0 --dry-run=false

**Example: Promoting a release candidate v0.31.0-rc-1 to official v0.30.0 release.**

    hack/release.sh --promote-release-candidate v0.31.0-rc-1 --dry-run=false

**Example: Creating a patch release v0.31.1. The branch will automatically be detected.**

    hack/release.sh --new-release v0.31.1 --dry-run=false

## Creating New Minor Releases
The release process is mostly automatic and consists of the following steps:

1. Create the branch and initial RC.

   `hack/release.sh --new-branch $RELEASE_BRANCH --new-tag ${TAG}.rc.0`

2. Continue to create RC's according to the release schedule

3. Once it is time for an official release, promote the most recent RC if no blockers exist.

   `hack/release.sh --promote-release-candidate ${TAG}.rc.5`

4. Wait for [Prow job](https://prow.ci.kubevirt.io/?repo=kubevirt%2Fkubevirt) to
   finish, and check that the binary artifacts got attached to the release
   at `https://github.com/kubevirt/kubevirt/releases/tag/${TAG}`
   and that the containers were correctly tagged and pushed to
   <https://quay.io/organization/kubevirt/>

5. If release looks correct, click "edit" on the release in the github UI
   and uncheck the "This is a pre-release" box. This will make the release
   official

6. Send a friendly announcement email to <kubevirt-dev@googlegroups.com> using
   the release notes already present on the release's description in github.

## Creating New Patch Releases

Releases on the stable branch only increment the patch level.
The release itself is only a git signed tag as it's used for minor releases as well.

1. Create the patch release. Note that the branch is automatically detected.

   `hack/release.sh --new-tag ${TAG}`

2. Wait for [Prow job](https://prow.ci.kubevirt.io/?repo=kubevirt%2Fkubevirt) to
   finish, and check that the binary artifacts got attached to the release
   at `https://github.com/kubevirt/kubevirt/releases/tag/$TAG`
   and that the containers were correctly tagged and pushed to
   <https://quay.io/organization/kubevirt/>

3. If release looks correct, click "edit" on the release in the github UI
   and uncheck the "This is a pre-release" box. This will make the release
   official

4. Send a friendly announcement email to <kubevirt-dev@googlegroups.com> using
   the release notes already present on the release's description in github.

# Merging to Release Branches

For every release, a branch will be created following the pattern `release-x.y`.
For now, community members can propose pull requests to be included into a
stable branch.
Those pull requests should be limited to bug fixes and must not be
enhancements. More info related to the policy around backporting can be found
in this document, [docs/release-branch-backporting.md](https://github.com/kubevirt/kubevirt/blob/main/docs/release-branch-backporting.md)

Cherry picking can be used to pick a merge commit from the main branch
to a stable branch. An example:

```bash
git checkout release-0.6
git cherry-pick $THE_MERGE_COMMIT_ID -m 1 -sx
[release-0.6 acd756040] Merge pull request #1234 from great_person
 Author: Bob Builder <builder@bob.com>
 Date: Thu Jun 28 17:50:05 2018 +0300
 5 files changed, 55 insertions(+), 22 deletions(-)
git push $YOUR_REMOTE release-0.6:release-0.6-aParticularFix
```

After pushing the branch, you'll need to make sure to create a pull request
against the correct target branch in GitHub (in this case the target branch
is `release-0.6`).

# Understanding Release Notes

Release notes are automatically generated by our release tool. The notes are
sourced from the delta of PRs merged since the last official release. The text
from those PRs are sourced directly from the `release-notes` section in
each PRs description.

# Using GitHub labels for Release Notes

We use GitHub labels to filter release notes, first using a 'kind/' label and
second using a 'sig/' label where applicable.
In some cases, 'kubevirt-bot' applies one or more of these labels automatically,
however they might need to be manually applied by either the author or a reviewer of the PR.

Add a label by including the ```/kind``` and ```/sig``` commands in the ```release-note-labels``` part of the PR description template. For example: ```/kind enhancement /sig security /sig compute```

If a PR matches more than one 'kind/' or 'sig/' label, add all that apply. The release note will be included in whichever takes the greater priority.

As per the PR template, if the PR requires additional action from users switching to the new release, include the string "action required" in the release note. The PR will be flagged accordingly.

## The 'kind/' Labels Used for Release Notes

* ```kind/enhancement```: The PR adds (or mostly adds if it is one of several related PRs) a new feature or enhancement to KubeVirt.
* ```kind/api-change```: The PR adds, removes, or otherwise changes an API.
* ```kind/bug```: The PR fixes a bug.
* ```kind/deprecation```: The PR deprecates a feature that will be removed in a subsequent release.

## The 'sig/' Labels Used for Release Notes

* ```sig/security```: The PR is relevant to SIG Security.
* ```sig/compute```: The PR is relevant to SIG Compute.
* ```sig/network```:  The PR is relevant to SIG Networking.
* ```sig/storage```: The PR is relevant to SIG Storage.
* ```sig/code-quality```: The PR is relevant to SIG Code Quality.

## Viewing Release Notes

Release notes are included in the git tag comment for the release, and copied to the [changelog of the KubeVirt website](https://kubevirt.io/tag/changelog).

Below is an example of getting the release notes for v0.31.0-rc.0

    git show v0.31.0-rc.0

# Handling Release Blockers

Release blockers can be set on issues and PRs by [approvers](https://github.com/kubevirt/kubevirt/blob/main/OWNERS_ALIASES) of the project. A PR or
issue can be flagged as a blocker through the use of the `/release-blocker <branch>`
in a github comment.

The KubeVirt release tool scans for blocker Issues/PRs and will not allow certain
actions to take place until the blockers are resolved. A resolved blocker is
when an Issue/PR with a blocker label is closed. **Do not remove the blocker label
for closed issues!**

## Release Blocker Criteria

A release blocker is a critical bug, regression, or backwards incompatible change
that must be addressed before the next official release is made. Only KubeVirt
[approvers](https://github.com/kubevirt/kubevirt/blob/main/OWNERS_ALIASES) can set this label on a PR or Issue.

## Setting a Release Blocker

Once a release blocker is set, the label must never be removed unless we have
decided the issue or PR does not in fact need to block a release. This means
that the release blocker labels should remain even after an issue or PR is closed.

**Example: Signalling a PR/Issue should block the next release branch.** This
Will prevent a new release branch from being cut until PR/Issue is closed.

    /release-blocker main

**Example: Signalling a PR/Issue should block the official release of a
stable branch** This will prevent any existing RCs from being promoted
to an official release. A new RC will only able to be created once this
Issue/PR is closed.

    /release-blocker release-0.31

**Example: Canceling a release-blocker.** This will remove the signal that
an Issue/PR is a blocker. This should only be done if the issue truly
isn't a blocker.

    /release-blocker cancel release-0.31

and canceling a blocker on main would look like.

    /release-blocker cancel main
