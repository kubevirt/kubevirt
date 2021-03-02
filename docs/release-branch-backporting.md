# Release Branch Backporting Policy

Bug fixes are eligible to be backported from the master branch to any previous
release branch. The following criteria must be met before a backport can be
considered. It is the reviewer's and approver's responsibility to uphold this
policy.

- **Bug Fix Only:** The backport must be a bug fix and the bug fix must be
first merged into the master branch. The only exception is when a bug only
exists in a stable branch and does not exist in the master branch.

- **Release Note** The PR description's release-note section must indicate in
a brief one line statement what the backport addresses. This note gets
automatically put into our release notes when a new release is cut from the
stable branch.

- **Detailed Description** The backport pull request's description must either
contain a detailed explanation for what the bug fix addresses or contain a
link to such a explanation present in another PR/Issue.

- **CI Lanes Must Pass** The release branch being backported to must still be
able to execute the unit and functional test lanes in order to validate the bug
fix on that branch. This only impacts very old branches that are no longer
compatible with our CI system.




