package framework

import (
	k8sv1 "k8s.io/api/core/v1"
	"kubevirt.io/containerized-data-importer/tests/utils"
	"time"
)

// CreatePVFromDefinition is a wrapper around utils.CreatePVFromDefinition
func (f *Framework) CreatePVFromDefinition(def *k8sv1.PersistentVolume) (*k8sv1.PersistentVolume, error) {
	return utils.CreatePVFromDefinition(f.K8sClient, def)
}

// WaitTimeoutForPVReady is a wrapper around utils.WaitTimeouotForPVReady
func (f *Framework) WaitTimeoutForPVReady(pvName string, timeout time.Duration) error {
	return utils.WaitTimeoutForPVReady(f.K8sClient, pvName, timeout)
}
