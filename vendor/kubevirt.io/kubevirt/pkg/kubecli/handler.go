package kubecli

import (
	"fmt"
	"net/url"

	v1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
)

func NewVirtHandlerClient(client KubevirtClient) VirtHandlerClient {
	return &virtHandler{client}
}

type VirtHandlerClient interface {
	ForNode(nodeName string) VirtHandlerConn
}

type VirtHandlerConn interface {
	ConnectionDetails() (ip string, port string, err error)
	ConsoleURI(vmi *virtv1.VirtualMachineInstance) (*url.URL, error)
	Pod() (pod *v1.Pod, err error)
}

type virtHandler struct {
	client KubevirtClient
}

type virtHandlerConn struct {
	client KubevirtClient
	pod    *v1.Pod
	err    error
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
	pods, err := v.client.CoreV1().Pods(v1.NamespaceAll).List(
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
	return
}

//TODO move the actual ws handling in here, and work with channels
func (v *virtHandlerConn) ConsoleURI(vmi *virtv1.VirtualMachineInstance) (*url.URL, error) {
	ip, port, err := v.ConnectionDetails()
	if err != nil {
		return nil, err
	}
	return &url.URL{
		Path: fmt.Sprintf("/api/v1/namespaces/%s/virtualmachineinstances/%s/console", vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name),
		Host: ip + ":" + port,
	}, nil
}

func (v *virtHandlerConn) Pod() (pod *v1.Pod, err error) {
	if v.err != nil {
		err = v.err
		return
	}
	return v.pod, err
}
