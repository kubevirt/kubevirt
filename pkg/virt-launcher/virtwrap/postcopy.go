package virtwrap

import (
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

func (l *LibvirtDomainManager) updateVMIMigrationMode(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, mode v1.MigrationMode) error {
	domainSpec, err := l.getDomainSpec(dom)
	if err != nil {
		return err
	}

	if domainSpec.Metadata.KubeVirt.Migration == nil {
		domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{}
	}

	domainSpec.Metadata.KubeVirt.Migration.Mode = mode

	_, err = l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return err
	}

	return nil
}
