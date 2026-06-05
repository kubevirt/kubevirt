# KubeVirtCI K8S providers update automation

There exist automated steps for creating, updating and integrating k8s providers. These are all described as prow jobs in [project-infra](https://github.com/kubevirt/project-infra/).

| Trigger                                   | Job                                                                                                               | Result                    |
| ----------- |  ----------- | ----------- |
| release of a new kubernetes minor version | [`periodic-kubevirtci-cluster-minorversion-updater`](https://github.com/kubevirt/project-infra/search?q=periodic-kubevirtci-cluster-minorversion-updater)     | Creates a new provider for that release |
| release of a new kubernetes minor version | [`periodic-kubevirtci-provider-presubmit-creator`](https://github.com/kubevirt/project-infra/search?q=periodic-kubevirtci-provider-presubmit-creator)                                                         | Creates a PR with a new check-provision job to enable testing of the new provider |
| release of a new kubernetes minor version | [`periodic-kubevirt-job-copier`](https://github.com/kubevirt/project-infra/search?q=periodic-kubevirt-job-copier)                                                         | Creates a PR with a new set of kubevirt sig jobs to enable testing of kubevirt with the new provider |
| release of new kubernetes patch version   | [`periodic-kubevirtci-cluster-patchversion-updater`](https://github.com/kubevirt/project-infra/search?q=periodic-kubevirtci-cluster-patchversion-updater)     | Creates a PR that updates the patch version for each KubeVirtCI k8s provider |  
| merge to kubevirt/kubevirtci main branch  | [`periodic-kubevirtci-bump-kubevirt`](https://github.com/kubevirt/project-infra/search?q=periodic-kubevirtci-bump-kubevirt)                   | Creates a PR to update KubeVirtCI in kubevirt/kubevirt |
| at the start of each month                              | [`periodic-kubevirt-presubmit-requirer`](https://github.com/kubevirt/project-infra/search?q=periodic-kubevirt-presubmit-requirer)                   | Checks always_run and optional states of latest kubevirt sig test jobs  |
