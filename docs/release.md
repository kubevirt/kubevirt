
The goal of this document is to define the release process of KubeVirt, but it
is not intended to define any release criteria.


Overview
--------
- KubeVirt uses [semantic versioning](http://semver.org)
- Primary artifact is the source tree in form of a signed tap using
  [git evtag](https://github.com/cgwalters/git-evtag)
- Binary artifacts are built using automation
- The releases appear on a time based release schedule


Content
-------
The primary artifact of a release is the source tree itself. The trust on the
tree is established by using _git-evtag_ which can be used to sign the tree
recursively, including blobs and submodules.

For convenience a number of binary artifacts can be provided alongside a
release, those are:

- Container images (currently docker images), tagged and pushed to a registry
- Client side binaries (i.e. _virtctl_), published on the release page

These artifacts are provided in their respective channels and their natural way
of distribution.


Cadence
-------
A release will take place on the first Monday of each month.
The release owner is in charge of delaying a release if it is _really_
necessary.
If a release has to be delayed by more than a week, then the release must be
skipped.


Versioning
----------
The release owner is in charge of choosing the correct release version -
according to the [semantic versioning conventions](http://semver.org).

The determined version is then prefixed with a `v` (mostly for consistency,
because we started this way) and used as the tag name (`$TAG` below).

Examples:

```
v0.0.1-alpha.0
v0.0.1-alpha.1
```

Release notes
-------------
Every release must should be accompanied by release notes describing the
major highlights of the release.
The release notes must be provided in the commit message of the tag.


Announcement
------------
Every release must be announced on the `kubevirt-dev` mailinglist
<kubevirt-dev@googlegroups.com>.

The `hack/release-announce.sh` script should be used to generate the
announce email.
Using this script ensures consistency and documentation of the release
process and content between releases.


Topic: Signing
--------------
The workflows below use signed tags. The following link describes what you
need to do in order to setup correct signing for tags (and commits) on your
local machine and in GitHub, in order to get the commits verified.

https://help.github.com/articles/adding-a-new-gpg-key-to-your-github-account/

Setting up and using signing is mandatory for any release.


Releasing minor versions
------------------------
The release process is mostly automatic and consists of the following steps:

1. Run the `hack/release-announce.sh` script which runs functional tests,
   generates some stats and outputs the announce text.
2. Tag the commit using `git evtag sign $TAG` (which is a signed and annotated
   tag)
   1. Provide the announce text as part of the tag commit message
3. Push the tag to github `git push git@github.com:kubevirt/kubevirt.git $TAG`
4. Wait for [travis](https://travis-ci.org/kubevirt/kubevirt/) to finish, and
   check that the binary artifacts got attached to the release at
   `https://github.com/kubevirt/kubevirt/releases/tag/$TAG`
   and that the containers were correctly tagged and pushed to
   <https://hub.docker.com/r/kubevirt/>
5. Adjust the release details (draft, pre-release) as necessary at
   `https://github.com/kubevirt/kubevirt/releases/tag/$TAG`
6. Sent a friendly announcement email to <kubevirt-dev@googlegroups.com>

Stable Branches
---------------

> **Note:** Before a bug is fixed in a stable branch, the same bug must be fixed
> in the `master` branch. The only exception is when a bug exists in a stable
> branch only.

For every release a branch will be created following the pattern `release-x.y`.
For now, community members can propose pull requests to be included into a
stable branch.
Those pull requests should be limited to bug fixes and must not be
enhancements.

Cherry picking can be used to pick a merge commit from the master branch
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

Releasing patch versions
------------------------

Releases on the stable branch only increment the patch level.
The release itself is only a evsigned tag as it's used for minor releases as well.

1. Tag the commit using `git evtag sign $TAG` (which is a signed and annotated
   tag)
   1. Use the tag name as the commit message
3. Push the tag to github `git push git@github.com:kubevirt/kubevirt.git $TAG`
4. Wait for [travis](https://travis-ci.org/kubevirt/kubevirt/) to finish, and
   check that the binary artifacts got attached to the release at
   `https://github.com/kubevirt/kubevirt/releases/tag/$TAG`
   and that the containers were correctly tagged and pushed to
   <https://hub.docker.com/r/kubevirt/>
5. Set the tag name as the release on the release details page and ensure Draft is unset
   `https://github.com/kubevirt/kubevirt/releases/tag/$TAG`
6. Sent a friendly announcement email to <kubevirt-dev@googlegroups.com>

