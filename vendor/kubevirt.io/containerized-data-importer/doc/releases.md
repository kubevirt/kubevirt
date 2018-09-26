# Version and Release

* [Overview](#overview)
* [Version Scheme](#version-scheme)
* [Releasing a New Version](#releasing-a-new-version)
    * [Verifying the Release](#verifying-the-release)
    * [Travis CI](#travis-ci)
### Overview

### Version Scheme

CDI adheres to the [semantic version definitions](https://semver.org/) format of vMAJOR.MINOR.PATCH.  These are defined as follows:

- Major - Non-backwards compatible, API contract changes.  Incrementing a Major version means the consumer will have to make changes to the way they interact with the CDI API.  Failing do to so will result in unexpected behavior.  When these changes occur, the Major version will be incremented at the end of the sprint instead of the Minor Version.

- Minor - End of Sprint release. Encapsulates non-API-breaking changes within the current Major version.  The current Sprint cycle is 2 weeks long, producing in bug fixes and feature additions.  Publishing a Minor version at the end of the cycle allows consumers to immediately access the end product of that Sprint's goals. Issues or bugs can be reported and addressed in the following Sprint.  It is expected that this patch contain myriad commits.

- Patch - mid-Sprint release for fixing blocker bugs. In the case that a bug is blocking CDI consumers' workflow, a fix may be released as soon as it is merged.  A Patch should be limited expressly to the bug fix and not include anything unrelated.

### Releasing a New Version

Release branches are used to isolate a stable version of CDI.  Git tags are used within these release branches to track incrementing of Minor and Patch versions.  When a Major version is incremented, a new stable branch should be created corresponding to the release.

- Release branches should adhere to the `release-v#.#.#` pattern.

- Tags should adhere to the `v#.#.#(-alpha.#)` pattern.

When creating a new release branch, follow the below process.  This assumes that `origin` references a fork of `kubevirt/containerized-data-importer` and you have added the main repository as the remote alias `<upstream>`.  If you have cloned `kubevirt/containerized-data-importer` directly, omit the `<upstream>` alias.

1. Make sure you have the latest upstream code

    `$ git fetch <upstream>`

1. Checkout the release branch locally

    `$ git checkout release-v#.#`

    e.g. `$ git checkout release-v1.1`

1. Create an annotated tag corresponding to the version

    `$ git tag -a -m "v#.#.#" v#.#.#`

1. Push the new branch and tag to the main kubevirt repo.  (If you have cloned the main repo directly, use `origin` for <`upstream`>)

    `$ git push v#.#.#`

CI will be triggered when a tag matching `v#.#.#(-alpha.#)` is pushed.  The automation will handle release artifact testing, building, and publishing.

Following the release, `make release-description` should be executed to generate a github release description template.  The `Notable Changes` section should be filled in manually, briefly listing major changes that the new release includes.  Copy/Paste this template into the corresponding github release.

#### Verifying the Release

##### Images

-  Check hub.docker.com/r/kubevirt repository for the newly tagged images. If you do not see the tags corresponding to the version, check the travis build log for errors.

   [CDI-Controller](https://hub.docker.com/r/kubevirt/cdi-controller/tags/)

   [CDI-Importer](https://hub.docker.com/r/kubevirt/cdi-importer/)

   [CDI-Cloner](https://hub.docker.com/r/kubevirt/cdi-cloner/)

##### Travis CI

Track the CI job for the pushed tag.  Navigate to the [CDI Travis dashboard](https://travis-ci.org/kubevirt/containerized-data-importer/branches) and select the left most colored box (either Green, Yellow, or Red) for the branch corresponding to the version 
