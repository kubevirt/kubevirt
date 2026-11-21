package notifymanager

import (
	"errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

func NewUpgradeAwareManager(old *pipeManager, new *notifyV2Manager) *upgradeAwareManager {
	return &upgradeAwareManager{
		pipeManager:     old,
		notifyV2Manager: new,
	}
}

type upgradeAwareManager struct {
	pipeManager     *pipeManager
	notifyV2Manager *notifyV2Manager
}

func (m *upgradeAwareManager) StartDomainNotify(domainPipeStopChan <-chan struct{}, vmi *v1.VirtualMachineInstance) error {
	err := m.notifyV2Manager.StartDomainNotify(domainPipeStopChan, vmi)
	if errors.Is(err, notImplemented) {
		log.Log.Object(vmi).Info("Launcher doesn't support notify v2")
		return m.pipeManager.StartDomainNotify(domainPipeStopChan, vmi)
	}
	return err
}
