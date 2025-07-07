package export

import (
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// VolumeInfo contains paths for a volume
type VolumeInfo struct {
	Path       string
	ArchiveURI string
	DirURI     string
	RawURI     string
	RawGzURI   string
}

// ServerPaths contains static paths and per-volume paths
type ServerPaths struct {
	VMURI     string
	SecretURI string
	Volumes   []VolumeInfo
}

// EnvironToMap converts the environment variables to a map
func EnvironToMap() map[string]string {
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		envMap[kv[0]] = kv[1]
	}
	return envMap
}

// ContainerEnvToMap converts the container environment variables to a map
func ContainerEnvToMap(env []corev1.EnvVar) map[string]string {
	envMap := make(map[string]string)
	for _, e := range env {
		envMap[e.Name] = e.Value
	}
	return envMap
}

// CreateServerPaths creates a ServerPaths object from the environment variables
func CreateServerPaths(env map[string]string) *ServerPaths {
	result := &ServerPaths{
		VMURI:     env["EXPORT_VM_DEF_URI"],
		SecretURI: env["EXPORT_SECRET_DEF_URI"],
	}
	for k, v := range env {
		if strings.HasSuffix(k, "_EXPORT_PATH") {
			envPrefix := strings.TrimSuffix(k, "_EXPORT_PATH")
			vi := VolumeInfo{
				Path:       v,
				ArchiveURI: env[envPrefix+"_EXPORT_ARCHIVE_URI"],
				DirURI:     env[envPrefix+"_EXPORT_DIR_URI"],
				RawURI:     env[envPrefix+"_EXPORT_RAW_URI"],
				RawGzURI:   env[envPrefix+"_EXPORT_RAW_GZIP_URI"],
			}
			result.Volumes = append(result.Volumes, vi)
		}
	}
	return result
}

// GetVolumeInfo returns the VolumeInfo for a given PVC name
func (sp *ServerPaths) GetVolumeInfo(pvcName string) *VolumeInfo {
	for _, v := range sp.Volumes {
		_, n := filepath.Split(filepath.Clean(v.Path))
		if n == pvcName {
			return &v
		}
	}
	return nil
}
