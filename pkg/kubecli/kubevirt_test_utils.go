package kubecli

import (
	"errors"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
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
