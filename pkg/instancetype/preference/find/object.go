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
	preferenceFinder        *preferenceFinder
	clusterPreferenceFinder *clusterPreferenceFinder
	revisionFinder          *revisionFinder
}

func NewObjectFinder(store, clusterStore, revisionStore cache.Store, virtClient kubecli.KubevirtClient) *objectFinder {
	return &objectFinder{
		preferenceFinder:        NewPreferenceFinder(store, virtClient),
		clusterPreferenceFinder: NewClusterPreferenceFinder(clusterStore, virtClient),
		revisionFinder:          NewRevisionFinder(revisionStore, virtClient),
	}
}

func decode(runtimeObj runtime.Object) (metav1.Object, error) {
	obj, ok := runtimeObj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("unknown type found within runtime Object")
	}
	return obj, nil
}

func (o *objectFinder) FindPreference(vm *virtv1.VirtualMachine) (metav1.Object, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}

	if vm.Spec.Preference.RevisionName != "" {
		revision, err := o.revisionFinder.FindPreference(vm)
		if err != nil {
			return nil, err
		}
		return decode(revision.Data.Object)
	}

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case api.SingularResourceName, api.PluralResourceName:
		preference, err := o.preferenceFinder.FindPreference(vm)
		if err != nil {
			return nil, err
		}
		return decode(preference)

	case api.ClusterSingularResourceName, api.ClusterPluralResourceName, "":
		clusterPreference, err := o.clusterPreferenceFinder.FindPreference(vm)
		if err != nil {
			return nil, err
		}
		return decode(clusterPreference)

	default:
		return nil, fmt.Errorf(unexpectedKindFmt, vm.Spec.Preference.Kind)
	}
}

func (o *objectFinder) FindPreferenceFromVMI(vmi *virtv1.VirtualMachineInstance) (metav1.Object, error) {
	if _, ok := vmi.GetLabels()[virtv1.InstancetypeAnnotation]; ok {
		instancetype, err := o.preferenceFinder.FindPreferenceFromVMI(vmi)
		if err != nil {
			return nil, err
		}
		return decode(instancetype)
	}
	if _, ok := vmi.GetLabels()[virtv1.ClusterInstancetypeAnnotation]; ok {
		instancetype, err := o.clusterPreferenceFinder.FindPreferenceFromVMI(vmi)
		if err != nil {
			return nil, err
		}
		return decode(instancetype)
	}
	return nil, nil
}
