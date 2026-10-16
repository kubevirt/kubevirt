package disk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	out, err := QEMUImg(context.Background(), nil, "info", imagePath, "--output", "json")
	if err != nil {
		return nil, err
	}
	info := &DiskInfo{}
	err = json.Unmarshal(out, info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk info: %v", err)
	}
	return info, err
}

func QEMUImg(ctx context.Context, extraFiles []*os.File, subcmd string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, QEMUIMGPath, append([]string{subcmd}, args...)...)
	cmd.ExtraFiles = extraFiles
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil, fmt.Errorf("qemu-img %s failed: %w: %s", subcmd, err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("qemu-img %s failed: %w", subcmd, err)
	}
	return out, nil
}
