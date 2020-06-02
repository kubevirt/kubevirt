# Continuous integration

Kubevirt uses prow as its CI
See [Project-infra](https://github.com/kubevirt/project-infra)

## CI developer mode

In order to save resources and run only selected lanes (i.e. for debugging),
a developer can use KUBEVIRT_LANE_FOCUS environment variable (located at automation/test.sh).
The variable can be set to a list of selected lanes TARGET names, separated by space.
For example:
`export KUBEVIRT_LANE_FOCUS="kind-k8s-1.17.0-ipv6"`
Will run only kind IPv6 lane.
See [prow presubmit](https://github.com/kubevirt/project-infra/blob/master/github/ci/prow/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml)
in order to select the needed TARGET name accoring the lane name.
The other lanes will fail immediately, thus saving CI resources and allow developers to get the results faster as well.
In addition, the PR won't be mergeable until this selection will be reverted (unset the KUBEVIRT_LANE_FOCUS),
and all the lanes will run and succeed.

## Using make targets to focus lanes

Focus on one target
`make focus lanes=target`
or
`make focus lanes="target"`

Clear focus
`make focus`

Focus on multi targets
`make focus lanes="target1 target2"`

