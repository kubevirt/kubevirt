package disk

import (
	"bytes"
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
	// #nosec No risk for attacker injection. Only get information about an image
	// Use -U (--force-share) to allow reading image metadata while the qcow2
	// image is in use by a running VM, avoiding exclusive locks during probing.
	args := []string{"info", "-U", imagePath, "--output", "json"}
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
