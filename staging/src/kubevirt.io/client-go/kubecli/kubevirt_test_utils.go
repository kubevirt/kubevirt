package kubecli

import (
	"errors"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	v12 "kubevirt.io/client-go/apis/core/v1"
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

func NewMinimalMigration(name string) *v12.VirtualMachineInstanceMigration {
	return &v12.VirtualMachineInstanceMigration{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "VirtualMachineInstanceMigration"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewMinimalVM(name string) *v12.VirtualMachine {
	return &v12.VirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "VirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewMigrationList(migrations ...v12.VirtualMachineInstanceMigration) *v12.VirtualMachineInstanceMigrationList {
	return &v12.VirtualMachineInstanceMigrationList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "VirtualMachineInstanceMigrationList"}, Items: migrations}
}

func NewVMList(vms ...v12.VirtualMachine) *v12.VirtualMachineList {
	return &v12.VirtualMachineList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "VirtualMachineList"}, Items: vms}
}

func NewVirtualMachineInstanceReplicaSetList(rss ...v12.VirtualMachineInstanceReplicaSet) *v12.VirtualMachineInstanceReplicaSetList {
	return &v12.VirtualMachineInstanceReplicaSetList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "VirtualMachineInstanceReplicaSetList"}, Items: rss}
}

func NewMinimalVirtualMachineInstanceReplicaSet(name string) *v12.VirtualMachineInstanceReplicaSet {
	return &v12.VirtualMachineInstanceReplicaSet{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "VirtualMachineInstanceReplicaSet"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewMinimalKubeVirt(name string) *v12.KubeVirt {
	return &v12.KubeVirt{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "KubeVirt"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewKubeVirtList(kubevirts ...v12.KubeVirt) *v12.KubeVirtList {
	return &v12.KubeVirtList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "KubeVirtList"}, Items: kubevirts}
}

func NewVirtualMachineInstancePresetList(rss ...v12.VirtualMachineInstancePreset) *v12.VirtualMachineInstancePresetList {
	return &v12.VirtualMachineInstancePresetList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "VirtualMachineInstancePresetList"}, Items: rss}
}

func NewMinimalVirtualMachineInstancePreset(name string) *v12.VirtualMachineInstancePreset {
	return &v12.VirtualMachineInstancePreset{TypeMeta: k8smetav1.TypeMeta{APIVersion: v12.GroupVersion.String(), Kind: "VirtualMachineInstancePreset"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}
