package hypervisor

type MshvL1vhHypervisor struct{}

func (m *MshvL1vhHypervisor) GetK8sResourceName() string {
	return "devices.kubevirt.io/mshv"
}
