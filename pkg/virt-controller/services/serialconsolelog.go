package services

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func generateSerialConsoleLogContainer(vmi *v1.VirtualMachineInstance, image string, config *virtconfig.ClusterConfig) *k8sv1.Container {
	// TODO: eventually add a cluster-wide default
	if (vmi.Spec.Domain.Devices.AutoattachSerialConsole == nil || *vmi.Spec.Domain.Devices.AutoattachSerialConsole == true) && vmi.Spec.Domain.Devices.LogSerialConsole != nil && *vmi.Spec.Domain.Devices.LogSerialConsole == true {
		var serialPort uint = 0

		followretry := "-F"
		quiet := "--quiet"
		nodup := "-n+1"
		logFile := fmt.Sprintf("%s/%s/virt-serial%d-log", util.VirtPrivateDir, vmi.ObjectMeta.UID, serialPort)
		args := []string{quiet, nodup, followretry, logFile}

		resources := resourcesForSerialConsoleLogContainer(false, false, config)
		noPrivilegeEscalation := false
		nonRoot := true
		var userId int64 = util.NonRootUID

		return &k8sv1.Container{
			Name:            "guest-console-log",
			Image:           image,
			ImagePullPolicy: k8sv1.PullIfNotPresent,
			Command:         []string{"/usr/bin/tail"},
			Args:            args,
			VolumeMounts: []k8sv1.VolumeMount{
				k8sv1.VolumeMount{
					Name:      "private",
					MountPath: util.VirtPrivateDir,
					ReadOnly:  true,
				},
			},
			Resources: resources,
			SecurityContext: &k8sv1.SecurityContext{
				RunAsUser:                &userId,
				RunAsNonRoot:             &nonRoot,
				AllowPrivilegeEscalation: &noPrivilegeEscalation,
				Capabilities: &k8sv1.Capabilities{
					Drop: []k8sv1.Capability{"ALL"},
				},
			},
		}
	}

	return nil
}

func resourcesForSerialConsoleLogContainer(dedicatedCPUs bool, guaranteedQOS bool, config *virtconfig.ClusterConfig) k8sv1.ResourceRequirements {
	// TODO: tune this

	resources := k8sv1.ResourceRequirements{Requests: k8sv1.ResourceList{}, Limits: k8sv1.ResourceList{}}

	resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("10m")
	if reqCpu := config.GetSupportContainerRequest(v1.VirtioFS, k8sv1.ResourceCPU); reqCpu != nil {
		resources.Requests[k8sv1.ResourceCPU] = *reqCpu
	}
	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("80M")
	if limMem := config.GetSupportContainerLimit(v1.VirtioFS, k8sv1.ResourceMemory); limMem != nil {
		resources.Limits[k8sv1.ResourceMemory] = *limMem
	}

	resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("100m")
	if limCpu := config.GetSupportContainerLimit(v1.VirtioFS, k8sv1.ResourceCPU); limCpu != nil {
		resources.Limits[k8sv1.ResourceCPU] = *limCpu
	}
	if dedicatedCPUs || guaranteedQOS {
		resources.Requests[k8sv1.ResourceCPU] = resources.Limits[k8sv1.ResourceCPU]
	}

	if guaranteedQOS {
		resources.Requests[k8sv1.ResourceMemory] = resources.Limits[k8sv1.ResourceMemory]
	} else {
		resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1M")
		if reqMem := config.GetSupportContainerRequest(v1.VirtioFS, k8sv1.ResourceMemory); reqMem != nil {
			resources.Requests[k8sv1.ResourceMemory] = *reqMem
		}
	}

	return resources

}
