package hypervisor

type KVMHypervisor struct{}

func (k *KVMHypervisor) GetK8sResourceName() string {
	return "devices.kubevirt.io/kvm"
}
