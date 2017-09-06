package rest

import (
	"fmt"
	"log"
	"net/http"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sv1meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

func StartFunc(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		log.Fatal(err)
	}

	vmConfig, err := virtCli.VMConfig(namespace).Get(name, k8sv1meta.GetOptions{})
	if errors.IsNotFound(err) {
		response.WriteError(http.StatusNotFound, fmt.Errorf("VMConfig does not exist"))
		return
	}
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	domainSpec := ConfigToSpec(vmConfig)
	// TODO: Generate / supply different name, don't copy the VMConfig.
	vm := v1.NewMinimalVM(vmConfig.ObjectMeta.Name)
	vm.Spec.Domain = domainSpec

	_, err = virtCli.VM(namespace).Create(vm)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
	}

	response.WriteHeader(http.StatusOK)
}

func StopFunc(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		log.Fatal(err)
	}

	vmConfig, err := virtCli.VMConfig(namespace).Get(name, k8sv1meta.GetOptions{})
	if errors.IsNotFound(err) {
		response.WriteError(http.StatusNotFound, fmt.Errorf("VMConfig does not exist"))
		return
	}
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	// TODO: Needs to be symmetric to the StartFunc naming semantics.
	vmName := vmConfig.ObjectMeta.Name

	err = virtCli.VM(namespace).Delete(vmName, &k8sv1meta.DeleteOptions{})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
	}

	response.WriteHeader(http.StatusOK)
}
