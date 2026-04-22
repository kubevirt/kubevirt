package callback

import (
	"encoding/xml"
	"fmt"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const libvirtDomainQemuSchema = "http://libvirt.org/schemas/domain/qemu/1.0"

type DomainSpecMutator interface {
	Mutate(*domainschema.DomainSpec) (*domainschema.DomainSpec, error)
}

func OnDefineDomain(domainXML []byte, domSpecMutator DomainSpecMutator) ([]byte, error) {
	domainSpec := &domainschema.DomainSpec{
		XmlNS: libvirtDomainQemuSchema,
	}
	if err := xml.Unmarshal(domainXML, domainSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal given domain spec: %v", err)
	}

	updatedDomainSpec, err := domSpecMutator.Mutate(domainSpec)
	if err != nil {
		return nil, err
	}

	updatedDomainSpecXML, err := xml.Marshal(updatedDomainSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated domain spec: %v", err)
	}

	return updatedDomainSpecXML, nil
}
