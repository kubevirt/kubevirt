package controller

import (
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, hcoutil.ClusterInfo) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, ci hcoutil.ClusterInfo) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, ci); err != nil {
			return err
		}
	}
	return nil
}
