# Version and Release

### Overview

### Version Scheme

CDI adheres to the [semantic version definitions](https://semver.org/) format of vMAJOR.MINOR.PATCH.  These are defined as follows:

- Major - Non-backwards compatible, API contract changes.  Incrementing a Major version means the consumer will have to make changes to the way they interact with the CDI API.  Failing do to so will result in unexpected behavior.  When these changes occur, the Major version will be incremented at the end of the sprint instead of the Minor Version.

- Minor - End of Sprint release. Encapsulates non-API-breaking changes within the current Major version.  The current Sprint cycle is 2 weeks long, producing in bug fixes and feature additions.  Publishing a Minor version at the end of the cycle allows consumers to immediately access the end product of that Sprint's goals. Issues or bugs can be reported and addressed in the following Sprint.  It is expected that this patch contain myriad commits.

- Patch - mid-Sprint release for fixing blocker bugs. In the case that a bug is blocking CDI consumers' workflow, a fix may be released as soon as it is merged.  A Patch should be limited expressly to the bug fix and not include anything unrelated.

### Releasing a New Version

 The version number is tracked in several files in CDI as well as through a git tag.  To reduce the chance of human error, a help script is used to change the version in all known locations.

     DO NOT EDIT ANY VERSION STRINGS IN CDI!!

1. Set the new release version

    A recipe has been provided in `Makefile` to handle version setting. Use ONLY this command to set versions. Do NOT edit the values manually.

        $ make  set-version VERSION=v#.#.#

    The `set-version` recipe will locate files in CDI containing the current version value, substitute in the new version, then commit and tag the changes.  The user will be shown a list of files to be changed and prompted to continue before the substitutions are made.

1. Verify the changes

    Before publishing the changes, make one last check to verify the correct version value has been substituted in.

        $ git diff HEAD~1

1. Push the changes to Github

        $ git push upstream master && git push upstream --tags

   Travis CI will detect the new tag and execute the deploy script.  This will publish the newly updated controller manifest and CDI binaries to git releases.


#### Verifying the Release

##### Images

-  Check hub.docker.com/r/kubevirt repository for the newly tagged images. If you do not see the tags corresponding to the version, something has gone wrong.

   [CDI-Controller](https://hub.docker.com/r/kubevirt/cdi-controller/tags/)

   [CDI-Importer](https://hub.docker.com/r/kubevirt/cdi-importer/)

##### Travis CI Jobs

Track the CI job for the pushed tag.  Navigate to the [CDI Travis dashboard](https://travis-ci.org/kubevirt/containerized-data-importer/branches) and select the left most colored box (either Green, Yellow, or Red) for the branch corresponding to the version 
