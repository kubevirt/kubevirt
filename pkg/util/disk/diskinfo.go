package disk

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

const (
	DiskSourceFallbackPath = "/disk"
	qemuImg                = "/usr/bin/qemu-img"
)

type DiskInfo struct {
	Format      string `json:"format"`
	BackingFile string `json:"backing-filename"`
	ActualSize  int64  `json:"actual-size"`
	VirtualSize int64  `json:"virtual-size"`
}

func (d *DiskInfo) Clone() *DiskInfo {
	out := *d
	return &out
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

// FetchDiskInfo retrieves information about a disk image by invoking the `qemu-img info` command.
//
// The function will fail with an error in the following cases:
// - The disk image is currently in use (e.g., mounted or locked).
// - The `qemu-img` command fails to execute or returns an error.
// - The output from `qemu-img` cannot be parsed as JSON.
func FetchDiskInfo(path string) (*DiskInfo, error) {
	cmd := exec.Command(qemuImg, "info", path, "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		var e *exec.ExitError
		if errors.As(err, &e) {
			if len(e.Stderr) > 0 {
				return nil, fmt.Errorf("failed to perform qemu-img: %w: %q", err, string(e.Stderr))
			}
		}
		return nil, fmt.Errorf("failed to perform qemu-img: %w", err)
	}
	info := &DiskInfo{}
	if err = json.Unmarshal(out, info); err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %w", err)
	}
	return info, nil
}
