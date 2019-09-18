package kubecli

import (
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/util"
)

const (
	consoleTemplateURI = "wss://%s:%s/v1/namespaces/%s/virtualmachineinstances/%s/console"
	vncTemplateURI     = "wss://%s:%s/v1/namespaces/%s/virtualmachineinstances/%s/vnc"
)

func NewVirtHandlerClient(client KubevirtClient) VirtHandlerClient {
	return &virtHandler{client}
}

type VirtHandlerClient interface {
	ForNode(nodeName string) VirtHandlerConn
}

type VirtHandlerConn interface {
	ConnectionDetails() (ip string, port string, err error)
	ConsoleURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	VNCURI(vmi *virtv1.VirtualMachineInstance) (string, error)
	Pod() (pod *v1.Pod, err error)
	SetPort(port int) VirtHandlerConn
}

type virtHandler struct {
	client KubevirtClient
}

type virtHandlerConn struct {
	client KubevirtClient
	pod    *v1.Pod
	err    error
	port   string
}

func (v *virtHandler) ForNode(nodeName string) VirtHandlerConn {
	pod, found, err := v.getVirtHandler(nodeName)
	conn := &virtHandlerConn{}
	if !found {
		conn.err = fmt.Errorf("No virt-handler on node %s found", nodeName)
	}
	if err != nil {
		conn.err = err
	}
	conn.pod = pod
	conn.client = v.client
	return conn
}

func (v *virtHandler) getVirtHandler(nodeName string) (*v1.Pod, bool, error) {

	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + nodeName)
	labelSelector, err := labels.Parse(virtv1.AppLabel + " in (virt-handler)")
	if err != nil {
		return nil, false, err
	}
	ns, err := util.GetNamespace()
	if err != nil {
		return nil, false, err
	}
	pods, err := v.client.CoreV1().Pods(ns).List(
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

func (v *virtHandlerConn) ConnectionDetails() (ip string, port string, err error) {
	if v.err != nil {
		err = v.err
		return
	}
	// TODO depending on in which network namespace virt-handler runs, we might have to choose the NodeIPt d
	ip = v.pod.Status.PodIP
	// TODO get rid of the hardcoded port
	port = "8185"
	if v.port != "" {
		port = v.port
	}
	return
}

//TODO move the actual ws handling in here, and work with channels
func (v *virtHandlerConn) ConsoleURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	ip, port, err := v.ConnectionDetails()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(consoleTemplateURI, ip, port, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name), nil
}

func (v *virtHandlerConn) SetPort(port int) VirtHandlerConn {
	v.port = strconv.Itoa(port)
	return v
}

func (v *virtHandlerConn) VNCURI(vmi *virtv1.VirtualMachineInstance) (string, error) {
	ip, port, err := v.ConnectionDetails()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(vncTemplateURI, ip, port, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name), nil
}

func (v *virtHandlerConn) Pod() (pod *v1.Pod, err error) {
	if v.err != nil {
		err = v.err
		return
	}
	return v.pod, err
}
