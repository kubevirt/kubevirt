package machineremediation

import (
	"context"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
)

// Remediator apply machine remediation strategy under a specific infrastructure.
type Remediator interface {
	// Reboot the machine.
	Reboot(context.Context, *mrv1.MachineRemediation) error
	// Recreate the machine.
	Recreate(context.Context, *mrv1.MachineRemediation) error
}
