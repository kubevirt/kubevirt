package cgroup

const (
	ProcMountPoint   = "/proc"
	CgroupMountPoint = "/sys/fs/cgroup"
)

const (
	HostRootPath       = "/proc/1/root" // ihol3
	cgroupBasePath     = "/sys/fs/cgroup"
	HostCgroupBasePath = HostRootPath + cgroupBasePath
)
