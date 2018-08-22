package config

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
)

// Type represents allowed config types like ConfigMap or Secret
type Type string

const (
	// ConfigMap respresents a configmap type,
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/
	ConfigMap Type = "configmap"
	// Secret represents a secret type,
	// https://kubernetes.io/docs/concepts/configuration/secret/
	Secret Type = "secret"

	mountBaseDir = "/var/run/kubevirt-private"

	// ConfigMapSourceDir represents a location where ConfigMap is attached to the pod
	ConfigMapSourceDir = mountBaseDir + "/config-map"
	// SecretSourceDir represents a location where Secrets is attached to the pod
	SecretSourceDir = mountBaseDir + "/secret"

	// ConfigMapDisksDir represents a path to ConfigMap iso images
	ConfigMapDisksDir = mountBaseDir + "/config-map-disks"
	// SecretDisksDir represents a path to Secrets iso images
	SecretDisksDir = mountBaseDir + "/secret-disks"
)

func getFilesLayout(dirPath string, volumeName string) ([]string, error) {
	var filesPath []string
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fileName := file.Name()
		filesPath = append(filesPath, fileName+"="+filepath.Join(dirPath, fileName))
	}
	return filesPath, nil
}

func createIsoConfigImage(output string, files []string) error {
	var args []string
	args = append(args, "-output")
	args = append(args, output)
	args = append(args, "-volid")
	args = append(args, "cfgdata")
	args = append(args, "-joliet")
	args = append(args, "-rock")
	args = append(args, "-graft-points")
	args = append(args, files...)

	cmd := exec.Command("genisoimage", args...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return nil
}
