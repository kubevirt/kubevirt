package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"

	"github.com/spf13/pflag"
	vmSchema "kubevirt.io/api/core/v1"
	"libvirt.org/go/libvirtxml"
)

const (
	virtioFsSocketDirAnnotation       = "serviceaccounts.vm.kubevirt.io/socketSourceDir"
	serviceAccountTargetDirAnnotation = "serviceaccounts.vm.kubevirt.io/targetDir"
)

func onDefineDomain(vmiJSON, domainXML []byte) (string, error) {
	vmiSpec := vmSchema.VirtualMachineInstance{}
	if err := json.Unmarshal(vmiJSON, &vmiSpec); err != nil {
		return "", fmt.Errorf("failed to unmarshal given VMI spec: %s %s", err, string(vmiJSON))
	}

	// Read the source socket file and its mount target from the VMI spec
	if _, ok := vmiSpec.Annotations[serviceAccountTargetDirAnnotation]; !ok {
		return "", fmt.Errorf("target directory annotation not set, exiting")
	}

	if _, ok := vmiSpec.Annotations[virtioFsSocketDirAnnotation]; !ok {
		return "", fmt.Errorf("source directory annotation not set, exiting")
	}

	domainSpec := libvirtxml.Domain{}
	if err := xml.Unmarshal(domainXML, &domainSpec); err != nil {
		return "", fmt.Errorf("failed to unmarshal given domain spec: %s %s", err, string(domainXML))
	}

	// TODO: Verify if this configuration applies to everything
	var (
		d uint = 0x0000
		b uint = 0x01
		f uint = 0x00
		s uint = 0x00
	)

	// Create a libvirt domain entry for a virtio filesystem that maps to the socket file of the service account mounted from the pod
	domainSpec.Devices.Filesystems = append(domainSpec.Devices.Filesystems, libvirtxml.DomainFilesystem{
		Address: &libvirtxml.DomainAddress{
			PCI: &libvirtxml.DomainAddressPCI{
				Domain:   &d,
				Bus:      &b,
				Function: &f,
				Slot:     &s,
			},
		},
		Source: &libvirtxml.DomainFilesystemSource{Mount: &libvirtxml.DomainFilesystemSourceMount{Dir: vmiSpec.Annotations[virtioFsSocketDirAnnotation]}},
		Target: &libvirtxml.DomainFilesystemTarget{Dir: vmiSpec.Annotations[serviceAccountTargetDirAnnotation]},
		Driver: &libvirtxml.DomainFilesystemDriver{Type: "virtiofs"}})

	domainSpec.Devices.Shmems = append(domainSpec.Devices.Shmems, libvirtxml.DomainShmem{
		Name: "shmem1",
		Size: &libvirtxml.DomainShmemSize{
			Value: 32,
			Unit:  "M",
		},
		Model: &libvirtxml.DomainShmemModel{
			Type: "ivshmem-plain",
		},
	})

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal new Domain spec: %s %+v", err, domainSpec)
	}

	return string(newDomainXML), nil
}

func main() {
	var vmiJSON, domainXML string
	pflag.StringVar(&vmiJSON, "vmi", "", "VMI to change in JSON format")
	pflag.StringVar(&domainXML, "domain", "", "Domain spec in XML format")
	pflag.Parse()

	logger := log.New(os.Stderr, "serviceaccounts", log.Ldate)
	if vmiJSON == "" || domainXML == "" {
		logger.Printf("Bad input vmi=%d, domain=%d", len(vmiJSON), len(domainXML))
		os.Exit(1)
	}

	domainXML, err := onDefineDomain([]byte(vmiJSON), []byte(domainXML))
	if err != nil {
		logger.Printf("onDefineDomain failed: %s", err)
		panic(err)
	}
	fmt.Println(domainXML)
}
