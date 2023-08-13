package infraconfigurators

import (
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type PasstPodNetworkConfigurator struct {
	handler netdriver.NetworkHandler
}

func NewPasstPodNetworkConfigurator(handler netdriver.NetworkHandler) *PasstPodNetworkConfigurator {
	return &PasstPodNetworkConfigurator{
		handler: handler,
	}
}

func (b *PasstPodNetworkConfigurator) DiscoverPodNetworkInterface(_ string) error {
	return nil
}

func (b *PasstPodNetworkConfigurator) GenerateNonRecoverableDHCPConfig() *cache.DHCPConfig {
	return nil
}

func (b *PasstPodNetworkConfigurator) GenerateNonRecoverableDomainIfaceSpec() *api.Interface {
	return nil
}
