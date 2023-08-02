package convertmachinetype

import (
	"context"
	"fmt"
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

	matchesGlob, err = matchMachineType(machine.Type)
	if err != nil {
		return false, err
	}
	return machine.Type == virtconfig.DefaultAMD64MachineType || !matchesGlob, nil
}

func (c *JobController) UpdateMachineType(vm *v1.VirtualMachine, running bool) error {
	err := c.patchMachineType(vm)
	if err != nil {
		return err
	}

	if running {
		// if force restart flag is set, restart running VMs immediately
		// don't apply warning label to VMs being restarted
		if RestartNow {
			return c.VirtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
		}

		// adding the warning label to the running VMs to indicate to the user
		// they must manually be restarted
		patchString := fmt.Sprintf(`[{ "op": "add", "path": "/status/machineTypeRestartRequired", "value": %t }]`, true)
		return c.statusUpdater.PatchStatus(vm, types.JSONPatchType, []byte(patchString), &metav1.PatchOptions{})
	}
	return nil
}
