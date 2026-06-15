package disk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

const (
	DiskSourceFallbackPath = "/disk"
)

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

func GetDiskInfoWithValidation(imagePath string, diskMemoryLimitBytes int64) (*DiskInfo, error) {
	// #nosec No risk for attacker injection. Only get information about an image
	args := []string{"info", imagePath, "--output", "json"}
	cmd := exec.Command(QEMUIMGPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to invoke qemu-img: %v: %s", err, stderr.String())
	}
	info := &DiskInfo{}
	err = json.Unmarshal(out, info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %v", err)
	}
	return info, err
}
