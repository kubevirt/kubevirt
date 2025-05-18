package services

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
)

var _ = Describe("Container spec renderer", func() {
	exampleCommand := []string{"/bin/bash"}

	var specRenderer *ContainerSpecRenderer

	const (
		containerName = "exampleContainer"
		img           = "megaimage2000"
		pullPolicy    = k8sv1.PullAlways
	)

	Context("without any options", func() {
		BeforeEach(func() {
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy)
		})

		It("should have the unprivileged root user security context", func() {
			Expect(specRenderer.Render(exampleCommand).SecurityContext).Should(
				Equal(unprivilegedRootUserSecurityContext()))
		})
	})

	Context("with non root user option", func() {
		const nonRootUser = 207

		BeforeEach(func() {
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithNonRoot(nonRootUser))
		})

		It("should feature the XDG environment variables", func() {
			Expect(specRenderer.Render(exampleCommand).Env).Should(
				ConsistOf(
					k8sv1.EnvVar{
						Name:  cacheHomeEnvVarName,
						Value: util.VirtPrivateDir,
					}, k8sv1.EnvVar{
						Name:  configHomeEnvVarName,
						Value: util.VirtPrivateDir,
					}, k8sv1.EnvVar{
						Name:  runtimeDirEnvVarName,
						Value: varRun,
					},
				))
		})

		It("AllowPrivilegeEscalation should be set to false", func() {
			Expect(*specRenderer.Render(exampleCommand).SecurityContext.AllowPrivilegeEscalation).Should(BeFalse())
		})
	})

	Context("with privileged option", func() {
		BeforeEach(func() {
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithPrivileged())
		})

		It("should feature a privileged security context", func() {
			Expect(specRenderer.Render(exampleCommand).SecurityContext).Should(
				Equal(privilegedRootUserSecurityContext()),
			)
		})
	})

	Context("vmi capabilities", func() {
		allowedCapabilities := []k8sv1.Capability{
			CAP_NET_BIND_SERVICE,
			CAP_SYS_NICE,
		}
		Context("a VMI running as root", func() {
			BeforeEach(func() {
				specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithCapabilities(simplestVMI()))
			})

			It("must request to add the NET_BIND_SERVICE and SYS_NICE capabilities", func() {
				Expect(specRenderer.Render(exampleCommand).SecurityContext.Capabilities.Add).To(
					ConsistOf(allowedCapabilities))
			})

			Context("with a virtioFS filesystem", func() {
				BeforeEach(func() {
					const rootUser = 0
					specRenderer = NewContainerSpecRenderer(
						containerName,
						img,
						pullPolicy,
						WithCapabilities(vmiWithVirtioFS(rootUser)))
				})

				It("cannot request additional capabilities", func() {
					Expect(specRenderer.Render(exampleCommand).SecurityContext.Capabilities.Add).Should(
						ConsistOf(allowedCapabilities))
				})
			})
		})

		Context("a VMI belonging to a non root user", func() {
			BeforeEach(func() {
				const nonRootUser = 207
				specRenderer = NewContainerSpecRenderer(
					containerName,
					img,
					pullPolicy,
					WithCapabilities(nonRootVMI(nonRootUser)))
			})

			It("must request the NET_BIND_SERVICE capability", func() {
				Expect(specRenderer.Render(exampleCommand).SecurityContext.Capabilities.Add).Should(
					ConsistOf(k8sv1.Capability(CAP_NET_BIND_SERVICE)))
			})
		})
	})

	Context("with volume devices option", func() {
		const (
			volumeName = "asd"
			volumePath = "/tmp/mega-path"
		)

		DescribeTable("the expected `VolumeDevice`s are rendered into the container", func(devices ...k8sv1.VolumeDevice) {
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithVolumeDevices(devices...))
			Expect(specRenderer.Render(exampleCommand).VolumeDevices).To(ConsistOf(devices))
		},
			Entry("no volume devices are passed as options"),
			Entry("one optional volume device is added to the renderer", volumeDevice(volumeName, volumePath)),
		)
	})

	Context("with volume mounts option", func() {
		const (
			mountName = "asd"
			mountPath = "/tmp/mega-path"
		)

		DescribeTable("the expected `VolumeMount`s are rendered into the container", func(mounts ...k8sv1.VolumeMount) {
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithVolumeMounts(mounts...))
			Expect(specRenderer.Render(exampleCommand).VolumeMounts).To(ConsistOf(mounts))
		},
			Entry("no volume devices are passed as options"),
			Entry("one optional volume device is added to the renderer", volumeMount(mountName, mountPath)),
		)
	})

	Context("with resources option", func() {
		var expectedResource k8sv1.ResourceRequirements

		BeforeEach(func() {
			expectedResource = resources("10", "100")
			specRenderer = NewContainerSpecRenderer(
				containerName, img, pullPolicy, WithResourceRequirements(expectedResource))
		})

		It("the resource requirements are rendered into the container", func() {
			Expect(specRenderer.Render(exampleCommand).Resources).To(Equal(expectedResource))
		})
	})

	Context("with no capabilities option", func() {
		It("all capabilities should be dropped", func() {
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithNoCapabilities())
			Expect(specRenderer.Render(exampleCommand).SecurityContext.Capabilities.Drop).To(Equal([]k8sv1.Capability{"ALL"}))
		})
	})

	Context("with drop-all capabilities option", func() {
		It("all capabilities should be dropped, but added caps should be kept", func() {
			vmi := simplestVMI()
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithCapabilities(vmi), WithDropALLCapabilities())
			Expect(specRenderer.Render(exampleCommand).SecurityContext.Capabilities.Drop).To(Equal([]k8sv1.Capability{"ALL"}))
			Expect(specRenderer.Render(exampleCommand).SecurityContext.Capabilities.Add).ToNot(BeEmpty())
		})
	})

	Context("vmi with ports allowed in its spec", func() {
		var ports []v1.Port

		BeforeEach(func() {
			const ifaceName = "not-relevant"
			ports = []v1.Port{{
				Name: "http", Port: 80},
				{Protocol: "UDP", Port: 80},
				{Port: 90},
				{Name: "other-http", Port: 80}}
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithPorts(
				vmiWithInterfaceWithPortAllowList(ifaceName, ports...)))
		})

		It("the container should feature the same port list", func() {
			Expect(specRenderer.Render(exampleCommand).Ports).To(HaveLen(len(ports)))
			for i := range ports {
				Expect(specRenderer.Render(exampleCommand).Ports[i]).To(
					Equal(vmPortToContainerPort(ports[i])))
			}
		})
	})

	Context("container command and arguments", func() {
		DescribeTable("", func(args ...string) {
			specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithArgs(args))
			Expect(specRenderer.Render(exampleCommand).Command).To(Equal(exampleCommand))
			Expect(specRenderer.Render(exampleCommand).Args).To(Equal(args))
		},
			Entry("without input args"),
			Entry("with an input argument", "do-stuff"),
		)
	})

	Context("vmi with probes", func() {
		Context("readiness probe", func() {
			It("its pod should feature the same probe but with an additional 10 seconds initial delay", func() {
				probe := dummyProbe()
				specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithReadinessProbe(
					vmiWithReadinessProbe(probe)))
				Expect(specRenderer.Render(exampleCommand).ReadinessProbe).To(Equal(probeWithDelay(probe)))
			})
		})

		Context("liveness probe", func() {
			It("its pod should feature the same probe but with an additional 10 seconds initial delay", func() {
				probe := dummyProbe()
				specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithLivelinessProbe(
					vmiWithLivenessProbe(probe)))
				Expect(specRenderer.Render(exampleCommand).LivenessProbe).To(Equal(probeWithDelay(probe)))
			})
		})

		Context("liveness exec probe", func() {
			It("should wrap the liveness exec probe command inside virt-probe while preserving the original command", func() {
				probe := dummyProbe()
				probe.Handler = v1.Handler{
					Exec: &k8sv1.ExecAction{Command: []string{"dummy-cli"}},
				}
				specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithLivelinessProbe(
					vmiWithLivenessProbe(probe)))
				Expect(specRenderer.Render(exampleCommand).LivenessProbe.Exec.Command).To(HaveExactElements(
					"virt-probe",
					"--domainName", "_",
					"--timeoutSeconds", strconv.FormatInt(int64(dummyProbe().TimeoutSeconds), 10),
					"--command", "dummy-cli",
					"--"))
			})
		})

		Context("pre-wrapped liveness exec probe", func() {
			It("should avoid wrapping the liveness exec probe a second time", func() {
				var expectedExecCmd = []string{"virt-probe", "--", "dummy-cli"}
				probe := dummyProbe()
				probe.Handler = v1.Handler{
					Exec: &k8sv1.ExecAction{Command: expectedExecCmd},
				}
				specRenderer = NewContainerSpecRenderer(containerName, img, pullPolicy, WithLivelinessProbe(
					vmiWithLivenessProbe(probe)))
				Expect(specRenderer.Render(exampleCommand).LivenessProbe.Exec.Command).To(Equal(expectedExecCmd))
			})
		})
	})
})

func vmiWithInterfaceWithPortAllowList(ifaceName string, ports ...v1.Port) *v1.VirtualMachineInstance {
	vmi := simplestVMI()
	vmi.Spec.Domain = v1.DomainSpec{
		Devices: v1.Devices{
			Interfaces: []v1.Interface{
				{Name: ifaceName, Ports: ports},
			}},
	}
	return vmi
}

func vmiWithVirtioFS(user uint64) *v1.VirtualMachineInstance {
	const fsName = "0_o"
	vmi := nonRootVMI(user)
	vmi.Spec.Domain = v1.DomainSpec{
		Devices: v1.Devices{
			Filesystems: []v1.Filesystem{{
				Name:     fsName,
				Virtiofs: &v1.FilesystemVirtiofs{},
			}},
		},
	}
	return vmi
}

func nonRootVMI(user uint64) *v1.VirtualMachineInstance {
	vmi := simplestVMI()
	vmi.Status = v1.VirtualMachineInstanceStatus{
		RuntimeUser: user,
	}
	return vmi
}

func simplestVMI() *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		Spec: v1.VirtualMachineInstanceSpec{},
	}
}

func unprivilegedRootUserSecurityContext() *k8sv1.SecurityContext {
	return securityContext(util.RootUser, false, nil)
}

func privilegedRootUserSecurityContext() *k8sv1.SecurityContext {
	return securityContext(util.RootUser, true, nil)
}

func volumeDevice(name string, path string) k8sv1.VolumeDevice {
	return k8sv1.VolumeDevice{
		Name:       name,
		DevicePath: path,
	}
}

func volumeMount(name string, path string) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      name,
		ReadOnly:  false,
		MountPath: path,
	}
}

func resources(cpu string, memory string) k8sv1.ResourceRequirements {
	return k8sv1.ResourceRequirements{
		Limits: map[k8sv1.ResourceName]resource.Quantity{
			k8sv1.ResourceCPU:    resource.MustParse(cpu),
			k8sv1.ResourceMemory: resource.MustParse(memory),
		},
	}
}

func vmPortToContainerPort(vmiPort v1.Port) k8sv1.ContainerPort {
	protocol := "TCP"
	if vmiPort.Protocol != "" {
		protocol = vmiPort.Protocol
	}
	return k8sv1.ContainerPort{
		Name:          vmiPort.Name,
		ContainerPort: vmiPort.Port,
		Protocol:      k8sv1.Protocol(protocol),
	}
}

func dummyProbe() *v1.Probe {
	return &v1.Probe{
		Handler:             v1.Handler{HTTPGet: httpHandler()},
		InitialDelaySeconds: 2,
		TimeoutSeconds:      3,
		PeriodSeconds:       4,
		SuccessThreshold:    100,
		FailureThreshold:    200,
	}
}

func httpHandler() *k8sv1.HTTPGetAction {
	return &k8sv1.HTTPGetAction{
		Path:        "1234",
		Port:        intstr.IntOrString{},
		Host:        "1234",
		Scheme:      "1234",
		HTTPHeaders: nil,
	}
}

func vmiWithReadinessProbe(probe *v1.Probe) *v1.VirtualMachineInstance {
	vmi := simplestVMI()
	vmi.Spec.ReadinessProbe = probe
	return vmi
}

func vmiWithLivenessProbe(probe *v1.Probe) *v1.VirtualMachineInstance {
	vmi := simplestVMI()
	vmi.Spec.LivenessProbe = probe
	return vmi
}

func probeWithDelay(probe *v1.Probe) *k8sv1.Probe {
	if probe == nil {
		return nil
	}
	return &k8sv1.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds + 10,
		TimeoutSeconds:      probe.TimeoutSeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
		ProbeHandler: k8sv1.ProbeHandler{
			Exec:      probe.Exec,
			HTTPGet:   probe.HTTPGet,
			TCPSocket: probe.TCPSocket,
		},
	}
}
