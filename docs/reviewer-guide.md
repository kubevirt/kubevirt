# Reviewer guide

Some tips and tricks to obtain better reviews.

Make a few passes over the code you want to review.

1st pass, make sure the general design makes sense and that the code is structured in a way that's consistent with the project.

2nd pass, detailed look. depending on the complexity of the PR, sometimes it easier check out the code and step through it in your editor.

3rd pass, verify diffs from previous feedback rounds.

4th pass, through the whole PR once you think that it is ready to be merged, try to put yourself in the position of someone who needs that feature desperately and play with the flow several times

## Good to check:

* User input validation
* Reasonable error messages
* Reasonable info messages
* Unit and functional tests to avoid regressions (the core case must be tested)
* The PR implement what is needed (changes out of the PR scope should belong to a separate one)

## Pull Request structure

* It's preferred that authors rebase on master instead of merging master into their PRs.
* We merge PRs into our branches.
* Commits in a PR should make sense: Ask people to squash commits like "Fix reviewer comments", "wip", "addition", ...
