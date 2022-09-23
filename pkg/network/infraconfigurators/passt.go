package infraconfigurators

import (
	"kubevirt.io/client-go/log"

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

func (b *PasstPodNetworkConfigurator) PreparePodNetworkInterface() error {
	log.Log.V(4).Info("Configuring ping group range")
	err := b.handler.ConfigurePingGroupRange()
	if err != nil {
		log.Log.Reason(err).Errorf("failed to configure ping group range")
		return err
	}
	// bugs.passt.top/show_bug.cgi?id=15 and bugs.passt.top/show_bug.cgi?id=18 are resolved
	log.Log.V(4).Info("Configuring unprivilegedPortStart to 0")
	err = b.handler.ConfigureUnprivilegedPortStart("0")
	if err != nil {
		log.Log.Reason(err).Errorf("failed to configure unprivilegedPortStart")
		return err
	}
	return nil
}

func (b *PasstPodNetworkConfigurator) GenerateNonRecoverableDomainIfaceSpec() *api.Interface {
	return nil
}
