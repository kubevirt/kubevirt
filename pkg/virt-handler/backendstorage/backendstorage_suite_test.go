package backendstorage

import (
	"log"
	"os"
	"path"
	"testing"

	"kubevirt.io/client-go/testutils"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
)

var (
	testTempDir string
)

func TestBackendStorage(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

func prepareFilesystemTestEnv(tempDirPrefix string) (string, error) {
	tempDir, err := os.MkdirTemp(os.TempDir(), tempDirPrefix)
	if err != nil {
		return "", err
	}
	for _, d := range []string{backendstorage.PodVMStatePath, backendstorage.PodNVRAMPath, backendstorage.PodSwtpmPath, backendstorage.PodSwtpmLocalcaPath} {
		if err := os.MkdirAll(path.Join(tempDir, d), os.ModePerm); err != nil {
			return "", err
		}
	}
	if err := os.Mkdir(path.Join(tempDir, backendstorage.PodVMStatePath, "lost+found"), os.ModePerm); err != nil {
		return "", err
	}
	return tempDir, nil
}

func cleanupFilesystemTestEnv(tempDir string) error {
	log.Printf("Removing temporary directory: %s", tempDir)
	return os.RemoveAll(tempDir)
}
