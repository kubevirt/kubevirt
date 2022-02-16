package fake

import (
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type MockEphemeralDiskImageCreator struct {
	BaseDir string
}

func (m *MockEphemeralDiskImageCreator) CreateBackedImageForVolume(_ v1.Volume, _ string, _ string) error {
	return nil
}

func (m *MockEphemeralDiskImageCreator) CreateEphemeralImages(_ *v1.VirtualMachineInstance, _ *api.Domain) error {
	return nil
}

func (m *MockEphemeralDiskImageCreator) GetFilePath(volumeName string) string {
	return filepath.Join(m.BaseDir, volumeName, "disk.qcow2")
}

func (m *MockEphemeralDiskImageCreator) Init() error {
	return nil
}
