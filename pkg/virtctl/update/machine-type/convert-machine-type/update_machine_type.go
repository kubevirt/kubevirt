package convertmachinetype

import (
	"context"
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var (
	// machine type(s) which should be updated
	MachineTypeGlob = ""
	LabelSelector   labels.Selector
	// by default, update machine type across all namespaces
	Namespace = metav1.NamespaceAll
	// by default, should require manual restarting of VMIs
	RestartNow = false
	Testing    = false
)

func matchMachineType(machineType string) (bool, error) {
	matchMachineType, err := path.Match(MachineTypeGlob, machineType)
	if !matchMachineType || err != nil {
		return false, err
	}

	return true, nil
}

func (c *JobController) patchMachineType(vm *v1.VirtualMachine) error {
	// removing the machine type field from the VM spec reverts it to
	// the default machine type of the VM's arch
	updateMachineType := `[{"op": "remove", "path": "/spec/template/spec/domain/machine"}]`

	_, err := c.VirtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, []byte(updateMachineType), &metav1.PatchOptions{})
	return err
}

func isMachineTypeUpdated(vm *v1.VirtualMachine) (bool, error) {
	machine := vm.Spec.Template.Spec.Domain.Machine
	matchesGlob := false
	var err error

	// when running unit tests, updating the machine type
	// does not update it to the aliased machine type when
	// setting it to nil. This is to account for that for now
	if machine == nil && Testing {
		return true, nil
	}

	if machine.Type == virtconfig.DefaultAMD64MachineType {
		return true, nil
	}

	matchesGlob, err = matchMachineType(machine.Type)
	return !matchesGlob, err
}
