**TL;DR**- If you simply need to learn how to make a release, read [this](#creating-releases) section.

<!--ts-->
   * [Overview](#overview)
   * [Cadence and Timeline](#cadence-and-timeline)
   * [Versioning](#versioning)
   * [Announcement](#announcement)
   * [Handling Release Blockers](#handling-release-blockers)
      * [Release Blocker Criteria](#release-blocker-criteria)
      * [Setting a Release Blocker](#setting-a-release-blocker)
   * [Creating Releases](#creating-releases)
      * [Release Tool Credentials](#release-tool-credentials)
      * [Release Tool Usage Examples](#release-tool-usage-examples)
      * [Minor Releases](#creating-new-minor-releases)
      * [Patch Releases](#creating-new-patch-releases)
   * [Merging to Release Branches](#merging-to-release-branches)
   * [Understanding Release Notes](#understanding-release-notes)
<!--te-->


# Overview
- Release process is automated by [hack/release.sh](https://github.com/kubevirt/kubevirt/blob/main/hack/release.sh) script
- KubeVirt uses [semantic versioning](http://semver.org)
- Primary artifact is the source tree in form of signed git tag
- Binary artifacts are built using automation
- The releases appear on a time based release schedule
- Releases can be blocked using the ```release-blocker``` github comment

The primary artifact of a release is the source tree itself. The trust on the
tree is established by using a signed git tag.

For convenience a number of binary artifacts can be provided alongside a
release, those are:

- Container images (currently docker images), tagged and pushed to a registry
- Client side binaries (i.e. _virtctl_), published on the release page

These artifacts are provided in their respective channels and their natural way
of distribution.

# Cadence and Timeline
Both a new release branch and initial release candidate are cut on the **first
business day of each month.**

If no blocker issues are discovered the release candidate is **promoted to a full
release after 5 business days.**

If blockers are detected, a new release candidate is generated and will be
promoted after giving the impacted parties enough time to validate the blocker is
addressed.

**Timeline Example: No blockers detected, release is cut from the first rc**
```
July 1st  - release-v0.31 branch and v0.31.0-rc.0 tag are cut
July 8th  - promotion of v0.31.0-rc.0 to be the official v0.31.0 release
```

**Example: blocker is detected for release branch**
```
July 1st  - release-v0.31 branch and v0.31.0-rc.0 tag are cut
July 2nd  - blocker is detected and marked using ‘/release-block release-31` in a comment on an issue.
July 6th  - blocker issue is closed, new rc v0.31.0-rc.1 is allowed to be created
July 13th - promotion of v0.31.0-rc.1 to be the official v0.31.0 release
```

# Versioning
**Branches are created for every minor release and take the form of** `release-<major>.<minor>`
For example, the release branch for an upcoming v0.30.0 release will be 
`release-0.30`

**Releases are cut from release branches must adhere to** [semantic versioning conventions](http://semver.org).
For example, the initial release candidate for branch `release-0.30` is called
`v0.30.1-rc.0`

The determined version is then prefixed with a `v` (mostly for consistency,
because we started this way) and used as the tag name (`$TAG` below).

**RC Version Examples:**
```
v0.31.1-rc.0
v0.31.1-rc.1
```

**Official Release Version Examples**
```
v0.31.1
v0.31.2
```

# Announcement
Every official release must be announced on the `kubevirt-dev` mailinglist
<kubevirt-dev@googlegroups.com> with a body containing the release notes.

You can retrieve the auto generated release notes from the git tag's commit message.

Below is an example of getting the release notes for v0.31.0-rc.0

```
git show v0.31.0-rc.0
```

# Handling Release Blockers

Release blockers can be set on issues and PRs by [approvers](https://github.com/kubevirt/kubevirt/blob/main/OWNERS_ALIASES) of the project. A PR or
issue can be flaged as a blocker through the use of the `/release-blocker <branch>`
in a github comment.

The KubeVirt release tool scans for blocker Issues/PRs and will not allow certain
actions to take place until the blockers are resolved. A resolved blocker is
when an Issue/PR with a blocker label is closed. **Do not remove the blocker label
for closed issues!**

## Release Blocker Criteria

A release blocker is a critial bug, regression, or backwards incompatible change
that must be addressed before the next official release is made. Only KubeVirt
[approvers](https://github.com/kubevirt/kubevirt/blob/main/OWNERS_ALIASES) can set this label on a PR or Issue.

## Setting a Release Blocker

Once a release blocker is set, the label must never be removed unless we have
decided the issue or PR does not in fact need to block a release. This means
that the release blocker labels should remain even after an issue or PR is closed.

**Example: Signalling a PR/Issue should block the next release branch.** This
Will prevent a new release branch from being cut until PR/Issue is closed.

```/release-blocker main```

**Example: Signalling a PR/Issue should block the official release of a
stable branch** This will prevent any existing RCs from being promoted
to an official release. A new RC will only able to be created once this
Issue/PR is closed.

```/release-blocker release-0.31```

**Example: Canceling a release-blocker.** This will remove the signal that
an Issue/PR is a blocker. This should only be done if the issue truely
isn't a blocker.

```/release-blocker cancel release-0.31```

and canceling a blocker on main would look like.

```/release-blocker cancel main```

# Creating Releases

The actual releases are all cut using the kubevirt release-tool. This tool
automates the entire process of creating branches, signing tags, generating
prow configs, and more. All you need to do is gather a few credentials
in order to use the tool.

## Release Tool Credentials

You must have 2 items before you can create a release.

1. **GPG key and passphrase files used for creating signed releases.**

[Instructions for adding GPG key to your github account](https://help.github.com/articles/adding-a-new-gpg-key-to-your-github-account)

After adding the GPG key to github, export both the key and passphrase to files.
Be aware that this results in the key and passphrase being placed into a plain
text file on your machine. Make sure you don't place this in shared storage.

**Example of exporting key to file**

```gpg --export-secret-key -a <your-email-address> > ${HOME}/gpg-private```

**Example of putting passphrase in file**

```echo "<insert your passphrase here>" > ${HOME}/gpg-passphrase```

2. **Github API token file used for accessing the github api**

When you create this token the only permission you need to give it is the
ability to access github repositories. That's it.

[Instructions for creating access token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token)

**Place your token in a text file such as** ```${HOME}/github-api-token```

## Release Tool Usage Examples

Once you have your credentials in files, the kubevirt release tool handles
all the rest. All you need to do is provide your credentials and tell the tool
what release you want to make.

Place the paths to your credential files in the following environment variables.

```
export GPG_PRIVATE_KEY_FILE="${HOME}/gpg-private"
export GPG_PASSPHRASE_FILE="${HOME}/gpg-passphrase"
export GITHUB_API_TOKEN_FILE="${HOME}/github-api-token"
```

Now you can use the release tool to do whatever you'd like. Note that you can
use the ```--dry-run=true``` argument to test a change before executing it.

**Example: creating a new release branch with the initial release candidate v0.31.0-rc.0**
```
hack/release.sh --new-branch release-0.31 --new-release v0.31.0-rc.0 --dry-run=false
```

**Example: Creating a new rc v0.31.0-rc.0**
```
hack/release.sh --new-release v0.31.0-rc.0 --dry-run=false
```

**Example: Promoting a release candidate v0.31.0-rc-1 to official v0.30.0 release.**
```
hack/release.sh --promote-release-candidate v0.31.0-rc-1 --dry-run=false
```

**Example: Creating a patch release v0.31.1. The branch will automatically be detected.**
```
hack/release.sh --new-release v0.31.1 --dry-run=false
```

## Creating New Minor Releases
The release process is mostly automatic and consists of the following steps:

1. Create the branch and initial RC.

   ```hack/release.sh --new-branch $RELEASE_BRANCH --new-tag ${TAG}.rc.0```

2. Wait 5 business days

3. Promote RC to official release if no blockers exist.

   ```hack/release.sh --promote-release-candidate ${TAG}.rc.0```

4. Wait for [travis](https://travis-ci.org/kubevirt/kubevirt/) to finish, and
   check that the binary artifacts got attached to the release at
   `https://github.com/kubevirt/kubevirt/releases/tag/${TAG}`
   and that the containers were correctly tagged and pushed to
   <https://hub.docker.com/r/kubevirt/>

5. If release looks correct, click "edit" on the release in the github UI
   and uncheck the "This is a pre-release" box. This will make the release
   official

6. Sent a friendly announcement email to <kubevirt-dev@googlegroups.com> using
   the release notes already present on the release's description in github.

## Creating New Patch Releases

Releases on the stable branch only increment the patch level.
The release itself is only a git signed tag as it's used for minor releases as well.

1. Create the patch release. Note that the branch is automatically detected.

   ```hack/release.sh --new-tag ${TAG}```

2. Wait for [travis](https://travis-ci.org/kubevirt/kubevirt/) to finish, and
   check that the binary artifacts got attached to the release at
   `https://github.com/kubevirt/kubevirt/releases/tag/$TAG`
   and that the containers were correctly tagged and pushed to
   <https://hub.docker.com/r/kubevirt/>

3. If release looks correct, click "edit" on the release in the github UI
   and uncheck the "This is a pre-release" box. This will make the release
   official

4. Sent a friendly announcement email to <kubevirt-dev@googlegroups.com> using
   the release notes already present on the release's description in github.

# Merging to Release Branches

For every release a branch will be created following the pattern `release-x.y`.
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
from those PRs are sourced directly from the ```release-notes``` section in
each PRs description.

Below is an example of getting the release notes for v0.31.0-rc.0

```
git show v0.31.0-rc.0
```

