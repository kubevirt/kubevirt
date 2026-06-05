package disk

import (
	"encoding/json"
	"fmt"
	"io"
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
	args := []string{"info", imagePath, "--output", "json"}
	cmd := exec.Command(QEMUIMGPath, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr for qemu-img command: %v", err)
	}
	out, err := cmd.Output()
	if err != nil {
		errout, _ := io.ReadAll(stderr)
		return nil, fmt.Errorf("failed to invoke qemu-img: %v: %s", err, errout)
	}
	info := &DiskInfo{}
	err = json.Unmarshal(out, info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %v", err)
	}
	return info, err
}
