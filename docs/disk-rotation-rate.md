# Disk Rotation Rate Configuration

KubeVirt supports SSD emulation for virtual disks.

## Overview

The `rotationRate` field allows you to specify the rotation rate for disk devices, which affects how the guest operating system perceives the storage device. This feature is supported for **SCSI and SATA** bus types only.

This is particularly useful for:

- **SSD Emulation**: Setting `rotationRate: 1` emulates a solid-state drive with no rotation

## Usage

### SSD Emulation

To configure a disk as an SSD (no rotation):

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
          rotationRate: 1  # 1 is used for SSD emulation, empty for no emulation
        name: ssd-disk
    memory:
      guest: 1Gi
  volumes:
  - name: ssd-disk
    persistentVolumeClaim:
      claimName: ssd-pvc
```

## Supported Values

- **1**: SSD emulation (no rotation)

## Bus Compatibility

The `rotationRate` field is compatible with the following disk bus types:
- `scsi`
- `sata`

**Note**: USB and virtio bus types do not support rotation rate configuration.

## Technical Details

The rotation rate is passed to QEMU as the `rotation_rate` parameter for disk devices. This affects:

1. **Guest OS Detection**: The guest operating system will detect the disk as having the specified rotation characteristics
2. **Performance Expectations**: Some guest OS features may adjust behavior based on perceived disk type
3. **Monitoring**: Disk monitoring tools in the guest will report the configured rotation rate

## Example Output

When using `rotationRate: 1` (SSD emulation), the QEMU command line will include:

```
-device 'scsi-hd,bus=virtioscsi0.0,channel=0,scsi-id=0,lun=0,drive=drive-scsi0,id=scsi0,rotation_rate=1,bootindex=100'
```

## Notes

- The `rotationRate` field is optional. If not specified, no rotation rate is set
- This feature requires QEMU support for the `rotation_rate` parameter on SCSI and SATA devices
- The rotation rate is purely for emulation and does not affect the actual performance of the underlying storage
- Only `rotationRate: 1` is supported at this time.