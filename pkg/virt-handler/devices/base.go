package devices

import (
	"fmt"
	"os"
	"path"

	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

const (
	cgroupPrefix     = "/proc/1/root/sys/fs/cgroup/devices"
	allowDevicesFile = "devices.allow"
)

type Device interface {
	// Right now, including bridge/veth, only setup is needed, since veths are deleted if namespaces where they are part of are deleted
	Setup(vmi *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error

	// Available can be used to test if kernel modules, bridges, ... are present.
	// If nil is returned it is assumed that under normal conditions setting up the container would work
	Available() error
}

func whitelistDevice(dev *KernelDevice, acl string, slice string) error {
	f, err := os.OpenFile(path.Join(cgroupPrefix, slice, allowDevicesFile), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(fmt.Sprintf("%s %d:%d %s\n", dev.Type, dev.Major, dev.Minor, acl)); err != nil {
		return err
	}
	return nil
}

type KernelDevice struct {
	// Major represent the major device number
	Major int64
	// Minor represents the minor device number
	Minor int64
	// Represents the device type
	Type string
}

func (k *KernelDevice) MKDev() int {
	return int(uint32(unix.Mkdev(uint32(k.Major), uint32(k.Minor))))
}

func NewKernelDevice(deviceType string, major int64, minor int64) *KernelDevice {
	return &KernelDevice{
		Type:  deviceType,
		Major: major,
		Minor: minor,
	}
}
