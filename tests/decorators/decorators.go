package decorators

import . "github.com/onsi/ginkgo/v2"

var (
	SigCompute           = []interface{}{Label("sig-compute")}
	SigOperator          = []interface{}{Label("sig-operator")}
	SigNetwork           = []interface{}{Label("sig-network")}
	SigStorage           = []interface{}{Label("sig-storage")}
	SigComputeRealtime   = []interface{}{Label("sig-compute-realtime")}
	SigComputeMigrations = []interface{}{Label("sig-compute-migrations")}
	SigMonitoring        = []interface{}{Label("sig-monitoring")}
	StorageReq           = []interface{}{Label("storage-req")}
	Sysprep              = []interface{}{Label("Sysprep")}
	Windows              = []interface{}{Label("Windows")}
	Multus               = []interface{}{Label("Multus")}
	Networking           = []interface{}{Label("Networking")}
	VMIlifecycle         = []interface{}{Label("VMIlifecycle")}
	Expose               = []interface{}{Label("Expose")}
	Macvtap              = []interface{}{Label("Macvtap")}
	GPU                  = []interface{}{Label("GPU")}
	VGPU                 = []interface{}{Label("VGPU")}
	SRIOV                = []interface{}{Label("SRIOV")}
	NonRoot              = []interface{}{Label("verify-non-root")}
	NativeSsh            = []interface{}{Label("native-ssh")}
	ExcludeNativeSsh     = []interface{}{Label("exclude-native-ssh")}
	Reenlightenment      = []interface{}{Label("Reenlightenment")}
	Invtsc               = []interface{}{Label("Invtsc")}
	PasstGate            = []interface{}{Label("PasstGate")}
)
