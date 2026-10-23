package kubecli

import (
	clone "kubevirt.io/api/clone/v1beta1"

	"kubevirt.io/api/migrations/v1alpha1"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

func NewMinimalMigration(name string) *v1.VirtualMachineInstanceMigration {
	return &v1.VirtualMachineInstanceMigration{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceMigration"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewMinimalVM(name string) *v1.VirtualMachine {
	return &v1.VirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewMigrationList(migrations ...v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstanceMigrationList {
	return &v1.VirtualMachineInstanceMigrationList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceMigrationList"}, Items: migrations}
}

func NewVMList(vms ...v1.VirtualMachine) *v1.VirtualMachineList {
	return &v1.VirtualMachineList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineList"}, Items: vms}
}

func NewVirtualMachineInstanceReplicaSetList(rss ...v1.VirtualMachineInstanceReplicaSet) *v1.VirtualMachineInstanceReplicaSetList {
	return &v1.VirtualMachineInstanceReplicaSetList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceReplicaSetList"}, Items: rss}
}

func NewMinimalVirtualMachineInstanceReplicaSet(name string) *v1.VirtualMachineInstanceReplicaSet {
	return &v1.VirtualMachineInstanceReplicaSet{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceReplicaSet"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewMinimalKubeVirt(name string) *v1.KubeVirt {
	return &v1.KubeVirt{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "KubeVirt"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewKubeVirtList(kubevirts ...v1.KubeVirt) *v1.KubeVirtList {
	return &v1.KubeVirtList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "KubeVirtList"}, Items: kubevirts}
}

func NewVirtualMachineInstancePresetList(rss ...v1.VirtualMachineInstancePreset) *v1.VirtualMachineInstancePresetList {
	return &v1.VirtualMachineInstancePresetList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstancePresetList"}, Items: rss}
}

func NewMinimalVirtualMachineInstancePreset(name string) *v1.VirtualMachineInstancePreset {
	return &v1.VirtualMachineInstancePreset{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstancePreset"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewMinimalMigrationPolicy(name string) *v1alpha1.MigrationPolicy {
	return &v1alpha1.MigrationPolicy{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: v1alpha1.MigrationPolicyKind.Kind},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewMinimalMigrationPolicyList(policies ...v1alpha1.MigrationPolicy) *v1alpha1.MigrationPolicyList {
	return &v1alpha1.MigrationPolicyList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: v1alpha1.MigrationPolicyListKind.Kind}, Items: policies}
}

func NewMinimalClone(name string) *clone.VirtualMachineClone {
	return NewMinimalCloneWithNS(name, "")
}

func NewMinimalCloneWithNS(name, namespace string) *clone.VirtualMachineClone {
	return &clone.VirtualMachineClone{
		TypeMeta: k8smetav1.TypeMeta{APIVersion: clone.SchemeGroupVersion.String(), Kind: clone.VirtualMachineCloneKind.Kind},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewMinimalCloneList(clones ...clone.VirtualMachineClone) *clone.VirtualMachineCloneList {
	return &clone.VirtualMachineCloneList{
		TypeMeta: k8smetav1.TypeMeta{APIVersion: clone.SchemeGroupVersion.String(), Kind: clone.VirtualMachineCloneListKind.Kind},
		Items:    clones,
	}
}
