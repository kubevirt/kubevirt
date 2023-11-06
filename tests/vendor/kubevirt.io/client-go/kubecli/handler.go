package kubecli

import (
	"context"
	"fmt"
	"io"
	"net/http"

	v1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	netutils "k8s.io/utils/net"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/util"
)

const (
	consoleTemplateURI        = "wss://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/console"
	usbredirTemplateURI       = "wss://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/usbredir"
	vncTemplateURI            = "wss://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/vnc"
	vsockTemplateURI          = "wss://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/vsock"
	pauseTemplateURI          = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/pause"
	unpauseTemplateURI        = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/unpause"
	freezeTemplateURI         = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/freeze"
	unfreezeTemplateURI       = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/unfreeze"
	softRebootTemplateURI     = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/softreboot"
	guestInfoTemplateURI      = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/guestosinfo"
	userListTemplateURI       = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/userlist"
	filesystemListTemplateURI = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/filesystemlist"

	sevFetchCertChainTemplateURI         = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/sev/fetchcertchain"
	sevQueryLaunchMeasurementTemplateURI = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/sev/querylaunchmeasurement"
	sevInjectLaunchSecretTemplateURI     = "https://%s:%v/v1/namespaces/%s/virtualmachineinstances/%s/sev/injectlaunchsecret"
)

func NewVirtHandlerClient(virtCli KubevirtClient, httpCli *http.Client) VirtHandlerClient {
	return &virtHandler{
		virtCli:         virtCli,
		httpCli:         httpCli,
		virtHandlerPort: 0,
		namespace:       "",
	}
}

type VirtHandlerClient interface {
	ForNode(nodeName string) VirtHandlerConn
	Port(port int) VirtHandlerClient
	Namespace(namespace string) VirtHandlerClient
}

type VirtHandlerConn interface {
	ConnectionDetails() (ip string, port int, err error)
	ConsoleURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	USBRedirURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	VNCURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	VSOCKURI(vmi *virtv1.VirtualMachineInstance, port string, tls string) (string, error)
	PauseURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	UnpauseURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	FreezeURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	UnfreezeURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	SoftRebootURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	SEVFetchCertChainURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	SEVQueryLaunchMeasurementURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	SEVInjectLaunchSecretURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	Pod() (pod *v1.Pod, err error)
	Put(url string, body io.ReadCloser) error
	Get(url string) (string, error)
	GuestInfoURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	UserListURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	FilesystemListURI(vmi *virtv1.VirtualMachineInstance) (string, error)
}

type virtHandler struct {
	virtCli         KubevirtClient
	httpCli         *http.Client
	virtHandlerPort int
	namespace       string
}

type virtHandlerConn struct {
	pod        *v1.Pod
	err        error
	port       int
	httpClient *http.Client
}

func (v *virtHandler) Namespace(namespace string) VirtHandlerClient {
	v.namespace = namespace
	return v
}

func (v *virtHandler) Port(port int) VirtHandlerClient {
	v.virtHandlerPort = port
	return v
}
func (v *virtHandler) ForNode(nodeName string) VirtHandlerConn {
	var err error

	conn := &virtHandlerConn{
		httpClient: v.httpCli,
	}

	namespace := v.namespace
	if namespace == "" {
		namespace, err = util.GetNamespace()
		if err != nil {
			conn.err = err
			return conn
		}
	}
	pod, found, err := v.getVirtHandler(nodeName, namespace)
	if !found {
		conn.err = fmt.Errorf("No virt-handler on node %s found", nodeName)
	}
	if err != nil {
		conn.err = err
	}
	conn.pod = pod
	conn.port = v.virtHandlerPort
	return conn
}

func (v *virtHandler) getVirtHandler(nodeName string, namespace string) (*v1.Pod, bool, error) {

	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + nodeName)
	labelSelector, err := labels.Parse(virtv1.AppLabel + " in (virt-handler)")
	if err != nil {
		return nil, false, err
	}

	pods, err := v.virtCli.CoreV1().Pods(namespace).List(context.Background(),
		k8smetav1.ListOptions{
			FieldSelector: handlerNodeSelector.String(),
			LabelSelector: labelSelector.String()})
	if err != nil {
		return nil, false, err
	}
	if len(pods.Items) > 1 {
		return nil, false, fmt.Errorf("Expected to find one Pod, found %d Pods", len(pods.Items))
	}

	if len(pods.Items) == 0 {
		return nil, false, nil
	}
	return &pods.Items[0], true, nil
}

func (v *virtHandlerConn) ConnectionDetails() (ip string, port int, err error) {
	if v.err != nil {
		err = v.err
		return
	}
	// TODO depending on in which network namespace virt-handler runs, we might have to choose the NodeIPt d
	ip = v.pod.Status.PodIP
	// TODO get rid of the hardcoded port
	port = 8185
	if v.port != 0 {
		port = v.port
	}
	return
}

func (v *virtHandlerConn) formatURI(template string, vmi *virtv1.VirtualMachineInstance) (string, error) {
	ip, port, err := v.ConnectionDetails()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(template, formatIpForUri(ip), port, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name), nil
}

// TODO move the actual ws handling in here, and work with channels
func (v *virtHandlerConn) ConsoleURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(consoleTemplateURI, vmi)
}

func (v *virtHandlerConn) USBRedirURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(usbredirTemplateURI, vmi)
}

func (v *virtHandlerConn) VNCURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(vncTemplateURI, vmi)
}

func (v *virtHandlerConn) VSOCKURI(vmi *virtv1.VirtualMachineInstance, port string, tls string) (string, error) {
	baseURI, err := v.formatURI(vsockTemplateURI, vmi)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s?port=%s&tls=%s", baseURI, port, tls), nil
}

func (v *virtHandlerConn) FreezeURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(freezeTemplateURI, vmi)
}

func (v *virtHandlerConn) UnfreezeURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(unfreezeTemplateURI, vmi)
}

func (v *virtHandlerConn) SoftRebootURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(softRebootTemplateURI, vmi)
}

func (v *virtHandlerConn) PauseURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(pauseTemplateURI, vmi)
}

func (v *virtHandlerConn) UnpauseURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(unpauseTemplateURI, vmi)
}

func (v *virtHandlerConn) Pod() (pod *v1.Pod, err error) {
	if v.err != nil {
		err = v.err
		return
	}
	return v.pod, err
}

func (v *virtHandlerConn) doRequest(req *http.Request) (response string, err error) {
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("unexpected return code %d (%s)", resp.StatusCode, resp.Status)
	}

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read response body %v", err)
	}

	return string(responseBytes), nil
}

func (v *virtHandlerConn) Put(url string, body io.ReadCloser) error {
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return err
	}

	_, err = v.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func (v *virtHandlerConn) Get(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Accept", "application/json")
	response, err := v.doRequest(req)
	if err != nil {
		return "", err
	}

	return response, nil
}

func (v *virtHandlerConn) GuestInfoURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(guestInfoTemplateURI, vmi)
}

func formatIpForUri(ip string) string {
	if netutils.IsIPv6String(ip) {
		return "[" + ip + "]"
	}
	return ip
}

func (v *virtHandlerConn) UserListURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(userListTemplateURI, vmi)
}

func (v *virtHandlerConn) FilesystemListURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(filesystemListTemplateURI, vmi)
}

func (v *virtHandlerConn) SEVFetchCertChainURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(sevFetchCertChainTemplateURI, vmi)
}

func (v *virtHandlerConn) SEVQueryLaunchMeasurementURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(sevQueryLaunchMeasurementTemplateURI, vmi)
}

func (v *virtHandlerConn) SEVInjectLaunchSecretURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	return v.formatURI(sevInjectLaunchSecretTemplateURI, vmi)
}
