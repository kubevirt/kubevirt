package services

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("virtiofs container", func() {

	kv := &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
		Spec: v1.KubeVirtSpec{
			Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{},
			},
		},
		Status: v1.KubeVirtStatus{
			Phase:               v1.KubeVirtPhaseDeploying,
			DefaultArchitecture: "amd64",
		},
	}
	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featureGate},
					},
				},
			},
		})
	}

	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	BeforeEach(func() {
		enableFeatureGate(featuregate.VirtIOFSStorageVolumeGate)
	})

	AfterEach(func() {
		disableFeatureGates()
	})

	It("should create unprivileged containers only", func() {
		vmi := api.NewMinimalVMI("testvm")

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "sharedtestdisk",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: testutils.NewFakePersistentVolumeSource(),
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "sharedtestdisk",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "secret-volume",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: "test-secret",
				},
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "secret-volume",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})

		container := generateVirtioFSContainers(vmi, "virtiofs-container", config)
		Expect(container).To(HaveLen(2))

		// PV
		Expect(container[0].SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
		Expect(container[0].SecurityContext.AllowPrivilegeEscalation).To(HaveValue(BeFalse()))
		// Secret
		Expect(container[1].SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
		Expect(container[1].SecurityContext.AllowPrivilegeEscalation).To(HaveValue(BeFalse()))
	})

	It("should propagate SubPath and ReadOnly from FilesystemVirtiofs to the volume mount", func() {
		vmi := api.NewMinimalVMI("testvm")

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "sharedtestdisk",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: testutils.NewFakePersistentVolumeSource(),
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name: "sharedtestdisk",
			Virtiofs: &v1.FilesystemVirtiofs{
				SubPath:  "data/sub",
				ReadOnly: true,
			},
		})

		containers := generateVirtioFSContainers(vmi, "virtiofs-container", config)
		Expect(containers).To(HaveLen(1))

		// The first VolumeMount is the shared socket dir; the data volume mount is appended after it.
		Expect(containers[0].VolumeMounts).To(HaveLen(2))
		dataMount := containers[0].VolumeMounts[1]
		Expect(dataMount.Name).To(Equal("sharedtestdisk"))
		Expect(dataMount.SubPath).To(Equal("data/sub"))
		Expect(dataMount.ReadOnly).To(BeTrue())
	})

	It("should default SubPath to empty and ReadOnly to false when FilesystemVirtiofs is empty", func() {
		vmi := api.NewMinimalVMI("testvm")

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "sharedtestdisk",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: testutils.NewFakePersistentVolumeSource(),
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "sharedtestdisk",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})

		containers := generateVirtioFSContainers(vmi, "virtiofs-container", config)
		Expect(containers).To(HaveLen(1))

		Expect(containers[0].VolumeMounts).To(HaveLen(2))
		dataMount := containers[0].VolumeMounts[1]
		Expect(dataMount.Name).To(Equal("sharedtestdisk"))
		Expect(dataMount.SubPath).To(BeEmpty())
		Expect(dataMount.ReadOnly).To(BeFalse())
	})

	It("should not add a data volume mount for auto-mounted volumes even when SubPath/ReadOnly are set", func() {
		vmi := api.NewMinimalVMI("testvm")

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "sa-volume",
			VolumeSource: v1.VolumeSource{
				ServiceAccount: &v1.ServiceAccountVolumeSource{
					ServiceAccountName: "default",
				},
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name: "sa-volume",
			Virtiofs: &v1.FilesystemVirtiofs{
				SubPath:  "ignored",
				ReadOnly: true,
			},
		})

		containers := generateVirtioFSContainers(vmi, "virtiofs-container", config)
		Expect(containers).To(HaveLen(1))

		// Auto-mounted volumes only get the socket mount; no data volume mount is appended,
		// so SubPath/ReadOnly have nowhere to land.
		Expect(containers[0].VolumeMounts).To(HaveLen(1))
	})

	It("should skip ContainerPath volumes", func() {
		vmi := api.NewMinimalVMI("testvm")

		// Add a PVC volume with filesystem - should create container
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "pvc-volume",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: testutils.NewFakePersistentVolumeSource(),
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "pvc-volume",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})

		// Add a ContainerPath volume with filesystem - should be skipped
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "token-volume",
			VolumeSource: v1.VolumeSource{
				ContainerPath: &v1.ContainerPathVolumeSource{
					Path: "/var/run/secrets/token",
				},
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "token-volume",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})

		containers := generateVirtioFSContainers(vmi, "virtiofs-container", config)

		// Only the PVC volume should have a container, not the ContainerPath volume
		Expect(containers).To(HaveLen(1))
		Expect(containers[0].Name).To(Equal("virtiofs-pvc-volume"))
	})

	It("should translate UIDs for ServiceAccounts", func() {
		vmi := api.NewMinimalVMI("testvm")

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "sa-volume",
			VolumeSource: v1.VolumeSource{
				ServiceAccount: &v1.ServiceAccountVolumeSource{
					ServiceAccountName: "default",
				},
			},
		})

		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "sa-volume",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})

		container := generateVirtioFSContainers(vmi, "virtiofs-container", config)
		Expect(container).To(HaveLen(1))

		mapUidToGuestRoot := fmt.Sprintf("--translate-uid=host:%d:0:1", util.NonRootUID)
		Expect(container[0].Args).Should(ContainElement(mapUidToGuestRoot))
	})
})
