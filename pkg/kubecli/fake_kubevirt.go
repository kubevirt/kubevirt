package kubecli

import (
	"errors"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	k8sapiv1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

// parameter to control behavior of fake objects
var InvalidFakeClient bool
var InvalidFakeResource bool
var InvalidFakeService bool

// holding a copy of the created service, so that the test code could verify
var CurrentFakeService *k8sapiv1.Service

// entry point for testing, could be used to override GetKubevirtClientFromClientConfig
func GetFakeKubevirtClientFromClientConfig(cmdConfig clientcmd.ClientConfig) (KubevirtClient, error) {
	var cli FakeKubevirtClient
	if InvalidFakeClient {
		return cli, errors.New("invalid fake client")
	} else {
		return cli, nil
	}
}

// implementation of the KubevirtClient interface returning fake objects
type FakeKubevirtClient struct {
	KubevirtClient
}

func (FakeKubevirtClient) VM(namespace string) VMInterface {
	return FakeVMInterface{}
}
func (FakeKubevirtClient) OfflineVirtualMachine(namespace string) OfflineVirtualMachineInterface {
	return FakeOfflineVirtualMachineInterface{}
}
func (FakeKubevirtClient) ReplicaSet(namespace string) ReplicaSetInterface {
	return FakeReplicaSetInterface{}
}
func (FakeKubevirtClient) CoreV1() k8sv1.CoreV1Interface {
	return FakeCoreV1Interface{}
}

// implementation of CoreV1Interface returning fake service object
type FakeCoreV1Interface struct {
	k8sv1.CoreV1Interface
}

func (FakeCoreV1Interface) Services(namespace string) k8sv1.ServiceInterface {
	return FakeServiceInterface{}
}

// implementation of ServiceInterface returning fake service object
type FakeServiceInterface struct {
	k8sv1.ServiceInterface
}

func (FakeServiceInterface) Create(service *k8sapiv1.Service) (*k8sapiv1.Service, error) {
	if InvalidFakeService {
		return nil, errors.New("invalid fake service")
	}
	// copy the service over, so it could be observed from the outside
	CurrentFakeService = service
	return service, nil
}

// VMInterface implementation returning fake objects
type FakeVMInterface struct {
	VMInterface
}

func (FakeVMInterface) Get(name string, options k8smetav1.GetOptions) (*v1.VirtualMachine, error) {
	if InvalidFakeResource {
		return nil, errors.New("invalid fake resource")
	}
	return &v1.VirtualMachine{}, nil
}

// OfflineVirtualMachineInterface implementation returning fake objects
type FakeOfflineVirtualMachineInterface struct {
	OfflineVirtualMachineInterface
}

func (FakeOfflineVirtualMachineInterface) Get(name string, options k8smetav1.GetOptions) (*v1.OfflineVirtualMachine, error) {
	if InvalidFakeResource {
		return nil, errors.New("invalid fake resource")
	}
	return &v1.OfflineVirtualMachine{}, nil
}

// ReplicaSetInterface implementation returning fake objects
type FakeReplicaSetInterface struct {
	ReplicaSetInterface
}

func (FakeReplicaSetInterface) Get(name string, options k8smetav1.GetOptions) (*v1.VirtualMachineReplicaSet, error) {
	if InvalidFakeResource {
		return nil, errors.New("invalid fake resource")
	}
	return &v1.VirtualMachineReplicaSet{
		Spec: v1.VMReplicaSetSpec{
			Selector: &k8smetav1.LabelSelector{
				MatchLabels: map[string]string{"kubevirt.io/vmReplicaSet": "testvm"},
			},
		},
	}, nil
}
