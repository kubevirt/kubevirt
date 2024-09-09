---
name: Flake Report
about: Report a flaky test in KubeVirt
title: 'test_name'
labels: kind/flake
assignees: ''
---

# What happened

<!-- insert test name -->
Flaky test detected: {test_name} [1]

<!-- labels for flaky tests -->
/kind flake
/priority critical-urgent

<!-- sig assignment
     all tests contain a sig identifier, please assign the corresponding SIG to the issue, i.e.
     for a test name containing [sig-compute] or [sig-operator]
-->
/sig compute

<!-- note: the flakefinder url needs to be a stable one, i.e. instead of the moving latest report use any with a date instead -->
[1]: {flakefinder_url}

## Additional context
Add any other context about the problem here.

# Flake Action Plan

**As the assignee**, **thoroughly review the issue** and put the resulting report as comment into this issue.

## Then decide on one of the following actions:

### **The flake is a bug**

* Add label commenting `/triage accepted`
* **Create a PR to fix the bug**
* Reference this issue in the PR
* Keep this issue open until the testcase does not fail anymore

### **The flake is non-critical or an issue that is hard to fix**
* **Quarantine the test** by creating a pull request assigning [QUARANTINE] tag to test name and [Quarantine decorator](https://github.com/kubevirt/kubevirt/blob/f85a8117b8a90fd913ec0719faae9506866d1525/tests/decorators/decorators.go#L7) to the test
* Reference this issue on the PR

### There was an infrastructure issue

An infra issue is anything "below" the testcase

* Add label commenting `/triage infra-issue`
* Close the issue, adding a comment with details about the infrastructure issue.

After the flake has been fixed, document the learning from it inside the fix PR
