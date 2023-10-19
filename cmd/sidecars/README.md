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

Besides a binary, one could also execute shell or python scripts by making them available at the
expected location.

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

## Using ConfigMap to run custom script

Besides injecting a script into the container image, one can also store it in a ConfigMap and 
use annotations to make sure the script is run before the VMI creation. The flow would be as below:

1. Create a ConfigMap containing the shell or python script you would like to run
2. Create a VMI containing the annotation `hooks.kubevirt.io/hookSidecars` and mention the
   ConfigMap information in it.

## Examples

Create a ConfigMap using one of the following examples. In both the examples, we are modifying
the value of baseboard manufacturer for the VMI.

### Shell script

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config-map
data:
  my_script.sh: |
    #!/bin/sh
    tempFile=`mktemp --dry-run`
    echo $4 > $tempFile
    sed -i "s|<baseBoard></baseBoard>|<baseBoard><entry name='manufacturer'>Radical Edward</entry></baseBoard>|" $tempFile
    cat $tempFile
```

### Python script

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config-map
data:
  my_script.sh: |
    #!/usr/bin/env python

    import xml.etree.ElementTree as ET
    import sys

    def main(s):
        # write to a temporary file
        f = open("/tmp/orig.xml", "w")
        f.write(s)
        f.close()

        # parse xml from file
        xml = ET.parse("/tmp/orig.xml")
        # get the root element
        root = xml.getroot()
        # find the baseBoard element
        baseBoard = root.find("sysinfo").find("baseBoard")

        # prepare new element to be inserted into the xml definition
        element = ET.Element("entry", {"name": "manufacturer"})
        element.text = "Radical Edward"
        # insert the element
        baseBoard.insert(0, element)

        # write to a new file
        xml.write("/tmp/new.xml")
        # print file contents to stdout
        f = open("/tmp/new.xml")
        print(f.read())
        f.close()

    if __name__ == "__main__":
        main(sys.argv[4])
```

Above ConfigMap manifests have a shell/python script embedded in it. The script, when executed, modifies the baseboard
manufacturer information of the VMI in its XML definition and prints the output on stdout. This output is then used
to create a VMI on the cluster.

After creating one of the above ConfigMap, create the VMI using the manifest in
[this example](../../examples/vmi-with-sidecar-hook-configmap.yaml). Of importance here is the ConfigMap information stored in
the annotations:

```yaml
annotations:
  hooks.kubevirt.io/hookSidecars: '[{"args": ["--version", "v1alpha2"],
    "image": "registry:5000/kubevirt/sidecar-shim:devel",
    "configMap": {"name": "my-config-map", "key": "my_script.sh", "hookPath": "/usr/bin/onDefineDomain"}}]'
```

The `name` field indicates the name of the ConfigMap on the cluster which contains the script you 
want to execute. The `key` field indicates the key in the ConfigMap which contains the script to 
be executed. Finally, `hookPath` indicates the path where you would like the script to be 
mounted. It could be either of `/usr/bin/onDefineDomain` or `/usr/bin/preCloudInitIso` depending 
upon the hook you would like to execute.

After creating the VMI, verify that it is in the `Running` state, and connect to its console and
see if the desired changes to baseboard manufacturer get reflected:

```shell
# Once the VM is ready, connect to its display and login using name and password "fedora"
cluster/virtctl.sh vnc vmi-with-sidecar-hook-configmap

# Check whether the base board manufacturer value was successfully overwritten
sudo dmidecode -s baseboard-manufacturer
```