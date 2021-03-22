package virtwrap

import (
	"fmt"
	"net"
	"runtime"
	"strconv"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

func (l *LibvirtDomainManager) finalizeMigrationTarget(vmi *v1.VirtualMachineInstance) error {
	if err := l.hotPlugHostDevices(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to hot-plug host-devices")
	}

	if err := l.setGuestTime(vmi); err != nil {
		return err
	}

	return nil
}

func (l *LibvirtDomainManager) prepareMigrationTarget(vmi *v1.VirtualMachineInstance, useEmulation bool) error {

	logger := log.Log.Object(vmi)

	var emulatorThreadCpu *int
	domain := &api.Domain{}
	podCPUSet, err := util.GetPodCPUSet()
	if err != nil {
		logger.Reason(err).Error("failed to read pod cpuset.")
		return fmt.Errorf("failed to read pod cpuset: %v", err)
	}
	// reserve the last cpu for the emulator thread
	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		if len(podCPUSet) > 0 {
			emulatorThreadCpu = &podCPUSet[len(podCPUSet)]
			podCPUSet = podCPUSet[:len(podCPUSet)-1]
		}
	}
	// Check if PVC volumes are block volumes
	isBlockPVCMap := make(map[string]bool)
	isBlockDVMap := make(map[string]bool)
	diskInfo := make(map[string]*containerdisk.DiskInfo)
	for i, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			isBlockPVC, err := isBlockDeviceVolume(volume.Name)
			if err != nil {
				logger.Reason(err).Errorf("failed to detect volume mode for Volume %v and PVC %v.",
					volume.Name, volume.VolumeSource.PersistentVolumeClaim.ClaimName)
				return err
			}
			isBlockPVCMap[volume.Name] = isBlockPVC
		} else if volume.VolumeSource.ContainerDisk != nil {
			image, err := containerdisk.GetDiskTargetPartFromLauncherView(i)
			if err != nil {
				return err
			}
			info, err := converter.GetImageInfo(image)
			if err != nil {
				return err
			}
			diskInfo[volume.Name] = info
		} else if volume.VolumeSource.DataVolume != nil {
			isBlockDV, err := isBlockDeviceVolume(volume.Name)
			if err != nil {
				logger.Reason(err).Errorf("failed to detect volume mode for Volume %v and DataVolume %v.",
					volume.Name, volume.VolumeSource.DataVolume.Name)
				return err
			}
			isBlockDVMap[volume.Name] = isBlockDV
		}

	}
	// Map the VirtualMachineInstance to the Domain
	c := &converter.ConverterContext{
		Architecture:          runtime.GOARCH,
		VirtualMachine:        vmi,
		UseEmulation:          useEmulation,
		CPUSet:                podCPUSet,
		IsBlockPVC:            isBlockPVCMap,
		IsBlockDV:             isBlockDVMap,
		DiskType:              diskInfo,
		EmulatorThreadCpu:     emulatorThreadCpu,
		OVMFPath:              l.ovmfPath,
		UseVirtioTransitional: vmi.Spec.Domain.Devices.UseVirtioTransitional != nil && *vmi.Spec.Domain.Devices.UseVirtioTransitional,
	}
	if err := converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c); err != nil {
		return fmt.Errorf("conversion failed: %v", err)
	}

	dom, err := l.preStartHook(vmi, domain)
	if err != nil {
		return fmt.Errorf("pre-start pod-setup failed: %v", err)
	}

	err = l.generateCloudInitISO(vmi, nil)
	if err != nil {
		return err
	}
	// TODO this should probably a OnPrepareMigration hook or something.
	// Right now we need to call OnDefineDomain, so that additional setup, which might be done
	// by the hook can also be done for the new target pod
	hooksManager := hooks.GetManager()
	_, err = hooksManager.OnDefineDomain(&dom.Spec, vmi)
	if err != nil {
		return fmt.Errorf("executing custom preStart hooks failed: %v", err)
	}

	loopbackAddress := ip.GetLoopbackAddress()
	if err := updateHostsFile(fmt.Sprintf("%s %s\n", loopbackAddress, vmi.Status.MigrationState.TargetPod)); err != nil {
		return fmt.Errorf("failed to update the hosts file: %v", err)
	}

	isBlockMigration := (vmi.Status.MigrationMethod == v1.BlockMigration)
	migrationPortsRange := migrationproxy.GetMigrationPortsList(isBlockMigration)
	for _, port := range migrationPortsRange {
		// Prepare the direct migration proxy
		key := migrationproxy.ConstructProxyKey(string(vmi.UID), port)
		curDirectAddress := net.JoinHostPort(loopbackAddress, strconv.Itoa(port))
		unixSocketPath := migrationproxy.SourceUnixFile(l.virtShareDir, key)
		migrationProxy := migrationproxy.NewSourceProxy(unixSocketPath, curDirectAddress, nil, nil)

		err := migrationProxy.StartListening()
		if err != nil {
			logger.Reason(err).Errorf("proxy listening failed, socket %s", unixSocketPath)
			return err
		}
	}

	// since the source vmi is paused, add the vmi uuid to the pausedVMIs as
	// after the migration this vmi should remain paused.
	if vmiHasCondition(vmi, v1.VirtualMachineInstancePaused) {
		log.Log.Object(vmi).V(3).Info("adding vmi uuid to pausedVMIs list on the target")
		l.paused.add(vmi.UID)
	}

	return nil
}
