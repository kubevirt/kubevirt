package decorators

import . "github.com/onsi/ginkgo/v2"

var (
	//
	Quarantine = []interface{}{Label("QUARANTINE")}

	// SIGs
	SigCompute           = []interface{}{Label("sig-compute")}
	SigOperator          = []interface{}{Label("sig-operator")}
	SigNetwork           = []interface{}{Label("sig-network")}
	SigStorage           = []interface{}{Label("sig-storage")}
	SigComputeRealtime   = []interface{}{Label("sig-compute-realtime")}
	SigComputeMigrations = []interface{}{Label("sig-compute-migrations")}
	SigMonitoring        = []interface{}{Label("sig-monitoring")}

	// HW
	GPU         = []interface{}{Label("GPU")}
	VGPU        = []interface{}{Label("VGPU")}
	SEV         = []interface{}{Label("SEV")}
	SRIOV       = []interface{}{Label("SRIOV")}
	StorageReq  = []interface{}{Label("storage-req")}
	Multus      = []interface{}{Label("Multus")}
	Macvtap     = []interface{}{Label("Macvtap")}
	Invtsc      = []interface{}{Label("Invtsc")}
	KSMRequired = []interface{}{Label("KSM-required")}

	// Features
	Sysprep                              = []interface{}{Label("Sysprep")}
	Windows                              = []interface{}{Label("Windows")}
	Networking                           = []interface{}{Label("Networking")}
	VMIlifecycle                         = []interface{}{Label("VMIlifecycle")}
	Expose                               = []interface{}{Label("Expose")}
	NativeSsh                            = []interface{}{Label("native-ssh")}
	ExcludeNativeSsh                     = []interface{}{Label("exclude-native-ssh")}
	Reenlightenment                      = []interface{}{Label("Reenlightenment")}
	TscFrequencies                       = []interface{}{Label("TscFrequencies")}
	PasstGate                            = []interface{}{Label("PasstGate")}
	VMX                                  = []interface{}{Label("VMX")}
	Upgrade                              = []interface{}{Label("Upgrade")}
	Istio                                = []interface{}{Label("Istio")}
	InPlaceHotplugNICs                   = []interface{}{Label("in-place-hotplug-NICs")}
	MigrationBasedHotplugNICs            = []interface{}{Label("migration-based-hotplug-NICs")}
	NetCustomBindingPlugins              = []interface{}{Label("netCustomBindingPlugins")}
	RequiresTwoSchedulableNodes          = []interface{}{Label("requires-two-schedulable-nodes")}
	VMLiveUpdateFeaturesGate             = []interface{}{Label("VMLiveUpdateFeaturesGate")}
	RequiresRWXFilesystemStorage         = []interface{}{Label("rwxfs")}
	USB                                  = []interface{}{Label("USB")}
	AutoResourceLimitsGate               = []interface{}{Label("AutoResourceLimitsGate")}
	RequiresTwoWorkerNodesWithCPUManager = []interface{}{Label("requires-two-worker-nodes-with-cpu-manager")}

	// Storage classes
	RequiresSnapshotStorageClass = []interface{}{Label("RequiresSnapshotStorageClass")}
	// Requires a storage class without support for snapshots
	RequiresNoSnapshotStorageClass = []interface{}{Label("RequiresNoSnapshotStorageClass")}
)
