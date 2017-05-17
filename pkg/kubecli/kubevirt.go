package kubecli

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"net/http"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	k8smetav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

type KubevirtClient interface {
	VM(namespace string) VMInterface
}

type kubevirt struct {
	restClient *rest.RESTClient
}

func (k *kubevirt) VM(namespace string) VMInterface {
	return &vms{k.restClient, namespace}
}

type VMInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.VM, bool, error)
	List(opts k8sv1.ListOptions) (*v1.VMList, error)
	Create(*v1.VM) (*v1.VM, error)
	Update(*v1.VM) (*v1.VM, error)
	Delete(name string, options *k8sv1.DeleteOptions) error
}

type vms struct {
	restClient *rest.RESTClient
	namespace  string
}

func (v *vms) Get(name string, options k8smetav1.GetOptions) (vm *v1.VM, exists bool, err error) {
	vm = &v1.VM{}
	err = v.restClient.Get().
		Resource("vms").
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, api.ParameterCodec).
		Do().
		Into(vm)
	exists, err = checkExists(err)
	vm.SetGroupVersionKind(v1.GroupVersionKind)
	return
}

func (v *vms) List(options k8sv1.ListOptions) (vmList *v1.VMList, err error) {
	vmList = &v1.VMList{}
	err = v.restClient.Get().
		Resource("vms").
		Namespace(v.namespace).
		VersionedParams(&options, api.ParameterCodec).
		Do().
		Into(vmList)
	for _, vm := range vmList.Items {
		vm.SetGroupVersionKind(v1.GroupVersionKind)
	}

	return
}

func checkExists(err error) (bool, error) {
	if err != nil {
		err, ok := err.(*errors.StatusError)
		if ok && err.Status().Code == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (v *vms) Create(vm *v1.VM) (result *v1.VM, err error) {
	result = &v1.VM{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource("vms").
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.GroupVersionKind)
	return
}

func (v *vms) Update(vm *v1.VM) (result *v1.VM, err error) {
	result = &v1.VM{}
	err = v.restClient.Put().
		Namespace(v.namespace).
		Resource("vms").
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.GroupVersionKind)
	return
}

func (v *vms) Delete(name string, options *k8sv1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource("vms").
		Name(name).
		Body(options).
		Do().
		Error()
}
