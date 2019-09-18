package consts

const (
	// AnnotationBareMetalHost contains the annotation key for bare metal host
	AnnotationBareMetalHost = "metal3.io/BareMetalHost"
	// AnnotationMachine contains the annotation key for machine
	AnnotationMachine = "machine.openshift.io/machine"
	// ControllerMachineDisruptionBudget contains the name of MachineDisruptionBudget controller
	ControllerMachineDisruptionBudget = "machine-disruption-budget"
	// ControllerMachineHealthCheck contains the name of MachineHealthCheck controller
	ControllerMachineHealthCheck = "machine-health-check"
	// ControllerMachineRemediation contains the name of achineRemediation controller
	ControllerMachineRemediation = "machine-remediation"
	//MasterRoleLabel contains master role label
	MasterRoleLabel = "node-role.kubernetes.io/master"
	// MasterMachineHealthCheck contains the MachineHealthCheck name for master nodes
	MasterMachineHealthCheck = "masters"
	// MasterMachineDisruptionBudget contains the MachineDisruptionBudget name for master nodes
	MasterMachineDisruptionBudget = "masters"
	// NamespaceOpenshiftMachineAPI contains namespace name for the machine-api componenets under the OpenShift cluster
	NamespaceOpenshiftMachineAPI = "openshift-machine-api"
)
