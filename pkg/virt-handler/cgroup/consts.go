package cgroup

const (
	// Cgroup paths
	procMountPoint     = "/proc"
	HostRootPath       = procMountPoint + "/1/root"
	BasePath           = "/sys/fs/" + cgroupStr
	HostCgroupBasePath = HostRootPath + BasePath
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
	// common error messages / formats
	vmiNotDedicatedErrFmt                    = "vmi %s is expected to be defined with dedicated CPUs"
	cgroupAlreadyExistsErrFmt                = "creating child cgroup: child cgroup in path %s already exists"
	handledDedicatedCpusSuccessfully         = "handled dedicated cpus successfully"
	attachedEmulatorTasksSuccessfully        = "attached emulator tasks successfully"
	castingToConcreteTypeFailedErrFmt        = "casting of cgroup manager to %s concrete manager failed - this shouldn't happen"
	initializedEmulatorContainerSuccessfully = "emulator container is successfully initialized for vmi %s/%s"
)

const (
	// others consts
	V1 CgroupVersion = "v1"
	V2 CgroupVersion = "v2"

	loggingVerbosity     = 2
	detailedLogVerbosity = 4

	cgroupStr = "cgroup"
)

const (
	// emulator container cgroup names
	EmulatorContainerCgroupAmbassador   = "ambassador"
	EmulatorContainerCgroupEmulator     = "emulator"
	EmulatorContainerCgroupVcpu         = "vcpu"
	EmulatorContainerCgroupHousekeeping = "housekeeping"
)
