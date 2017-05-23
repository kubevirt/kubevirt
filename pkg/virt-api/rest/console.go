package rest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	k8scorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	k8sv1meta "k8s.io/client-go/pkg/apis/meta/v1"

	"k8s.io/client-go/pkg/api/errors"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Console struct {
	virtClient      kubecli.KubevirtClient
	k8sClient       k8scorev1.CoreV1Interface
	VirtHandlerPort string
}

func NewConsoleResource(virtClient kubecli.KubevirtClient, k8sClient k8scorev1.CoreV1Interface) *Console {
	return &Console{virtClient: virtClient, k8sClient: k8sClient}
}

func (t *Console) Console(request *restful.Request, response *restful.Response) {
	console := request.QueryParameter("console")
	vmName := request.PathParameter("name")

	vm, err := t.virtClient.VM(v1.NamespaceDefault).Get(vmName, k8sv1meta.GetOptions{})
	if errors.IsNotFound(err) {
		logging.DefaultLogger().Info().V(3).Msgf("VM '%s' does not exist", vmName)
		response.WriteError(http.StatusNotFound, fmt.Errorf("VM does not exist"))
		return
	}
	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msgf("Error fetching VM '%s'", vmName)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	log := logging.DefaultLogger().Object(vm)

	if !vm.IsRunning() {
		log.Info().V(3).Reason(err).Msg("VM is not running")
		response.WriteError(http.StatusBadRequest, fmt.Errorf("VM is not running"))
		return
	}

	// Get virt-handler pod
	targetNode, err := t.k8sClient.Nodes().Get(vm.Status.NodeName, k8sv1meta.GetOptions{})
	if err != nil {
		log.Error().Reason(err).Msgf("Could not fetch node '%s' where the VM is running on", vm.Status.NodeName)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	var dstAddr string
	for _, addr := range targetNode.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			dstAddr = addr.Address
			break
		}
	}

	if dstAddr == "" {
		log.Error().Reason(err).Msgf("Could not determine internal IP of node '%s'", vm.Status.NodeName)
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("Could not find a connection IP for node %s", vm.Status.NodeName))
		return
	}

	// FIXME, don't hardcode virt-handler port. virt-handler should register itself somehow
	port := "8185"
	if t.VirtHandlerPort != "" {
		port = t.VirtHandlerPort
	}

	u := url.URL{Scheme: "ws", Host: dstAddr + ":" + port, Path: fmt.Sprintf("/api/v1/console/%s", vmName)}
	if console != "" {
		u.RawQuery = "console=" + console
	}
	handlerSocket, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err := fmt.Errorf("%s", buf.String())
			log.Error().Reason(err).
				With("statusCode", resp.StatusCode).
				Msgf("Failed to connect to virt-handler")
			response.WriteError(resp.StatusCode, err)
		} else {
			log.Error().Reason(err).Msgf("Failed to connect to virt-handler")
			response.WriteError(http.StatusInternalServerError, err)
		}
		return
	}
	defer handlerSocket.Close()

	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		log.Error().Reason(err).Msgf("Failed to upgrade client websocket connection")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	defer clientSocket.Close()

	errorChan := make(chan error)

	go func() {
		_, err := io.Copy(clientSocket.UnderlyingConn(), handlerSocket.UnderlyingConn())
		errorChan <- err
	}()

	go func() {
		_, err := io.Copy(handlerSocket.UnderlyingConn(), clientSocket.UnderlyingConn())
		errorChan <- err
	}()

	err = <-errorChan
	if err != nil {
		log.Error().Reason(err).Msgf("Proxied Web Socket connection failed")
	}
	response.WriteHeader(http.StatusOK)
}
