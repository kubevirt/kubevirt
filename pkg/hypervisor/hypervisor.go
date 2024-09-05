package hypervisor

// Define Hypervisor interface
type Hypervisor interface {
	// The `ps` RSS for virt-launcher-monitor
	GetVirtLauncherMonitorOverhead() string
	// The `ps` RSS for the virt-launcher process
	GetVirtLauncherOverhead() string
	// The `ps` RSS for virtlogd
	GetVirtlogdOverhead() string
	// The `ps` RSS for hypervisor daemon, e.g., virtqemud or libvirtd
	GetHypervisorDaemonOverhead() string
	// The `ps` RSS for vmm, minus the RAM of its (stressed) guest, minus the virtual page table
	GetHypervisorOverhead() string

	SupportsIso() bool

	// TODO Probably not needed
	SupportsNonRootUser() bool

	GetHypervisorDevice() string

	ShouldRunPrivileged() bool
}

// Define QemuHypervisor struct that implements the Hypervisor interface
type QemuHypervisor struct {
}

type CloudHypervisor struct {
}

// Implement ShouldRunPrivileged method for QemuHypervisor
func (q *QemuHypervisor) ShouldRunPrivileged() bool {
	return false
}

// Implement GetHypervisorDevice method for QemuHypervisor
func (q *QemuHypervisor) GetHypervisorDevice() string {
	return "devices.kubevirt.io/kvm"
}

// Implement GetVirtLauncherMonitorOverhead method for QemuHypervisor
func (q *QemuHypervisor) GetVirtLauncherMonitorOverhead() string {
	return "25Mi"
}

// Implement GetVirtLauncherOverhead method for QemuHypervisor
func (q *QemuHypervisor) GetVirtLauncherOverhead() string {
	return "100Mi"
}

// Implement GetVirtlogdOverhead method for QemuHypervisor
func (q *QemuHypervisor) GetVirtlogdOverhead() string {
	return "20Mi"
}

// Implement GetHypervisorDaemonOverhead method for QemuHypervisor
func (q *QemuHypervisor) GetHypervisorDaemonOverhead() string {
	return "35Mi"
}

// Implement GetHypervisorOverhead method for QemuHypervisor
func (q *QemuHypervisor) GetHypervisorOverhead() string {
	return "30Mi"
}

func (q *QemuHypervisor) SupportsIso() bool {
	return true
}

func (q *QemuHypervisor) SupportsNonRootUser() bool {
	return true
}

// Implement ShouldRunPrivileged method for CloudHypervisor
func (c *CloudHypervisor) ShouldRunPrivileged() bool {
	return true
}

// Implement GetHypervisorDevice method for CloudHypervisor
func (c *CloudHypervisor) GetHypervisorDevice() string {
	return "devices.kubevirt.io/mshv"
}

// Implement GetVirtLauncherMonitorOverhead method for CloudHypervisor
func (c *CloudHypervisor) GetVirtLauncherMonitorOverhead() string {
	return "25Mi"
}

// Implement GetVirtLauncherOverhead method for CloudHypervisor
func (c *CloudHypervisor) GetVirtLauncherOverhead() string {
	return "100Mi"
}

// Implement GetVirtlogdOverhead method for CloudHypervisor
func (c *CloudHypervisor) GetVirtlogdOverhead() string {
	return "20Mi"
}

// Implement GetHypervisorDaemonOverhead method for CloudHypervisor
func (c *CloudHypervisor) GetHypervisorDaemonOverhead() string {
	return "35Mi"
}

// Implement GetHypervisorOverhead method for CloudHypervisor
func (c *CloudHypervisor) GetHypervisorOverhead() string {
	return "30Mi"
}

func (c *CloudHypervisor) SupportsIso() bool {
	return false
}

func (c *CloudHypervisor) SupportsNonRootUser() bool {
	return false
}

func NewHypervisor(hypervisor string) Hypervisor {
	if hypervisor == "qemu" {
		return &QemuHypervisor{}
	} else if hypervisor == "ch" {
		return &CloudHypervisor{}
	} else {
		return nil
	}
}
