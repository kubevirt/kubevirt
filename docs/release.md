Release process
===============

The goal of this document is to define the release process of KubeVirt, but it
is not intended to define any release criteria.


Overview
--------
- KubeVirt uses [semantic versioning](http://semver.org)
- Primary artefact is the source tree in form of a signed tap using
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

- Container images (currently docker images)
- Client sided binaries (i.e. _virtctl_)


Cadence
-------
A release is taking place on every first Monday of a month.
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


Releasing
---------
The release process is mostly automatic and consists of the following steps:

1. Tag a commit using `git evtag sign $TAG` (which is a signed and annotated
   tag)
   1. Provide the release notes as part of the tag commit message
2. Push the tag to github `git push origin $TAG`
3. Wait for [travis](https://travis-ci.org/kubevirt/kubevirt/) to finish, and
   check that the artifacts got attached to the release at
   <https://github.com/kubevirt/kubevirt/releases/tag/$TAG>
4. Adjust the release details (draft, pre-release) as necessary at
   <https://github.com/kubevirt/kubevirt/releases/tag/$TAG>
5. Sent a friendly announcement email to <kubevirt-dev@googlegroups.com>
