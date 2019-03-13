package controller

import (
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/hyperconverged"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, hyperconverged.Add)
}
