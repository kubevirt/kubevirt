package cgroup

const (
	// Cgroup paths
	procMountPoint     = "/proc"
	HostRootPath       = procMountPoint + "/1/root"
	cgroupBasePath     = "/sys/fs/" + cgroupStr
	HostCgroupBasePath = HostRootPath + cgroupBasePath
)

const (
	// Cgroup files
	v1ThreadsFilename = "tasks"
	v2ThreadsFilename = "cgroup.threads"
	procsFilename     = "cgroup.procs"
)

const (
	// Cgroup subsystems
	CgroupSubsystemCpu       string = "cpu"
	CgroupSubsystemCpuacct   string = "cpuacct"
	CgroupSubsystemCpuset    string = "cpuset"
	CgroupSubsystemMemory    string = "memory"
	CgroupSubsystemDevices   string = "devices"
	CgroupSubsystemFreezer   string = "freezer"
	CgroupSubsystemNetCls    string = "net_cls"
	CgroupSubsystemBlkio     string = "blkio"
	CgroupSubsystemIo        string = "io"
	CgroupSubsystemPerfEvent string = "perf_event"
	CgroupSubsystemNetPrio   string = "net_prio"
	CgroupSubsystemHugetlb   string = "hugetlb"
	CgroupSubsystemPids      string = "pids"
	CgroupSubsystemRdma      string = "rdma"
)

const (
	// others consts
	V1 CgroupVersion = "v1"
	V2 CgroupVersion = "v2"

	loggingVerbosity     = 2
	detailedLogVerbosity = 4

	cgroupStr = "cgroup"

	cgroupAlreadyExistsErrFmt = "creating child cgroup: child cgroup in path %s already exists"
)
