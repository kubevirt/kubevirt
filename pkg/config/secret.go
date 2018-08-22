package config

import (
	"path/filepath"
)

// GetSecretDiskPath returns a path to Secret iso image created based on volume name
func GetSecretDiskPath(volumeName string) string {
	return filepath.Join(SecretDisksDir, volumeName+".iso")
}
