package virtwrap

import (
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"

	libvirt "libvirt.org/libvirt-go"
)

func (l *LibvirtDomainManager) registerIterationEventForPostCopy(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, options *cmdclient.MigrationOptions, migrationError chan error) {
	l.virConn.DomainEventMigrationIterationRegister(func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventMigrationIteration) {
		if event.Iteration == 1 {
			log.Log.Object(vmi).Info("Signaled start for post copy migration")

			err := dom.MigrateStartPostCopy(uint32(0))
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("Live postcopy migration failed to start.")
				migrationError <- err
				return
			}

			err = l.updateVMIMigrationMode(dom, vmi, v1.MigrationPostCopy)
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("Unable to update migration mode on domain xml")
			}
		}
	})
}

func vmiHasLocalStorage(vmi *v1.VirtualMachineInstance) bool {
	for _, volume := range vmi.Spec.Volumes {
		if volume.EmptyDisk != nil || volume.VolumeSource.Ephemeral != nil {
			return true
		}
	}

	return false
}

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
