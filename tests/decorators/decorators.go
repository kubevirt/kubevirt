package decorators

import . "github.com/onsi/ginkgo/v2"

var (
	Quarantine  = Label("QUARANTINE")
	Periodic    = Label("PERIODIC")
	Conformance = Label("conformance")

	// SIGs
	SigCompute           = Label("sig-compute")
	SigOperator          = Label("sig-operator")
	SigNetwork           = Label("sig-network")
	SigStorage           = Label("sig-storage")
	SigComputeRealtime   = Label("sig-compute-realtime")
	SigComputeMigrations = Label("sig-compute-migrations")
	SigMonitoring        = Label("sig-monitoring")
	SigPerformance       = Label("sig-performance")

	// HW
	GPU         = Label("GPU")
	VGPU        = Label("VGPU")
	SEV         = Label("SEV")
	SRIOV       = Label("SRIOV")
	StorageReq  = Label("storage-req")
	Multus      = Label("Multus")
	Macvtap     = Label("Macvtap")
	Invtsc      = Label("Invtsc")
	KSMRequired = Label("KSM-required")

	// Features
	Sysprep                              = Label("Sysprep")
	Windows                              = Label("Windows")
	Networking                           = Label("Networking")
	VMIlifecycle                         = Label("VMIlifecycle")
	Expose                               = Label("Expose")
	NativeSsh                            = Label("native-ssh")
	ExcludeNativeSsh                     = Label("exclude-native-ssh")
	Reenlightenment                      = Label("Reenlightenment")
	TscFrequencies                       = Label("TscFrequencies")
	PasstGate                            = Label("PasstGate")
	VMX                                  = Label("VMX")
	Upgrade                              = Label("Upgrade")
	CustomSELinux                        = Label("CustomSELinux")
	Istio                                = Label("Istio")
	InPlaceHotplugNICs                   = Label("in-place-hotplug-NICs")
	MigrationBasedHotplugNICs            = Label("migration-based-hotplug-NICs")
	NetCustomBindingPlugins              = Label("netCustomBindingPlugins")
	RequiresTwoSchedulableNodes          = Label("requires-two-schedulable-nodes")
	VMLiveUpdateRolloutStrategy          = Label("VMLiveUpdateRolloutStrategy")
	USB                                  = Label("USB")
	AutoResourceLimitsGate               = Label("AutoResourceLimitsGate")
	RequiresTwoWorkerNodesWithCPUManager = Label("requires-two-worker-nodes-with-cpu-manager")
	RequiresDualStackCluster             = Label("requires-dual-stack-cluster")
	RequiresHugepages2Mi                 = Label("requireHugepages2Mi")

	// Storage classes
	// Requires a storage class with support for snapshots
	RequiresSnapshotStorageClass = Label("RequiresSnapshotStorageClass")
	// Requires a storage class without support for snapshots
	RequiresNoSnapshotStorageClass = Label("RequiresNoSnapshotStorageClass")
	// Requires a storage class with ReadWriteMany Block support
	RequiresRWXBlock = Label("RequiresRWXBlock")
	// Requires a storage class with Block storage support
	RequiresBlockStorage = Label("RequiresBlockStorage")
	// Kubernetes versions
	Kubernetes130 = Label("kubernetes130")
	// WG archs
	WgS390x = Label("wg-s390x")

	// NoFlakeChecker decorates tests that are not compatible with the check-tests-for-flakes test lane.
	// This should only be used for legitimate purposes, like on tests that have a flake-checker-friendly clone.
	NoFlakeCheck = Label("no-flake-check")
)
