package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/controller/networkaddonsconfig"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs = []func(manager.Manager) error{
	networkaddonsconfig.Add,
}

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}
