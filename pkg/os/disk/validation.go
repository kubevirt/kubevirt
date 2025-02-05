package disk

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"kubevirt.io/client-go/log"
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
	cmd := exec.Command("bash", "-c", fmt.Sprintf("ulimit -t %d && ulimit -v %d && %v info %v --output json", 10, diskMemoryLimitBytes/1024, QEMUIMGPath, imagePath))
	log.Log.V(3).Infof("fetching image info. running command: %s", cmd.String())
	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if len(e.Stderr) > 0 {
				return nil, fmt.Errorf("failed to invoke qemu-img: %v: '%v'", err, string(e.Stderr))
			}
		}
		return nil, fmt.Errorf("failed to invoke qemu-img: %v", err)
	}
	info := &DiskInfo{}
	err = json.Unmarshal(out, info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %v", err)
	}
	return info, err
}
