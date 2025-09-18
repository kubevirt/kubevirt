# Disk Solid State (SSD) Emulation

KubeVirt supports SSD emulation for virtual disks using the `solidState` field.

## Overview

The `solidState` field allows you to specify whether a disk should be emulated as a SSD (non-rotational) or as a spinning disk (HDD). This feature is supported for **SCSI and SATA** bus types only.

- **SSD Emulation**: Setting `solidState: true` emulates a solid-state drive
- **HDD Emulation**: Setting `solidState: false` or leaving unset emulates a spinning disk

## Usage

### SSD Emulation

To configure a disk for SSD emulation (a non-rotational disk):

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: vm-with-ssd
spec:
  domain:
    devices:
      disks:
      - disk:
          bus: scsi  # Supported bus types include: scsi, sata
          solidState: true  # Enables SSD emulation for the target virtual disk
        name: ssd-disk
    memory:
      guest: 1Gi
  volumes:
  - name: ssd-disk
    persistentVolumeClaim:
      claimName: ssd-pvc
```

## Supported Values

- **true**: SSD emulation (non-rotational)
- **false** or unset: spinning disk (HDD emulation)

## Bus Compatibility

The `solidState` field is compatible with the following disk bus types:
- `scsi`
- `sata`

**Note**: USB and virtio bus types do not support SSD emulation.

## Technical Details

If `solidState: true`, the rotation rate is set to 1 in the underlying libvirt/QEMU configuration. Otherwise, no rotation rate is set (spinning disk).

## Example Output

When using `solidState: true` (SSD emulation), the QEMU command line will include (`rotation_rate=1`):

```
-device 'scsi-hd,bus=virtioscsi0.0,channel=0,scsi-id=0,lun=0,drive=drive-scsi0,id=scsi0,rotation_rate=1,bootindex=100'
```

## Notes

- The `solidState` field is optional. If not specified, the disk will continue to report as a rotational device (HDD)
- This feature requires QEMU support for the `rotation_rate` parameter on SCSI and SATA devices
- The SSD emulation feature is purely OS-level reporting, and will not impact the performance of the underlying devices