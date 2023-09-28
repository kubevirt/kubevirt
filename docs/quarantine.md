# Test Quarantine

The execution of tests on CI should be as deterministic as possible, besides
potential issues coming from infrastructure or external dependencies. When a
test sometimes fails and sometimes passes, apparently randomly, without any
relationship with the changes in a PR, it has very negative implications:
* The results of the whole suite are less valuable, we can no longer trust
the information given by the suite execution. Changes are added to the codebase
without actually knowing if they are going to break something.
* As a consequence, we can't be completely sure about the state of the product
at release time, it can be ok or not.
* The development process is affected negatively. Developers need to wait
until a lucky execution aligns all the required tests in green, which can take
more than a week in some cases.

In summary, the CI system stops helping the product evolution and instead becomes
an obstacle.

We have introduced a quarantine methodology to put apart the tests that don't
behave deterministically until they are fixed, so that we can keep the rest of
the suite as healthy as possible. You can read more about test quarantine
methodology in [1] and [2], and about an actual implementation in [3].

## Goal

The purpose of applying the methodology described in this document is increasing
the stability of the CI suite. We consider the CI stable if the whole test suite
has a failure rate below 10%.

We need to multiplicate the individual failure rates to obtain the whole suite
passing rate, this means that with a failure rate of 5% per individual test, more
than 2 flaky tests would  lead to an overall test suite failure rate above the 10%
goal (0.95 ** 2 = 0.9025). And, for instance, 17 failing tests with 5% failure rate
would lead to a terrible 41.81% passing rate of the whole suite (0.95 ** 17 = 0.4181).

## Procedure

In order to remove as much as possible the influence of changes in PRs to
determine the stability of the suite, we will take into account only results
from the periodics that run e2e tests from main (hese jobs can be checked
[on testgrid]) and presubmits that are executed on merged code (on tide merge
batches as reported by flakefinder).

We will consider test failures only in jobs where less than 5 tests failed, so
that we don't take into account systemic failures caused for instance by an
infrastructure problem.

### Putting tests in quarantine

A test must be put in quarantine when any of these conditions is met:
* It has a failure rate higher than 5% in the last two weeks.
* It has a failure rate higher than 20% in the last 3 days.

#### Quarantine PR

A PR will be proposed on Mondays every two weeks with a batch of the tests that
met the first condition. A PR can be proposed at any time for the tests that meet
the second condition. In both cases the PR will add the text `[QUARANTINE]` and
the `decorators.Quarantine` [labelDecorator](https://github.com/kubevirt/kubevirt/blob/9a3799f7a0b97b70033e119c0b401778c51dee14/tests/decorators/decorators.go#L5)
to each test's description in the code.
An email will be sent to the owners of the suspected tests.

After the PR with the quarantine candidates is proposed there is a grace period
of 2 days to prepare and land a fix for a test in the batch. If at least 5
consecutive executions with the fix pass the test can be removed from the batch.

#### Quarantined test owners

Each quarantined test must have a team owner. The PR will add the text
`[sig-{compute,network,storage,operator}]` to each test's description and
the proper label decorator.

#### Quarantining release blockers

When a test marked with the [release-blocker] meets the conditions to be
quarantined we will:
* Create github issue with a comment `/release-blocker main` to ensure that
the issue is addressed before a new release is cut.
* Ensure that the github issue is assigned to an individual who will own bringing
the blocker to completion within a quick time frame.

### Getting tests out of quarantine

A member of the team assigned to each quarantined tests should propose a fix for
the test or, after investigating the source of the errors, determine that the
test itself doesn't need changes to be fixed (maybe the fix needs to be done on
other parts of the code base or in a separate repo). In any case, the team
assigned must communicate when the test is expected to be stable.

A test must be put out of quarantine when:
* It hasn't failed on any of the periodic lanes in the two weeks after the time
indicated by the team assigned to bring the test back to the stable suite.


After two weeks with successful executions has passed, a quarantined tests will
be ready to join the stable suite again. A member of the team assigned to each
quarantined tests will propose a PR to remove the text `[QUARANTINE]` and the
label decorator from the test description in the code.
After merging this PR the test will be out of quarantine.

[1]: https://martinfowler.com/articles/nonDeterminism.html#Quarantine
[2]: https://www.thoughtworks.com/en-us/insights/blog/no-more-flaky-tests-go-team
[3]: https://docs.gitlab.com/ee/development/testing_guide/flaky_tests.html#quarantined-tests
[on testgrid]: https://testgrid.k8s.io/kubevirt-periodics
