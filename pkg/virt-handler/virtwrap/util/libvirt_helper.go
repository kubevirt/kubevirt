package util

import (
	"encoding/xml"
	"reflect"

	"github.com/libvirt/libvirt-go"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
)

func SetDomainSpec(virConn cli.Connection, vm *v1.VirtualMachine, wantedSpec api.DomainSpec) (cli.VirDomain, error) {
	xmlStr, err := xml.Marshal(&wantedSpec)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Generating the domain XML failed.")
		return nil, err
	}
	log.Log.Object(vm).V(3).With("xml", xmlStr).Info("Domain XML generated.")
	dom, err := virConn.DomainDefineXML(string(xmlStr))
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Defining the VM failed.")
		return nil, err
	}
	return dom, nil
}

func GetDomainSpec(dom cli.VirDomain) (*api.DomainSpec, error) {
	spec, err := GetDomainSpecWithFlags(dom, libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		return nil, err
	}

	inactiveSpec, err := GetDomainSpecWithFlags(dom, libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(spec.Metadata, inactiveSpec.Metadata) {
		// Metadata is updated on offline config only. As a result,
		// We have to merge updates to metadata into the domain spec.
		metadata := &inactiveSpec.Metadata
		metadata.DeepCopyInto(&spec.Metadata)
	}

	return spec, nil
}

func GetDomainSpecWithFlags(dom cli.VirDomain, flags libvirt.DomainXMLFlags) (*api.DomainSpec, error) {
	domain := &api.DomainSpec{}
	domxml, err := dom.GetXMLDesc(flags)
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal([]byte(domxml), domain)
	if err != nil {
		return nil, err
	}

	return domain, nil
}
