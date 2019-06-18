package lookup

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

func VirtualMachinesOnNode(cli kubecli.KubevirtClient, nodeName string) ([]*virtv1.VirtualMachineInstance, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", virtv1.NodeNameLabel, nodeName))
	if err != nil {
		return nil, err
	}
	list, err := cli.VirtualMachineInstance(v1.NamespaceAll).List(&metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	if err != nil {
		return nil, err
	}

	vmis := []*virtv1.VirtualMachineInstance{}

	for i := range list.Items {
		vmis = append(vmis, &list.Items[i])
	}
	return vmis, nil
}
