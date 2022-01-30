package matcher

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

// ThisPod fetches the latest state of the pod. If the object does not exist, nil is returned.
func ThisPod(pod *v1.Pod) func() (*v1.Pod, error) {
	return ThisPodWith(pod.Namespace, pod.Name)
}

// ThisPodWith fetches the latest state of the pod based on namespace and name. If the object does not exist, nil is returned.
func ThisPodWith(namespace string, name string) func() (*v1.Pod, error) {
	return func() (p *v1.Pod, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.CoreV1().Pods(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}

// ThisVMI fetches the latest state of the VirtualMachineInstance. If the object does not exist, nil is returned.
func ThisVMI(vmi *virtv1.VirtualMachineInstance) func() (*virtv1.VirtualMachineInstance, error) {
	return ThisVMIWith(vmi.Namespace, vmi.Name)
}

// ThisVMIWith fetches the latest state of the VirtualMachineInstance based on namespace and name. If the object does not exist, nil is returned.
func ThisVMIWith(namespace string, name string) func() (*virtv1.VirtualMachineInstance, error) {
	return func() (p *virtv1.VirtualMachineInstance, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}

// ThisVM fetches the latest state of the VirtualMachine. If the object does not exist, nil is returned.
func ThisVM(vm *virtv1.VirtualMachine) func() (*virtv1.VirtualMachine, error) {
	return ThisVMWith(vm.Namespace, vm.Name)
}

// ThisVMWith fetches the latest state of the VirtualMachine based on namespace and name. If the object does not exist, nil is returned.
func ThisVMWith(namespace string, name string) func() (*virtv1.VirtualMachine, error) {
	return func() (p *virtv1.VirtualMachine, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.VirtualMachine(namespace).Get(name, &k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}

// AllVMI fetches the latest state of all VMIs in a namespace.
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

// ThisDV fetches the latest state of the DataVolume. If the object does not exist, nil is returned.
func ThisDV(dv *v1beta1.DataVolume) func() (*v1beta1.DataVolume, error) {
	return ThisDVWith(dv.Namespace, dv.Name)
}

// ThisDVWith fetches the latest state of the DataVolume based on namespace and name. If the object does not exist, nil is returned.
func ThisDVWith(namespace string, name string) func() (*v1beta1.DataVolume, error) {
	return func() (p *v1beta1.DataVolume, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}

// ThisPVC fetches the latest state of the PVC. If the object does not exist, nil is returned.
func ThisPVC(pvc *v1.PersistentVolumeClaim) func() (*v1.PersistentVolumeClaim, error) {
	return ThisPVCWith(pvc.Namespace, pvc.Name)
}

// ThisPVCWith fetches the latest state of the PersistentVolumeClaim based on namespace and name. If the object does not exist, nil is returned.
func ThisPVCWith(namespace string, name string) func() (*v1.PersistentVolumeClaim, error) {
	return func() (p *v1.PersistentVolumeClaim, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}

// ThisMigration fetches the latest state of the Migration. If the object does not exist, nil is returned.
func ThisMigration(migration *virtv1.VirtualMachineInstanceMigration) func() (*virtv1.VirtualMachineInstanceMigration, error) {
	return ThisMigrationWith(migration.Namespace, migration.Name)
}

// ThisMigrationWith fetches the latest state of the Migration based on namespace and name. If the object does not exist, nil is returned.
func ThisMigrationWith(namespace string, name string) func() (*virtv1.VirtualMachineInstanceMigration, error) {
	return func() (p *virtv1.VirtualMachineInstanceMigration, err error) {
		virtClient, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		p, err = virtClient.VirtualMachineInstanceMigration(namespace).Get(name, &k8smetav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return
	}
}
