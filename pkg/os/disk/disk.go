package disk

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type DiskInfo struct {
	Format      string `json:"format"`
	BackingFile string `json:"backing-filename"`
	ActualSize  int64  `json:"actual-size"`
	VirtualSize int64  `json:"virtual-size"`
}

const (
	QEMUIMGPath = "/usr/bin/qemu-img"
)

func GetDiskInfo(imagePath string) (*DiskInfo, error) {
	// #nosec No risk for attacket injection. Only get information about an image
	args := []string{"info", imagePath, "--output", "json"}
	out, err := exec.Command(QEMUIMGPath, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to invoke qemu-img: %v", err)
	}
	info := &DiskInfo{}
	err = json.Unmarshal(out, info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %v", err)
	}
	return info, err
}
