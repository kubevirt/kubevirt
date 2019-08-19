package components

const (
	// ComponentMachineDisruptionBudget contains name for MachineDisruptionBudget component
	ComponentMachineDisruptionBudget = "machine-disruption-budget"
	// ComponentMachineHealthCheck contains name for MachineHealthCheck component
	ComponentMachineHealthCheck = "machine-health-check"
	// ComponentMachineRemediation contains name for MachineRemediation component
	ComponentMachineRemediation = "machine-remediation"
	// ComponentMachineRemediationOperator contains name for MachineRemediationOperator component
	ComponentMachineRemediationOperator = "machine-remediation-operator"
)

var (
	// Components contains names of all componenets that the operator should deploy
	Components = []string{
		ComponentMachineDisruptionBudget,
		ComponentMachineHealthCheck,
		ComponentMachineRemediation,
	}
)

const (
	// EnvVarOperatorVersion contains the name of operator version environment variable
	EnvVarOperatorVersion = "OPERATOR_VERSION"
)

const (
	// CRDMachineDisruptionBudget contains the kind of the MachineDisruptionBudget CRD
	CRDMachineDisruptionBudget = "machinedisruptionbudget"
	// CRDMachineHealthCheck contains the kind of the MachineHealthCheck CRD
	CRDMachineHealthCheck = "machinehealthcheck"
	// CRDMachineRemediation contains the kind of the MachineRemediation CRD
	CRDMachineRemediation = "machineremediation"
)

var (
	// CRDS contains names of all CRD's that the operator should deploy
	CRDS = []string{
		CRDMachineDisruptionBudget,
		CRDMachineHealthCheck,
		CRDMachineRemediation,
	}
)
