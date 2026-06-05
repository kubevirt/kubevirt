# Sidecars base image

## Introduction

In Kubernetes, Sidecar containers are containers that run along with the main container in the pod.
In the context of KubeVirt, we use the Sidecar container to apply changes before the Virtual Machine
is initialized.

> **Note**: The Sidecar feature gate must be enabled in the KubeVirt Custom Resource before using sidecars.
> Add `Sidecar` to `spec.configuration.developerConfiguration.featureGates` in your KubeVirt CR.
```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    developerConfiguration:
      featureGates:
        - Sidecar
```

Once enabled, every VM owner may use it to run arbitrary code in the context of virt-launcher which may have unexpected effects.

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

`sidecar-shim-image` is built as part of the Kubevirt build chain and its consumed as the
default image from `virt-operator` and `virt-controller`.

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
as a binary named `onDefineDomain` and installed under `/usr/bin` in a container that uses
`sidecar-shim-image` as base with an entrypoint of `/sidecar-shim`.

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

## Hook Sidecars Annotation

The `hooks.kubevirt.io/hookSidecars` annotation is a JSON array that defines one or more sidecar containers to be added to the virt-launcher pod. Each sidecar entry supports the following fields:

### Field Reference

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `image` | `string` | No | Container image to use for the sidecar. If not specified, the default `sidecar-shim-image` built by KubeVirt will be used. | `"image": "registry:5000/kubevirt/example-hook-sidecar:devel"` |
| `imagePullPolicy` | `string` | No | Image pull policy for the sidecar container. Must be one of: `IfNotPresent`, `Always`, or `Never`. If not specified, follows Kubernetes default behavior. | `"imagePullPolicy": "IfNotPresent"` |
| `command` | `array of strings` | No | Command to execute in the sidecar container. If not specified, the default entrypoint of the image will be used. | `"command": ["/custom-entrypoint", "--flag"]` |
| `args` | `array of strings` | No | Arguments to pass to the sidecar container command. For sidecar-shim, this typically includes the gRPC protocol version (e.g., `["--version", "v1alpha2"]`). | `"args": ["--version", "v1alpha2"]` |
| `configMap` | `object` | No | Reference to a ConfigMap containing a script to execute. The script will be mounted and executed by the sidecar-shim. See nested fields below. | See nested fields below |
| `configMap.name` | `string` | Yes | Name of the ConfigMap in the same namespace containing a script to execute. | `"name": "my-config-map"` |
| `configMap.key` | `string` | Yes | Key in the ConfigMap that contains the script. | `"key": "my_script.sh"` |
| `configMap.hookPath` | `string` | Yes | Path where the script will be mounted. Must be either `/usr/bin/onDefineDomain` or `/usr/bin/preCloudInitIso`. | `"hookPath": "/usr/bin/onDefineDomain"` |
| `pvc` | `object` | No | Reference to a PersistentVolumeClaim to mount in the sidecar container, optionally shared with the compute container. See nested fields below. | See nested fields below |
| `pvc.name` | `string` | Yes | Name of the PVC in the same namespace to mount in the sidecar container. | `"name": "my-pvc"` |
| `pvc.volumePath` | `string` | Yes | Mount path in the sidecar container. | `"volumePath": "/debug"` |
| `pvc.sharedComputePath` | `string` | No | Mount path in the compute (virt-launcher) container. If specified, the PVC will be shared between both containers. | `"sharedComputePath": "/var/run/debug"` |

### Complete Annotation Example

```yaml
annotations:
  hooks.kubevirt.io/hookSidecars: |
    [
      {
        "image": "registry:5000/kubevirt/example-hook-sidecar:devel",
        "imagePullPolicy": "IfNotPresent",
        "args": ["--version", "v1alpha2"]
      },
      {
        "imagePullPolicy": "IfNotPresent",
        "args": ["--version", "v1alpha2"],
        "configMap": {
          "name": "my-config-map",
          "key": "my_script.sh",
          "hookPath": "/usr/bin/onDefineDomain"
        }
      },
      {
        "image": "custom-sidecar:latest",
        "imagePullPolicy": "Always",
        "command": ["/custom-entrypoint"],
        "args": ["--custom-arg", "value"],
        "pvc": {
          "name": "shared-storage",
          "volumePath": "/data",
          "sharedComputePath": "/var/run/shared"
        }
      }
    ]
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
    "configMap": {"name": "my-config-map", "key": "my_script.sh", "hookPath": "/usr/bin/onDefineDomain"}}]'
```

Please notice that annotations set on VMs are not automatically propagated to VMIs so,
in the case of a VM, the VM owner should configure it on `/spec/template/metadata/annotations`
instead of directly annotating the VM as in [this example](../../examples/vm-cirros-with-sidecar-hook-configmap.yaml).
The annotation will be rendered on the generated VMI once the VM will be restarted.

After creating the VMI, verify that it is in the `Running` state, and connect to its console and
see if the desired changes to baseboard manufacturer get reflected:

```shell
# Once the VM is ready, connect to its display and login using name and password "fedora"
hack/virtctl.sh vnc vmi-with-sidecar-hook-configmap

# Check whether the base board manufacturer value was successfully overwritten
sudo dmidecode -s baseboard-manufacturer
# or
cat /sys/devices/virtual/dmi/id/board_vendor
```
