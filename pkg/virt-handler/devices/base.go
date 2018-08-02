package devices

import (
	"golang.org/x/sys/unix"

	"fmt"
	"os"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

type Device interface {
	// Right now, including bridge/veth, only setup is needed, since veths are deleted if namespaces where they are part of are deleted
	Setup(vmi *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error
}

func whitelistDevice(deviceType string, dev *KernelDevice, acl string, slice string) error {
	f, err := os.OpenFile("/proc/1/root/sys/fs/cgroup/devices"+slice+"/devices.allow", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(fmt.Sprintf("%s %d:%d %s\n", deviceType, dev.Major, dev.Minor, acl)); err != nil {
		return err
	}
	return nil
}

type KernelDevice struct {
	// Major represent the major device number
	Major int64
	// Minor represents the minor device number
	Minor int64
}

func (k *KernelDevice) MKDev() int {
	return int(uint32(unix.Mkdev(uint32(k.Major), uint32(k.Minor))))
}

func NewKernelDevice(major int64, minor int64) *KernelDevice {
	return &KernelDevice{
		Major: major,
		Minor: minor,
	}
}
