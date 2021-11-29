package flavor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/tools/cache"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/client-go/kubecli"
)

type Methods interface {
	FindProfile(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error)
	ApplyToVmi(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts
	CreateFlavorRevision(vm *virtv1.VirtualMachine) error
}

type Conflicts []*k8sfield.Path

func (c Conflicts) String() string {
	pathStrings := make([]string, 0, len(c))
	for _, path := range c {
		pathStrings = append(pathStrings, path.String())
	}
	return strings.Join(pathStrings, ", ")
}

type methods struct {
	flavorStore        cache.Store
	clusterFlavorStore cache.Store
	client             kubecli.KubevirtClient
}

var _ Methods = &methods{}

func NewMethods(client kubecli.KubevirtClient, flavorStore, clusterFlavorStore cache.Store) Methods {
	return &methods{
		flavorStore:        flavorStore,
		clusterFlavorStore: clusterFlavorStore,
		client:             client,
	}
}

func (m *methods) FindProfile(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
	if vm.Status.FlavorRevision != nil && vm.Status.FlavorRevision.Name != "" {
		rc, err := m.client.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), vm.Status.FlavorRevision.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		var profile flavorv1alpha1.VirtualMachineFlavorProfile

		err = json.Unmarshal(rc.Data.Raw, &profile)
		if err != nil {
			return nil, err
		}
		return &profile, nil
	}

	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	profiles, err := getProfiles(vm.Spec.Flavor.Name, vm.Namespace, vm.Spec.Flavor.Kind, m.flavorStore, m.clusterFlavorStore)
	if err != nil {
		return nil, err
	}

	if vm.Spec.Flavor.Profile == "" {
		profile := findFirstProfile(profiles, func(profile *flavorv1alpha1.VirtualMachineFlavorProfile) bool {
			return profile.Default
		})
		if profile == nil {
			return nil, fmt.Errorf("flavor does not specify a default profile")
		}
		return profile, nil
	} else {
		profile := findFirstProfile(profiles, func(profile *flavorv1alpha1.VirtualMachineFlavorProfile) bool {
			return profile.Name == vm.Spec.Flavor.Profile
		})
		if profile == nil {
			return nil, fmt.Errorf("flavor does not have a profile with name: %v", vm.Spec.Flavor.Profile)
		}
		return profile, nil
	}
}

func (m *methods) ApplyToVmi(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	var conflicts Conflicts

	conflicts = append(conflicts, applyCpu(field, profile, vmiSpec)...)

	return conflicts
}

func (m *methods) CreateFlavorRevision(vm *virtv1.VirtualMachine) error {
	// don't modify flavor revision if it already exists
	if vm.Status.FlavorRevision != nil {
		return nil
	}

	if vm.UID == "" {
		return nil
	}

	flavor, err := m.FindProfile(vm)
	if err != nil || flavor == nil {
		return err
	}

	createdCR, err := m.createControllerRevision(vm, flavor)
	if err != nil {
		return err
	}

	if vm.Status.FlavorRevision == nil {
		vm.Status.FlavorRevision = &virtv1.FlavorRevisionSpec{}
	}

	vm.Status.FlavorRevision.Name = createdCR.Name

	return nil
}

func getFlavorRevisionName(vmUID types.UID, generation int64) string {
	return fmt.Sprintf("%s-%d", fmt.Sprintf("revision-flavor-vm-%s", vmUID), generation)
}

func (m *methods) createControllerRevision(vm *virtv1.VirtualMachine, flavor *flavorv1alpha1.VirtualMachineFlavorProfile) (*appsv1.ControllerRevision, error) {
	flavorRevisionName := getFlavorRevisionName(vm.UID, vm.ObjectMeta.Generation)

	flavorBytes, err := json.Marshal(flavor)
	if err != nil {
		return nil, err
	}

	cr := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:            flavorRevisionName,
			Namespace:       vm.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
		},
		Data:     runtime.RawExtension{Raw: flavorBytes},
		Revision: vm.ObjectMeta.Generation,
	}

	createdCR, err := m.client.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
	if err != nil {
		if !k8serror.IsAlreadyExists(err) {
			return nil, err
		}
		return cr, nil
	}
	return createdCR, nil
}

func findFirstProfile(profiles []flavorv1alpha1.VirtualMachineFlavorProfile, predicate func(profile *flavorv1alpha1.VirtualMachineFlavorProfile) bool) *flavorv1alpha1.VirtualMachineFlavorProfile {
	for i := range profiles {
		profile := &profiles[i]
		if predicate(profile) {
			return profile
		}
	}
	return nil
}

func getKey(namespace string, name string) string {
	return namespace + "/" + name
}

func getProfiles(name string, namespace string, kind string, flavorStore, clusterFlavorStore cache.Store) ([]flavorv1alpha1.VirtualMachineFlavorProfile, error) {
	switch strings.ToLower(kind) {
	case "virtualmachineflavors", "virtualmachineflavor":
		key := getKey(namespace, name)
		obj, exists, err := flavorStore.GetByKey(key)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, k8serror.NewNotFound(flavorv1alpha1.Resource("virtualmachineflavor"), key)
		}
		flavor := obj.(*flavorv1alpha1.VirtualMachineFlavor)
		return flavor.Profiles, nil

	case "", "virtualmachineclusterflavors", "virtualmachineclusterflavor":
		obj, exists, err := clusterFlavorStore.GetByKey(name)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, k8serror.NewNotFound(flavorv1alpha1.Resource("virtualmachineclusterflavor"), name)
		}
		flavor := obj.(*flavorv1alpha1.VirtualMachineClusterFlavor)
		return flavor.Profiles, nil
	default:
		return nil, fmt.Errorf("got unexpected kind in FlavorMatcher: %s", kind)
	}
}

func applyCpu(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	if profile.CPU == nil {
		return nil
	}
	if vmiSpec.Domain.CPU != nil {
		return Conflicts{field.Child("domain", "cpu")}
	}

	vmiSpec.Domain.CPU = profile.CPU.DeepCopy()
	return nil
}
