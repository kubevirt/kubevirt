package ephemeraldisk

import (
	"fmt"
	"path/filepath"
)

const (
	mountBaseDir = "/var/run/libvirt/ephemeraldisk"
)

func generateBaseDir() string {
	return fmt.Sprintf("%s", mountBaseDir)
}
func generateVolumeMountDir(volumeName string) string {
	baseDir := generateBaseDir()
	return filepath.Join(baseDir, volumeName)
}

func GetFilePath(volumeName string) string {
	volumeMountDir := generateVolumeMountDir(volumeName)
	return filepath.Join(volumeMountDir, "disk.qcow2")
}
