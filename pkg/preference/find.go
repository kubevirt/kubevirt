//nolint:lll,goimports
package preference

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/instancetype/compatibility"
)

type Finder struct {
	PreferenceStore         cache.Store
	ClusterPreferenceStore  cache.Store
	ControllerRevisionStore cache.Store
	Clientset               kubecli.KubevirtClient
}

func (f *Finder) Find(vm *v1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	matcher := vm.Spec.Preference
	if matcher == nil {
		return nil, nil
	}

	if matcher.RevisionName != "" {
		return f.findSpecRevision(types.NamespacedName{
			Namespace: vm.Namespace,
			Name:      matcher.RevisionName,
		})
	}

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiinstancetype.SingularPreferenceResourceName, apiinstancetype.PluralPreferenceResourceName:
		preference, err := f.findPreference(vm)
		if err != nil {
			return nil, err
		}
		return &preference.Spec, nil

	case apiinstancetype.ClusterSingularPreferenceResourceName, apiinstancetype.ClusterPluralPreferenceResourceName, "":
		clusterPreference, err := f.findClusterPreference(vm)
		if err != nil {
			return nil, err
		}
		return &clusterPreference.Spec, nil

	default:
		return nil, fmt.Errorf("got unexpected kind in PreferenceMatcher: %s", vm.Spec.Preference.Kind)
	}
}

func (f *Finder) findSpecRevision(namespacedName types.NamespacedName) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	var (
		err      error
		revision *appsv1.ControllerRevision
	)

	if f.ControllerRevisionStore != nil {
		revision, err = f.getCRByInformer(namespacedName)
	} else {
		revision, err = f.getCRByClient(namespacedName)
	}

	if err != nil {
		return nil, err
	}

	return getSpecFromCR(revision)
}

func (f *Finder) getCRByInformer(namespacedName types.NamespacedName) (*appsv1.ControllerRevision, error) {
	obj, exists, err := f.ControllerRevisionStore.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.getCRByClient(namespacedName)
	}
	revision, ok := obj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in ControllerRevision informer")
	}
	return revision, nil
}

func (f *Finder) getCRByClient(namespacedName types.NamespacedName) (*appsv1.ControllerRevision, error) {
	revision, err := f.Clientset.AppsV1().ControllerRevisions(namespacedName.Namespace).Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return revision, nil
}

func (f *Finder) findPreference(vm *v1.VirtualMachine) (*v1beta1.VirtualMachinePreference, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}
	namespacedName := types.NamespacedName{
		Namespace: vm.Namespace,
		Name:      vm.Spec.Preference.Name,
	}
	if f.PreferenceStore != nil {
		return f.findPreferenceByInformer(namespacedName)
	}
	return f.findPreferenceByClient(namespacedName)
}

func (f *Finder) findPreferenceByInformer(namespacedName types.NamespacedName) (*v1beta1.VirtualMachinePreference, error) {
	obj, exists, err := f.PreferenceStore.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.findPreferenceByClient(namespacedName)
	}
	preference, ok := obj.(*v1beta1.VirtualMachinePreference)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachinePreference informer")
	}
	return preference, nil
}

func (f *Finder) findPreferenceByClient(namespacedName types.NamespacedName) (*v1beta1.VirtualMachinePreference, error) {
	preference, err := f.Clientset.VirtualMachinePreference(namespacedName.Namespace).Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return preference, nil
}

func (f *Finder) findClusterPreference(vm *v1.VirtualMachine) (*v1beta1.VirtualMachineClusterPreference, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}
	if f.ClusterPreferenceStore != nil {
		return f.findClusterPreferenceByInformer(vm.Spec.Preference.Name)
	}
	return f.findClusterPreferenceByClient(vm.Spec.Preference.Name)
}

func (f *Finder) findClusterPreferenceByInformer(resourceName string) (*v1beta1.VirtualMachineClusterPreference, error) {
	obj, exists, err := f.PreferenceStore.GetByKey(resourceName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.findClusterPreferenceByClient(resourceName)
	}
	preference, ok := obj.(*v1beta1.VirtualMachineClusterPreference)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachineClusterPreference informer")
	}
	return preference, nil
}

func (f *Finder) findClusterPreferenceByClient(resourceName string) (*v1beta1.VirtualMachineClusterPreference, error) {
	preference, err := f.Clientset.VirtualMachineClusterPreference().Get(context.Background(), resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return preference, nil
}

func getSpecFromCR(revision *appsv1.ControllerRevision) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	if err := compatibility.DecodeControllerRevision(revision); err != nil {
		return nil, err
	}
	switch obj := revision.Data.Object.(type) {
	case *v1beta1.VirtualMachinePreference:
		return &obj.Spec, nil
	case *v1beta1.VirtualMachineClusterPreference:
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}
