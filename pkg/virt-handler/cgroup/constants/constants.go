/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package constants

const (
	CgroupStr          = "cgroup"
	ProcMountPoint     = "/proc"
	hostRootPath       = ProcMountPoint + "/1/root"
	CgroupBasePath     = "/sys/fs/" + CgroupStr
	HostCgroupBasePath = hostRootPath + CgroupBasePath
)
