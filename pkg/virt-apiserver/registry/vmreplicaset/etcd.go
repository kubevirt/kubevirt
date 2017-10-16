package vmreplicaset

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
)

type REST struct {
	*genericregistry.Store
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (*REST, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		Copier:                   scheme,
		NewFunc:                  func() runtime.Object { return &v1.VirtualMachineReplicaSet{} },
		NewListFunc:              func() runtime.Object { return &v1.VirtualMachineReplicaSetList{} },
		PredicateFunc:            MatchVM,
		DefaultQualifiedResource: v1.Resource("virtualmachinereplicasets"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}

	if err := store.CompleteWithOptions(options); err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("Unable to create REST storage for virtualmachinereplicaset resource")
		return nil, fmt.Errorf("Unable to create REST storage for virtualmachinereplicaset resource: %v.", err)
	}
	return &REST{store}, nil
}
