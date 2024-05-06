# API serialization compatibility tests

This directory tree contains serialized API objects in json and yaml formats.

## Populating data for each release

When KubeVirt cuts a new release of the project, the current version files will be copied to the release version

For example, to capture compatibility data for `v1.2.0`:

```sh
export VERSION=release-1.2
git checkout ${VERSION}
cp -fr staging/src/kubevirt.io/api/testdata/{HEAD,${VERSION}}
git checkout -b add-${VERSION}-api-testdata master
git add .
git commit -m "Add ${VERSION} API testdata"
```

## Current version

The `HEAD` subdirectory contains serialized API objects generated from the current commit:

```
HEAD/
  <group>.<version>.<kind>.[json|yaml]
```

To run serialization tests just for the current version:

```sh
go test staging/src/kubevirt.io/api -run //HEAD
```

All the formats of a given group/version/kind are expected to decode successfully to identical objects,
and to round-trip back to serialized form with identical bytes.
Adding new fields or deprecating new fields or API types *is* expected to modify these fixtures. To regenerate them, run:

```sh
UPDATE_COMPATIBILITY_FIXTURE_DATA=true go test staging/src/kubevirt.io/api -run //HEAD
```

## Previous versions

The vX.Y.0 subdirectories contain serialized API objects from previous releases:

```
release-X.Y
  <group>.<version>.<kind>.[json|yaml]
```

To run serialization tests for a previous version, like `v1.1.0`:

```sh
go test staging/src/kubevirt.io/api -run //release-1.1
```

To run serialization tests for a particular group/version/kind, like `apps/v1` `Deployment`:
```sh
go test staging/src/kubevirt.io/api -run /apps.v1.Deployment/
```

Example output:

```    
--- FAIL: TestCompatibility/kubevirt.io.v1.VirtualMachineInstance (0.01s)
        --- FAIL: TestCompatibility/kubevirt.io.v1.VirtualMachineInstance/release-0.50 (0.01s)
            compatibility.go:416: json differs
            compatibility.go:417:   (
                        """
                        ... // 215 identical lines
                                      "readonly": true
                                    },
                -                   "floppy": {
                -                     "readonly": true,
                -                     "tray": "trayValue"
                -                   },
                                    "cdrom": {
                                      "bus": "busValue",
                        ... // 678 identical lines
                              "tscFrequency": -12
                            },
                -           "virtualMachineRevisionName": "virtualMachineRevisionNameValue"
                +           "virtualMachineRevisionName": "virtualMachineRevisionNameValue",
                +           "runtimeUser": 0
                          }
                        }
                        """
                  )
                
            compatibility.go:422: yaml differs
            compatibility.go:423:   (
                        """
                        ... // 237 identical lines
                                  pciAddress: pciAddressValue
                                  readonly: true
                -               floppy:
                -                 readonly: true
                -                 tray: trayValue
                                io: ioValue
                                lun:
                        ... // 341 identical lines
                          qosClass: qosClassValue
                          reason: reasonValue
                +         runtimeUser: 0
                          topologyHints:
                            tscFrequency: -12
                        ... // 22 identical lines
                        """
                  )
                
```

The above output shows that for VirtualMachineInstance:
1. api-field: `spec.domain.devices.disks.floppy` was dropped. [ref-1](https://github.com/kubevirt/kubevirt/issues/2016)[ref-2](https://github.com/kubevirt/kubevirt/pull/2164)
2. api-field: `status.runtimeUser` field was added[ref-3](https://github.com/kubevirt/kubevirt/pull/6709)

