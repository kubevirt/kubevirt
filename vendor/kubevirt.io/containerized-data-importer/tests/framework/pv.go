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

// ClearBlockPV resets the device to the initial state and wipes any junk left on it.
func (f *Framework) ClearBlockPV() error {
	pod, err := utils.FindPodByPrefix(f.K8sClient, f.CdiInstallNs, "cdi-block-device", "name=cdi-block-device")
	if err != nil {
		return err
	}
	_, err = f.ExecShellInPod(pod.Name, f.CdiInstallNs, "truncate --size 0 loop0")
	if err != nil {
		return err
	}
	_, err = f.ExecShellInPod(pod.Name, f.CdiInstallNs, "truncate --size 524288000 loop0")
	if err != nil {
		return err
	}
	_, err = f.ExecShellInPod(pod.Name, f.CdiInstallNs, "truncate --size 0 loop1")
	if err != nil {
		return err
	}
	_, err = f.ExecShellInPod(pod.Name, f.CdiInstallNs, "truncate --size 524288000 loop1")
	if err != nil {
		return err
	}
	return nil
}
