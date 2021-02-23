package main

import (
	"errors"

	v1 "kubevirt.io/client-go/api/v1"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

// FakeDomainManager for testing cmdclients
type FakeDomainManager struct{}

// Exec implementing DomainManager
func (f FakeDomainManager) Exec(domainName string, command string, args []string) (string, error) {
	if domainName == "error" {
		return "", errors.New("fake error")
	}
	if domainName == "fail" {
		return "command failed", agent.ExecExitCode{ExitCode: 1}
	}
	return "success", nil
}

// SyncVMI implementing DomainManager
func (f FakeDomainManager) SyncVMI(_ *v1.VirtualMachineInstance, _ bool, _ *cmdv1.VirtualMachineOptions) (*api.DomainSpec, error) {
	return nil, errors.New("not implemented")
}

// PauseVMI implementing DomainManager
func (f FakeDomainManager) PauseVMI(_ *v1.VirtualMachineInstance) error {
	return errors.New("not implemented")
}

// UnpauseVMI implementing DomainManager
func (f FakeDomainManager) UnpauseVMI(_ *v1.VirtualMachineInstance) error {
	return errors.New("not implemented")
}

// KillVMI implementing DomainManager
func (f FakeDomainManager) KillVMI(_ *v1.VirtualMachineInstance) error {
	return errors.New("not implemented")
}

// DeleteVMI implementing DomainManager
func (f FakeDomainManager) DeleteVMI(_ *v1.VirtualMachineInstance) error {
	return errors.New("not implemented")
}

// SignalShutdownVMI implementing DomainManager
func (f FakeDomainManager) SignalShutdownVMI(_ *v1.VirtualMachineInstance) error {
	return errors.New("not implemented")
}

// MarkGracefulShutdownVMI implementing DomainManager
func (f FakeDomainManager) MarkGracefulShutdownVMI(_ *v1.VirtualMachineInstance) error {
	return errors.New("not implemented")
}

// ListAllDomains implementing DomainManager
func (f FakeDomainManager) ListAllDomains() ([]*api.Domain, error) {
	return nil, errors.New("not implemented")
}

// MigrateVMI implementing DomainManager
func (f FakeDomainManager) MigrateVMI(_ *v1.VirtualMachineInstance, _ *cmdclient.MigrationOptions) error {
	return errors.New("not implemented")
}

// PrepareMigrationTarget implementing DomainManager
func (f FakeDomainManager) PrepareMigrationTarget(_ *v1.VirtualMachineInstance, _ bool) error {
	return errors.New("not implemented")
}

// GetDomainStats implementing DomainManager
func (f FakeDomainManager) GetDomainStats() ([]*stats.DomainStats, error) {
	return nil, errors.New("not implemented")
}

// CancelVMIMigration implementing DomainManager
func (f FakeDomainManager) CancelVMIMigration(_ *v1.VirtualMachineInstance) error {
	return errors.New("not implemented")
}

// GetGuestInfo implementing DomainManager
func (f FakeDomainManager) GetGuestInfo() (v1.VirtualMachineInstanceGuestAgentInfo, error) {
	return v1.VirtualMachineInstanceGuestAgentInfo{}, errors.New("not implemented")
}

// GetUsers implementing DomainManager
func (f FakeDomainManager) GetUsers() ([]v1.VirtualMachineInstanceGuestOSUser, error) {
	return nil, errors.New("not implemented")
}

// GetFilesystems implementing DomainManager
func (f FakeDomainManager) GetFilesystems() ([]v1.VirtualMachineInstanceFileSystem, error) {
	return nil, errors.New("not implemented")
}

// SetGuestTime implementing DomainManager
func (f FakeDomainManager) SetGuestTime(_ *v1.VirtualMachineInstance) error {
	return errors.New("not implemented")
}
