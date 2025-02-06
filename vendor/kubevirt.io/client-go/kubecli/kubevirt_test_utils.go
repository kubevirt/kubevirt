package kubecli

import (
	"errors"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	v1alpha12 "kubevirt.io/api/clone/v1alpha1"

	"kubevirt.io/api/migrations/v1alpha1"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	v1 "kubevirt.io/api/core/v1"
)

// GetMockKubevirtClientFromClientConfig, MockKubevirtClientInstance are used to create a mechanism
// for overriding the actual kubevirt client access. This is useful when the unit tested code invokes GetKubevirtClientFromClientConfig()
// and therefore the unit test code cannot generate the mock client directly. In such a case following steps are needed:
// (1) Override the GetKubevirtClientFromClientConfig() closure with GetMockKubevirtClientFromClientConfig() or
//     GetInvalidKubevirtClientFromClientConfig()
// (2) Then create the instance of the client, and assign into MockKubevirtClientInstance before the tests start:
//     ctrl := gomock.NewController(GinkgoT())
//     kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
// (3) Rest of the kubevirt mocking is automatically generated in generated_mock_kubevirt.go

// MockKubevirtClientInstance is a reference to the kubevirt client that could be manipulated by the test code
var MockKubevirtClientInstance *MockKubevirtClient

// GetMockKubevirtClientFromClientConfig is an entry point for testing, could be used to override GetKubevirtClientFromClientConfig
func GetMockKubevirtClientFromClientConfig(cmdConfig clientcmd.ClientConfig) (KubevirtClient, error) {
	return MockKubevirtClientInstance, nil
}

// GetInvalidKubevirtClientFromClientConfig is an entry point for testing case where client should be invalid
func GetInvalidKubevirtClientFromClientConfig(cmdConfig clientcmd.ClientConfig) (KubevirtClient, error) {
	return nil, errors.New("invalid fake client")
}

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

func NewMinimalClone(name string) *v1alpha12.VirtualMachineClone {
	return NewMinimalCloneWithNS(name, "")
}

func NewMinimalCloneWithNS(name, namespace string) *v1alpha12.VirtualMachineClone {
	return &v1alpha12.VirtualMachineClone{
		TypeMeta: k8smetav1.TypeMeta{APIVersion: clonev1alpha1.SchemeGroupVersion.String(), Kind: clonev1alpha1.VirtualMachineCloneKind.Kind},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewMinimalCloneList(clones ...v1alpha12.VirtualMachineClone) *v1alpha12.VirtualMachineCloneList {
	return &v1alpha12.VirtualMachineCloneList{
		TypeMeta: k8smetav1.TypeMeta{APIVersion: clonev1alpha1.SchemeGroupVersion.String(), Kind: clonev1alpha1.VirtualMachineCloneListKind.Kind},
		Items:    clones,
	}
}
