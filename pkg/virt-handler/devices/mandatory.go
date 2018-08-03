package devices

import (
	"os"
	"path/filepath"
	"syscall"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

const (
	kvm = "/dev/kvm"
	tun = "/dev/net/tun"
)

var kvmDevice = NewKernelDevice(10, 232)
var tunDevice = NewKernelDevice(10, 200)

type KVM struct {
}

type TUN struct {
}

// Setup creates /dev/kvm inside a container and sets the right permissions for qemu
func (k *KVM) Setup(_ *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error {

	devicePath := podNamespaces.MountRoot() + kvm

	stat := syscall.Stat_t{}
	err := syscall.Stat(devicePath, &stat)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Create the device if necessary
	if os.IsNotExist(err) {

		// Allow the container accessing the device
		err = whitelistDevice("c", kvmDevice, "rw", podNamespaces.Slice())
		if err != nil {
			return err
		}
		err = syscall.Mknod(devicePath, syscall.S_IFCHR, kvmDevice.MKDev())
		if err != nil {
			return err
		}
	}

	// Set group to qemu if necessary
	if stat.Gid != 107 {
		err = os.Chown(devicePath, int(stat.Uid), 107)
		if err != nil {
			return err
		}
	}
	// Set permissions to 0666 if necessary
	if stat.Mode != 0666 {
		err = os.Chmod(devicePath, 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

// Setup creates /dev/net/tun inside the container to all tun/tap based operations on VMIs
func (t *TUN) Setup(_ *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error {

	devicePath := podNamespaces.MountRoot() + tun

	stat := syscall.Stat_t{}
	err := syscall.Stat(devicePath, &stat)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Create the device if necessary
	if os.IsNotExist(err) {
		// Allow the container accessing the device
		err = whitelistDevice("c", tunDevice, "rw", podNamespaces.Slice())
		if err != nil {
			return err
		}
		// Create /dev/net if necessary
		err = os.MkdirAll(filepath.Dir(devicePath), 0755)
		if err != nil {
			return err
		}

		err = syscall.Mknod(devicePath, syscall.S_IFCHR, tunDevice.MKDev())
		if err != nil {
			return err
		}
	}
	// Set permissions to 0660 if necessary
	if stat.Mode != 0660 {
		err = os.Chmod(devicePath, 0660)
		if err != nil {
			return err
		}
	}
	return nil
}
