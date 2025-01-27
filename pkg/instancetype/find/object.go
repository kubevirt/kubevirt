package find

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	api "kubevirt.io/api/instancetype"
	"kubevirt.io/client-go/kubecli"
)

type objectFinder struct {
	instancetypeFinder        *instancetypeFinder
	clusterInstancetypeFinder *clusterInstancetypeFinder
	revisionFinder            *revisionFinder
}

func NewObjectFinder(store, clusterStore, revisionStore cache.Store, virtClient kubecli.KubevirtClient) *objectFinder {
	return &objectFinder{
		instancetypeFinder:        NewInstancetypeFinder(store, virtClient),
		clusterInstancetypeFinder: NewClusterInstancetypeFinder(clusterStore, virtClient),
		revisionFinder:            NewRevisionFinder(revisionStore, virtClient),
	}
}

func decode(runtimeObj runtime.Object) (metav1.Object, error) {
	obj, ok := runtimeObj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("unknown type found within runtime Object")
	}
	return obj, nil
}

func (o *objectFinder) Find(vm *virtv1.VirtualMachine) (metav1.Object, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}

	if vm.Spec.Instancetype.RevisionName != "" {
		revision, err := o.revisionFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return decode(revision.Data.Object)
	}

	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case api.SingularResourceName, api.PluralResourceName:
		instancetype, err := o.instancetypeFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return decode(instancetype)

	case api.ClusterSingularResourceName, api.ClusterPluralResourceName, "":
		clusterInstancetype, err := o.clusterInstancetypeFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return decode(clusterInstancetype)

	default:
		return nil, fmt.Errorf(unexpectedKindFmt, vm.Spec.Instancetype.Kind)
	}
}
