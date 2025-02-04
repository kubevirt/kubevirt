package containerdisk

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/os/disk"
)

const (
	DiskSourceFallbackPath = "/disk"
)

func VerifyQCOW2(diskInfo *disk.DiskInfo) error {
	if diskInfo.Format != "qcow2" {
		return fmt.Errorf("expected a disk format of qcow2, but got '%v'", diskInfo.Format)
	}

	if diskInfo.BackingFile != "" {
		return fmt.Errorf("expected no backing file, but found %v", diskInfo.BackingFile)
	}
	return nil
}

func VerifyImage(diskInfo *disk.DiskInfo) error {
	switch diskInfo.Format {
	case "qcow2":
		return VerifyQCOW2(diskInfo)
	case "raw":
		return nil
	default:
		return fmt.Errorf("unsupported image format: %v", diskInfo.Format)
	}
}
