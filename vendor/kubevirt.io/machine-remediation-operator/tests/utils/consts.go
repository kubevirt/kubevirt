package utils

import "time"

const (
	// KubeletKillerPodName contains the name of the pod that stops kubelet process
	KubeletKillerPodName = "kubelet-killer"
	// MachineAnnotationKey contains machine annotation key
	MachineAnnotationKey = "machine.openshift.io/machine"
	// MachineHealthCheckName contains the name of the machinehealthcheck used for tests
	MachineHealthCheckName = "workers-check"
	// NamespaceOpenShiftMachineAPI contains the openshift-machine-api namespace name
	NamespaceOpenShiftMachineAPI = "openshift-machine-api"
	// WorkerNodeRoleLabel contains the label of worker node
	WorkerNodeRoleLabel = "node-role.kubernetes.io/worker"
)

const (
	// WaitLong contains number of minutes to wait before long timeout
	WaitLong = 10 * time.Minute
)
