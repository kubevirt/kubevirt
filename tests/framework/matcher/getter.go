package matcher

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
)

// ThisPod fetches the latest state of the pod. If the object does not exist, nil is returned.
func ThisPod(pod *v1.Pod) func() (*v1.Pod, error) {
	return func() (p *v1.Pod, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}

// ThisVMI fetches the latest state of the pod. If the object does not exist, nil is returned.
func ThisVMI(vmi *virtv1.VirtualMachineInstance) func() (*virtv1.VirtualMachineInstance, error) {
	return func() (p *virtv1.VirtualMachineInstance, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}

// AllVMI fetches the latest state of the pod. If the object does not exist, nil is returned.
func AllVMIs(namespace string) func() ([]virtv1.VirtualMachineInstance, error) {
	return func() (p []virtv1.VirtualMachineInstance, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		list, err := virtClient.VirtualMachineInstance(namespace).List(&k8smetav1.ListOptions{})
		return list.Items, err
	}
}

// ThisDV fetches the latest state of the pod. If the object does not exist, nil is returned.
func ThisDV(dv *v1beta1.DataVolume) func() (*v1beta1.DataVolume, error) {
	return func() (p *v1beta1.DataVolume, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Get(context.Background(), dv.Name, k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}
