# Disk Rotation Rate Configuration

KubeVirt supports configuring the rotation rate for virtual disk devices, allowing you to emulate different types of storage devices (SSD vs HDD) in your virtual machines.

## Overview

The `rotationRate` field allows you to specify the rotation rate for disk devices, which affects how the guest operating system perceives the storage device. This feature is supported for **SCSI and SATA** bus types only.

This is particularly useful for:

- **SSD Emulation**: Setting `rotationRate: 1` emulates a solid-state drive with no rotation
- **HDD Emulation**: Setting values in the range 1025 to 65534 (e.g., `rotationRate: 7200`) emulates a hard disk drive with the specified RPM

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
          bus: scsi  # Supported bus types: scsi, sata
          rotationRate: 1  # SSD emulation
        name: ssd-disk
    memory:
      guest: 1Gi
  volumes:
  - name: ssd-disk
    persistentVolumeClaim:
      claimName: ssd-pvc
```

### HDD Emulation

To configure a disk as an HDD with specific RPM:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: vm-with-hdd
spec:
  domain:
    devices:
      disks:
      - disk:
          bus: scsi
          rotationRate: 7200  # HDD emulation at 7200 RPM
        name: hdd-disk
    memory:
      guest: 1Gi
  volumes:
  - name: hdd-disk
    persistentVolumeClaim:
      claimName: hdd-pvc
```

### Mixed Configuration

You can configure different rotation rates for different disks in the same VM:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: vm-mixed-storage
spec:
  domain:
    devices:
      disks:
      - disk:
          bus: scsi  # Supported bus types: scsi, sata
          rotationRate: 1  # SSD
        name: ssd-disk
      - disk:
          bus: scsi
          rotationRate: 5400  # HDD at 5400 RPM
        name: hdd-disk
      - disk:
          bus: scsi
          rotationRate: 10000  # High-performance HDD at 10000 RPM
        name: fast-hdd-disk
    memory:
      guest: 1Gi
  volumes:
  - name: ssd-disk
    persistentVolumeClaim:
      claimName: ssd-pvc
  - name: hdd-disk
    persistentVolumeClaim:
      claimName: hdd-pvc
  - name: fast-hdd-disk
    persistentVolumeClaim:
      claimName: fast-hdd-pvc
```

## Supported Values

- **1**: SSD emulation (no rotation)
- **1025 to 65534**: HDD emulation with the specified RPM value

### Common RPM Values

While any value in the range 1025-65534 is valid, these are commonly used:
- **5400**: HDD emulation at 5400 RPM (laptop drives)
- **7200**: HDD emulation at 7200 RPM (common desktop drives)
- **10000**: HDD emulation at 10000 RPM (high-performance drives)
- **15000**: HDD emulation at 15000 RPM (enterprise drives)

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
- Valid HDD rotation rate values are in the range 1025-65534 (per libvirt specification)
- Virtio disk devices do not support rotation rate configuration 