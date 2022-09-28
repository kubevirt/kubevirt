package converter

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func convertFileSystems(fileSystems []v1.Filesystem) []api.FilesystemDevice {
	domainFileSystems := []api.FilesystemDevice{}
	for _, fs := range fileSystems {
		if fs.Virtiofs == nil {
			continue
		}

		domainFileSystems = append(domainFileSystems,
			api.FilesystemDevice{
				Type:       "mount",
				AccessMode: "passthrough",
				Driver: &api.FilesystemDriver{
					Type:  "virtiofs",
					Queue: "1024",
				},
				Source: &api.FilesystemSource{
					Socket: services.VirtioFSSocketPath(fs.Name),
				},
				Target: &api.FilesystemTarget{
					Dir: fs.Name,
				},
			})
	}

	return domainFileSystems
}
