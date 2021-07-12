package containerdisk

import (
	"fmt"
)

const (
	DiskSourceFallbackPath = "/disk"
)

type DiskInfo struct {
	Format      string `json:"format"`
	BackingFile string `json:"backing-filename"`
	ActualSize  int64  `json:"actual-size"`
	VirtualSize int64  `json:"virtual-size"`
}

func VerifyQCOW2(diskInfo *DiskInfo) error {
	if diskInfo.Format != "qcow2" {
		return fmt.Errorf("expected a disk format of qcow2, but got '%v'", diskInfo.Format)
	}

	if diskInfo.BackingFile != "" {
		return fmt.Errorf("expected no backing file, but found %v", diskInfo.BackingFile)
	}
	return nil
}

func VerifyImage(diskInfo *DiskInfo) error {
	switch diskInfo.Format {
	case "qcow2":
		return VerifyQCOW2(diskInfo)
	case "raw":
		return nil
	default:
		return fmt.Errorf("unsupported image format: %v", diskInfo.Format)
	}
}
