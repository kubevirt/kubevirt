package domain

import (
	"fmt"

	vmschema "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	// PluginName must match KubeVirt CR binding registration and VMI interface binding.name.
	PluginName = "bpfbridge"
)

type NetworkConfigurator struct {
	vmiSpecIface *vmschema.Interface
	tapDevName   string
}

func NewNetworkConfigurator(ifaces []vmschema.Interface, networks []vmschema.Network, tapDevName string) (*NetworkConfigurator, error) {
	if tapDevName == "" {
		return nil, fmt.Errorf("tap device name is empty")
	}
	var bound *vmschema.Interface
	for i := range ifaces {
		if ifaces[i].Binding != nil && ifaces[i].Binding.Name == PluginName {
			bound = &ifaces[i]
			break
		}
	}
	if bound == nil {
		return nil, fmt.Errorf("no interface uses binding %q", PluginName)
	}
	net := vmispec.LookupNetworkByName(networks, bound.Name)
	if net == nil {
		return nil, fmt.Errorf("network %q not found for bpf-bridge interface", bound.Name)
	}

	return &NetworkConfigurator{vmiSpecIface: bound, tapDevName: tapDevName}, nil
}

func (c NetworkConfigurator) Mutate(domainSpec *domainschema.DomainSpec) (*domainschema.DomainSpec, error) {
	domainSpecCopy := domainSpec.DeepCopy()
	iface := lookupIfaceByAliasName(domainSpecCopy.Devices.Interfaces, c.vmiSpecIface.Name)
	if iface == nil {
		return nil, fmt.Errorf("domain has no interface with alias %q", c.vmiSpecIface.Name)
	}
	if iface.Type != "ethernet" {
		return nil, fmt.Errorf("interface %q: expected type ethernet for tap attachment, got %q", c.vmiSpecIface.Name, iface.Type)
	}
	iface.Target = &domainschema.InterfaceTarget{Device: c.tapDevName}
	return domainSpecCopy, nil
}

func lookupIfaceByAliasName(ifaces []domainschema.Interface, name string) *domainschema.Interface {
	for i := range ifaces {
		if ifaces[i].Alias != nil && ifaces[i].Alias.GetName() == name {
			return &ifaces[i]
		}
	}
	return nil
}
