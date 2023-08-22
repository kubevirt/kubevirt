# Sidecars base image

## Introduction

In Kubernetes, Sidecar containers are containers that run along with the main container in the pod.
In the context of KubeVirt, we use the Sidecar container to apply changes before the Virtual Machine
is initialized.

The Sidecar containers communicate with the main container over a socket with a gRPC protocol, [with
two versions at moment](../../pkg/hooks). The Sidecar is meant to do the changes over libvirt's XML
and return the new XML over gRPC for the VM creation.

## Sidecar-shim image

To reduce the amount of boilerplate that developers need to do in order to run VM with custom
modifications, we introduced the `sidecar-shim-image` that takes care of implementing the
communication with the main container.

The `sidecar-shim-image` contains the `sidecar-shim` binary which should be kept as the entrypoint
of the container. This binary will search in $PATH for binaries named after the Hook names (e.g
onDefineDomain) and run them and provide the necessary arguments as command line options (flags).

In the case of `onDefineDomain`, the arguments will be the VMI information as JSON string, (e.g
--vmi vmiJSON) and the current domain XML (e.g --domain domainXML) to the users binaries. As
standard output it expects the modified domain XML.

In the case of `preCloudInitIso`, the arguments will be the VMI information as JSON string, (e.g
--vmi vmiJSON) and the [CloudInitData](../../pkg/cloud-init/cloud-init.go) (e.g --cloud-init
cloudInitJSON) to the users binaries. As standard output it expects the modified CloudInitData (as
JSON).

## Notes

The `sidecar-shim` binary needs to inform what gRPC protocol version it'll communicate with, so it
requires a `--version` parameter (e.g: v1alpha2)

## Example

Using the current [smbios sidecar](../example-hook-sidecar/) as example. The `smbios.go` is compiled
and installed in `/usr/bin/onDefineDomain` in a container that uses `sidecar-shim-image` as base.

```go
const (
	baseBoardManufacturerAnnotation = "smbios.vm.kubevirt.io/baseBoardManufacturer"
)

func onDefineDomain(vmiJSON, domainXML []byte) string {
	vmiSpec := vmSchema.VirtualMachineInstance{}
	if err := json.Unmarshal(vmiJSON, &vmiSpec); err != nil {
		panic(err)
	}

	domainSpec := api.DomainSpec{}
	if err := xml.Unmarshal(domainXML, &domainSpec); err != nil {
		panic(err)
	}

	annotations := vmiSpec.GetAnnotations()
	if _, found := annotations[baseBoardManufacturerAnnotation]; !found {
		return string(domainXML)
	}

	domainSpec.OS.SMBios = &api.SMBios{Mode: "sysinfo"}
	if domainSpec.SysInfo == nil {
		domainSpec.SysInfo = &api.SysInfo{}
	}
	domainSpec.SysInfo.Type = "smbios"
	if baseBoardManufacturer, found := annotations[baseBoardManufacturerAnnotation]; found {
		domainSpec.SysInfo.BaseBoard = append(domainSpec.SysInfo.BaseBoard, api.Entry{
			Name:  "manufacturer",
			Value: baseBoardManufacturer,
		})
	}
	if newDomainXML, err := xml.Marshal(domainSpec); err != nil {
		panic(err)
	} else {
		return string(newDomainXML)
	}
}

func main() {
	var vmiJSON, domainXML string
	pflag.StringVar(&vmiJSON, "vmi", "", "VMI to change in JSON format")
	pflag.StringVar(&domainXML, "domain", "", "Domain spec in XML format")
	pflag.Parse()

	if vmiJSON == "" || domainXML == "" {
		os.Exit(1)
	}
	fmt.Println(onDefineDomain([]byte(vmiJSON), []byte(domainXML)))
}
```
