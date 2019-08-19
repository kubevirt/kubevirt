package utils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	mapiv1 "sigs.k8s.io/cluster-api/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetMachine get a machine by its name from the default machine API namespace.
func GetMachine(c client.Client, machineName string) (*mapiv1.Machine, error) {
	machine := &mapiv1.Machine{}
	if err := c.Get(context.TODO(), client.ObjectKey{Namespace: NamespaceOpenShiftMachineAPI, Name: machineName}, machine); err != nil {
		return nil, fmt.Errorf("error querying api for machine object: %v", err)
	}
	return machine, nil
}

// GetMachineFromNode returns the machine referenced by the "controllernode.MachineAnnotationKey" annotation in the given node
func GetMachineFromNode(c client.Client, node *corev1.Node) (*mapiv1.Machine, error) {
	machineNamespaceKey, ok := node.Annotations[MachineAnnotationKey]
	if !ok {
		return nil, fmt.Errorf("node %q does not have a MachineAnnotationKey %q", node.Name, MachineAnnotationKey)
	}
	namespace, machineName, err := cache.SplitMetaNamespaceKey(machineNamespaceKey)
	if err != nil {
		return nil, fmt.Errorf("machine annotation format is incorrect %v: %v", machineNamespaceKey, err)
	}

	if namespace != NamespaceOpenShiftMachineAPI {
		return nil, fmt.Errorf("Machine %q is forbidden to live outside of default %v namespace", machineNamespaceKey, NamespaceOpenShiftMachineAPI)
	}

	machine, err := GetMachine(c, machineName)
	if err != nil {
		return nil, fmt.Errorf("error querying api for machine object: %v", err)
	}

	return machine, nil
}

// GetMachinesSetByMachine retruns machine owner machine set
func GetMachinesSetByMachine(machine *mapiv1.Machine) (*mapiv1.MachineSet, error) {
	c, err := LoadClient()
	if err != nil {
		return nil, err
	}

	if len(machine.OwnerReferences) == 0 {
		return nil, fmt.Errorf("machine %s does not have owner controller", machine.Name)
	}

	machineSet := &mapiv1.MachineSet{}
	key := client.ObjectKey{
		Namespace: NamespaceOpenShiftMachineAPI,
		Name:      machine.OwnerReferences[0].Name,
	}
	err = c.Get(context.TODO(), key, machineSet)
	if err != nil {
		return nil, err
	}
	return machineSet, nil
}
