package decorators

import . "github.com/onsi/ginkgo/v2"

var (
	Quarantine  = Label("QUARANTINE")
	Periodic    = Label("PERIODIC")
	Conformance = Label("conformance")

	/* SIGs */

	SigCompute             = Label("sig-compute")
	SigOperator            = Label("sig-operator")
	SigNetwork             = Label("sig-network")
	SigStorage             = Label("sig-storage")
	SigComputeInstancetype = Label("sig-compute-instancetype")
	SigComputeRealtime     = Label("sig-compute-realtime")
	SigComputeMigrations   = Label("sig-compute-migrations")
	SigMonitoring          = Label("sig-monitoring")
	SigPerformance         = Label("sig-performance")

	/* HW */

	GPU             = Label("GPU")
	VGPU            = Label("VGPU")
	SEV             = Label("SEV")
	SEVES           = Label("SEVES")
	SEVSNP          = Label("SEVSNP")
	SecureExecution = Label("secure-execution")
	SRIOV           = Label("SRIOV")
	StorageReq      = Label("storage-req")
	Multus          = Label("Multus")
	Macvtap         = Label("Macvtap")
	Invtsc          = Label("Invtsc")
	KSMRequired     = Label("KSM-required")
	ACPI            = Label("ACPI")

	/* Deployment */

	SingleReplica = Label("single-replica")
	MultiReplica  = Label("multi-replica")

	/* Features */

	CPUModel                             = Label("cpumodel")
	VSOCK                                = Label("vsock")
	VirtioFS                             = Label("virtiofs")
	Sysprep                              = Label("Sysprep")
	Windows                              = Label("Windows")
	Networking                           = Label("Networking")
	VMIlifecycle                         = Label("VMIlifecycle")
	Expose                               = Label("Expose")
	Reenlightenment                      = Label("Reenlightenment")
	TscFrequencies                       = Label("TscFrequencies")
	HostDiskGate                         = Label("HostDiskGate")
	VMX                                  = Label("VMX")
	Upgrade                              = Label("Upgrade")
	Istio                                = Label("Istio")
	InPlaceHotplugNICs                   = Label("in-place-hotplug-NICs")
	MigrationBasedHotplugNICs            = Label("migration-based-hotplug-NICs")
	NetCustomBindingPlugins              = Label("netCustomBindingPlugins")
	RequiresTwoSchedulableNodes          = Label("requires-two-schedulable-nodes")
	RequiresThreeSchedulableNodes        = Label("requires-three-schedulable-nodes")
	RequiresDedicatedWorkerNodes         = Label("requires-dedicated-worker-nodes")
	VMLiveUpdateRolloutStrategy          = Label("VMLiveUpdateRolloutStrategy")
	USB                                  = Label("USB")
	RequiresTwoWorkerNodesWithCPUManager = Label("requires-two-worker-nodes-with-cpu-manager")
	RequiresNodeWithCPUManager           = Label("requires-node-with-cpu-manager")
	RequiresDualStackCluster             = Label("requires-dual-stack-cluster")
	RequiresHugepages2Mi                 = Label("requireHugepages2Mi")
	RequiresHugepages1Gi                 = Label("requireHugepages1Gi")
	GuestAgentProbes                     = Label("guest-agent-probes")

	/* Storage classes */

	// RequiresSnapshotStorageClass requires a storage class with support for snapshots
	RequiresSnapshotStorageClass = Label("RequiresSnapshotStorageClass")
	// RequiresWFFCStorageClass requires a storage class with support for WFFC bindingMode
	RequiresWFFCStorageClass = Label("RequiresWFFCStorageClass")
	// RequiresNoSnapshotStorageClass requires a storage class without support for snapshots
	RequiresNoSnapshotStorageClass = Label("RequiresNoSnapshotStorageClass")
	// RequiresRWXBlock requires a storage class with ReadWriteMany Block support
	RequiresRWXBlock = Label("RequiresRWXBlock")
	// RequiresRWOFsVMStateStorageClass requires the VMStateStorageClass to be set to ReadWriteOnce Filesystem storage class
	RequiresRWOFsVMStateStorageClass = Label("RequiresRWOFsVMStateStorageClass")
	// RequiresRWXFsVMStateStorageClass requires the VMStateStorageClass to be set to ReadWriteMany Filesystem storage class
	RequiresRWXFsVMStateStorageClass = Label("RequiresRWXFsVMStateStorageClass")

	// RequiresBlockStorage requires a storage class with Block storage support
	RequiresBlockStorage = Label("RequiresBlockStorage")
	// StorageCritical tests that ensure sig-storage functionality which are conformance-unready
	StorageCritical = Label("StorageCritical")
	// RequiresVolumeExpansion requires a storage class with volume expansion support
	RequiresVolumeExpansion = Label("RequiresVolumeExpansion")
	// RequiresDecentralizedLiveMigration request the feature gate is enabled
	RequiresDecentralizedLiveMigration = Label("RequiresDecentralizedLiveMigration")

	/* Provisioner */

	// RequiresSizeRoundUp requires a provisioner that rounds up the size of the volume
	RequiresSizeRoundUp = Label("RequiresSizeRoundUp")

	/* architecture working groups */

	WgS390x = Label("wg-s390x")
	WgArm64 = Label("wg-arm64")

	RequiresAMD64 = Label("requires-amd64")
	RequiresS390X = Label("requires-s390x")
	RequiresARM64 = Label("requires-arm64")

	// Virtctl related tests
	Virtctl = Label("virtctl")

	// NoFlakeCheck decorates tests that are not compatible with the check-tests-for-flakes test lane.
	// This should only be used for legitimate purposes, like on tests that have a flake-checker-friendly clone.
	NoFlakeCheck = Label("no-flake-check")
	// FlakeCheck decorates tests that are dedicated to the check-tests-for-flakes test lane.
	FlakeCheck = Label("flake-check")

	// Disruptive indicates that the test may cause a disruption to the cluster's normal operation
	Disruptive = Label("disruptive")

	// LargeStoragePoolRequired indicates that the test may fail in a cluster with a low storage pool capacity.
	// This decorator can be used to skip the test as the failure might not indicate a functional problem.
	LargeStoragePoolRequired = Label("large-storage-pool-required")

	// OncePerOrderedCleanup decorates Ordered tests to only cleanup after the last
	// test in an Ordered container.
	// Currently, in pilot mode, restricted to SIG-Network and virtctl only.
	OncePerOrderedCleanup = Label("OncePerOrderedCleanup")

	// Swap decorator is used in case a swap is required on a node.
	Swap = Label("SwapTest")
)
