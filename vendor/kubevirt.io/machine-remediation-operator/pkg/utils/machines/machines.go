package machines

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/utils/conditions"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	mapiv1 "sigs.k8s.io/cluster-api/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsMachineHealthy returns true if the the machine is running and machine node is healthy
func IsMachineHealthy(c client.Client, machine *mapiv1.Machine) (bool, error) {
	if machine.Status.NodeRef == nil {
		return false, fmt.Errorf("Machine %s does not have node reference", machine.Name)
	}

	node := &v1.Node{}
	key := client.ObjectKey{Namespace: metav1.NamespaceNone, Name: machine.Status.NodeRef.Name}
	err := c.Get(context.TODO(), key, node)
	if err != nil {
		return false, err
	}

	cmUnhealtyConditions, err := conditions.GetUnhealthyConditionsConfigMap(c, machine.Namespace)
	if err != nil {
		return false, err
	}

	nodeUnhealthyConditions, err := conditions.GetNodeUnhealthyConditions(node, cmUnhealtyConditions)
	if err != nil {
		return false, err
	}

	if len(nodeUnhealthyConditions) > 0 {
		glog.Infof("Machine %q unhealthy because of conditions: %v", machine.Name, nodeUnhealthyConditions)
		return false, nil
	}

	return true, nil
}

// GetMachineMachineDisruptionBudgets returns list of machine disruption budgets that suit for the machine
func GetMachineMachineDisruptionBudgets(c client.Client, machine *mapiv1.Machine) ([]*mrv1.MachineDisruptionBudget, error) {
	if len(machine.Labels) == 0 {
		return nil, fmt.Errorf("no MachineDisruptionBudgets found for machine %v because it has no labels", machine.Name)
	}

	list := &mrv1.MachineDisruptionBudgetList{}
	err := c.List(context.TODO(), list, client.InNamespace(machine.Namespace))
	if err != nil {
		return nil, err
	}

	var mdbs []*mrv1.MachineDisruptionBudget
	for i := range list.Items {
		mdb := &list.Items[i]
		selector, err := metav1.LabelSelectorAsSelector(mdb.Spec.Selector)
		if err != nil {
			glog.Warningf("invalid selector: %v", err)
			continue
		}

		// If a mdb with a nil or empty selector creeps in, it should match nothing, not everything.
		if selector.Empty() || !selector.Matches(labels.Set(machine.Labels)) {
			continue
		}
		mdbs = append(mdbs, mdb)
	}

	return mdbs, nil
}
