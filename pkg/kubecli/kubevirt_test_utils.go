package kubecli

import (
	"errors"

	"kubevirt.io/kubevirt/pkg/api/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// GetMockKubevirtClientFromClientConfig, MockKubevirtClientInstance and InvalidMockClient are used to create a mechanism
// for overriding the actual kubevirt client access. This is useful when the unit tested code invokes GetKubevirtClientFromClientConfig()
// and therefore the unit test code cannot generate the mock client directly. In such a case following steps are needed:
// (1) set InvalidMockClient to true/false to control the behavior of GetKubevirtClientFromClientConfig()
// (2) if set to "false", you should override the GetKubevirtClientFromClientConfig() closure with GetMockKubevirtClientFromClientConfig()
// (3) then create the instance of the client, and assign into MockKubevirtClientInstance before the tests start:
//     ctrl := gomock.NewController(GinkgoT())
//     kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
// (4) rest of the kubevirt mocking is automatically generated in generated_mock_kubevirt.go

// InvalidMockClient is a parameter to control behavior of mock objects
var InvalidMockClient bool

// MockKubevirtClientInstance is a reference to the kubevirt client that could be manipulated by the test code
var MockKubevirtClientInstance *MockKubevirtClient

// GetMockKubevirtClientFromClientConfig is an entry point for testing, could be used to override GetKubevirtClientFromClientConfig
func GetMockKubevirtClientFromClientConfig(cmdConfig clientcmd.ClientConfig) (KubevirtClient, error) {
	if InvalidMockClient {
		return nil, errors.New("invalid fake client")
	}
	return MockKubevirtClientInstance, nil
}

func NewMinimalOVM(name string) *v1.OfflineVirtualMachine {
	return &v1.OfflineVirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "OfflineVirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewOVMList(ovms ...v1.OfflineVirtualMachine) *v1.OfflineVirtualMachineList {
	return &v1.OfflineVirtualMachineList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "OfflineVirtualMachineList"}, Items: ovms}
}

func NewVMReplicaSetList(rss ...v1.VirtualMachineReplicaSet) *v1.VirtualMachineReplicaSetList {
	return &v1.VirtualMachineReplicaSetList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineReplicaSetList"}, Items: rss}
}

func NewMinimalVMReplicaSet(name string) *v1.VirtualMachineReplicaSet {
	return &v1.VirtualMachineReplicaSet{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineReplicaSet"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewVMList(vms ...v1.VirtualMachine) *v1.VirtualMachineList {
	return &v1.VirtualMachineList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineList"}, Items: vms}
}
