package disk

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

type DiskInfo struct {
	Format         string          `json:"format"`
	BackingFile    string          `json:"backing-filename"`
	ActualSize     int64           `json:"actual-size"`
	VirtualSize    int64           `json:"virtual-size"`
	FormatSpecific *FormatSpecific `json:"format-specific,omitempty"`
}

type FormatSpecific struct {
	Type string              `json:"type"`
	Data *FormatSpecificData `json:"data,omitempty"`
}

type FormatSpecificData struct {
	Bitmaps []BitmapInfo `json:"bitmaps,omitempty"`
}

type BitmapInfo struct {
	Name        string   `json:"name"`
	Granularity int64    `json:"granularity"`
	Flags       []string `json:"flags,omitempty"`
}

const (
	QEMUIMGPath = "/usr/bin/qemu-img"
)

func (d *DiskInfo) HasBitmap(name string) bool {
	if d.FormatSpecific == nil || d.FormatSpecific.Data == nil {
		return false
	}
	for _, bm := range d.FormatSpecific.Data.Bitmaps {
		if bm.Name == name {
			return true
		}
	}
	return false
}

func GetDiskInfo(imagePath string) (*DiskInfo, error) {
	return getDiskInfo(imagePath, false)
}

func GetDiskInfoWithForceShare(imagePath string) (*DiskInfo, error) {
	return getDiskInfo(imagePath, true)
}

func getDiskInfo(imagePath string, forceShare bool) (*DiskInfo, error) {
	// #nosec No risk for attacker injection. Only get information about an image
	args := []string{"info", imagePath, "--output", "json"}
	if forceShare {
		// -U (force-share) allows reading disk info even when the disk is in use by QEMU
		args = []string{"info", "-U", imagePath, "--output", "json"}
	}
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
