package services

import (
	"fmt"

	"k8s.io/utils/pointer"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func generateSerialConsoleLogContainer(vmi *v1.VirtualMachineInstance, image string, config *virtconfig.ClusterConfig, virtLauncherLogVerbosity uint) *k8sv1.Container {
	const serialPort = 0
	if isSerialConsoleLogEnabled(vmi, config) {
		logFile := fmt.Sprintf("%s/%s/virt-serial%d-log", util.VirtPrivateDir, vmi.ObjectMeta.UID, serialPort)

		resources := resourcesForSerialConsoleLogContainer(vmi.IsCPUDedicated(), vmi.WantsToHaveQOSGuaranteed(), config)

		guestConsoleLog := &k8sv1.Container{
			Name:            "guest-console-log",
			Image:           image,
			ImagePullPolicy: k8sv1.PullIfNotPresent,
			Command:         []string{"/usr/bin/virt-tail"},
			Args:            []string{"--logfile", logFile},
			VolumeMounts: []k8sv1.VolumeMount{
				k8sv1.VolumeMount{
					Name:      "private",
					MountPath: util.VirtPrivateDir,
					ReadOnly:  true,
				},
			},
			Resources: resources,
			SecurityContext: &k8sv1.SecurityContext{
				RunAsUser:                pointer.Int64(util.NonRootUID),
				RunAsNonRoot:             pointer.Bool(true),
				AllowPrivilegeEscalation: pointer.Bool(false),
				Capabilities: &k8sv1.Capabilities{
					Drop: []k8sv1.Capability{"ALL"},
				},
			},
		}

		guestConsoleLog.Env = append(guestConsoleLog.Env, k8sv1.EnvVar{Name: ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY, Value: fmt.Sprint(virtLauncherLogVerbosity)})

		return guestConsoleLog
	}

	return nil
}

func isSerialConsoleLogEnabled(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig) bool {
	if vmi.Spec.Domain.Devices.AutoattachSerialConsole != nil && *vmi.Spec.Domain.Devices.AutoattachSerialConsole == false {
		return false
	}
	if vmi.Spec.Domain.Devices.LogSerialConsole != nil {
		return *vmi.Spec.Domain.Devices.LogSerialConsole
	}
	return !config.IsSerialConsoleLogDisabled()
}

func resourcesForSerialConsoleLogContainer(dedicatedCPUs bool, guaranteedQOS bool, config *virtconfig.ClusterConfig) k8sv1.ResourceRequirements {
	resources := k8sv1.ResourceRequirements{Requests: k8sv1.ResourceList{}, Limits: k8sv1.ResourceList{}}

	resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("35M")
	if reqMem := config.GetSupportContainerRequest(v1.GuestConsoleLog, k8sv1.ResourceMemory); reqMem != nil {
		resources.Requests[k8sv1.ResourceMemory] = *reqMem
	}
	resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("5m")
	if reqCpu := config.GetSupportContainerRequest(v1.GuestConsoleLog, k8sv1.ResourceCPU); reqCpu != nil {
		resources.Requests[k8sv1.ResourceCPU] = *reqCpu
	}

	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("60M")
	if limMem := config.GetSupportContainerLimit(v1.GuestConsoleLog, k8sv1.ResourceMemory); limMem != nil {
		resources.Limits[k8sv1.ResourceMemory] = *limMem
	}
	resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("15m")
	if limCpu := config.GetSupportContainerLimit(v1.GuestConsoleLog, k8sv1.ResourceCPU); limCpu != nil {
		resources.Limits[k8sv1.ResourceCPU] = *limCpu
	}

	if dedicatedCPUs || guaranteedQOS {
		resources.Requests[k8sv1.ResourceCPU] = resources.Limits[k8sv1.ResourceCPU]
		resources.Requests[k8sv1.ResourceMemory] = resources.Limits[k8sv1.ResourceMemory]
	}

	return resources
}
