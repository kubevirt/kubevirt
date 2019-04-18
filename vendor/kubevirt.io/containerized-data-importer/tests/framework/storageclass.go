package framework

import (
	k8sv1 "k8s.io/api/storage/v1"
	"kubevirt.io/containerized-data-importer/tests/utils"
)

// CreateStorageClassFromDefinition is a wrapper around utils.CreateStorageClassFromDefinition
func (f *Framework) CreateStorageClassFromDefinition(def *k8sv1.StorageClass) (*k8sv1.StorageClass, error) {
	return utils.CreateStorageClassFromDefinition(f.K8sClient, def)
}
