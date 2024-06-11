package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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
	config, _, kvInformer := testutils.NewFakeClusterConfigUsingKV(kv)

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), &v1.KubeVirt{
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
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer.GetStore(), kv)
	}

	AfterEach(func() {
		disableFeatureGates()
	})

	DescribeTable("virtiofs privileged container", func(shouldEnableFeatureGate bool) {
		if shouldEnableFeatureGate {
			enableFeatureGate(virtconfig.VirtIOFSGate)
		}

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

		if shouldEnableFeatureGate {
			// PV
			Expect(container[0].SecurityContext.RunAsNonRoot).To(HaveValue(BeFalse()))
			Expect(container[0].SecurityContext.AllowPrivilegeEscalation).To(HaveValue(BeTrue()))
			// Secret
			Expect(container[1].SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
			Expect(container[1].SecurityContext.AllowPrivilegeEscalation).To(HaveValue(BeFalse()))
		} else {
			// PV
			Expect(container[0].SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
			Expect(container[0].SecurityContext.AllowPrivilegeEscalation).To(HaveValue(BeFalse()))
			// Secret
			Expect(container[1].SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
			Expect(container[1].SecurityContext.AllowPrivilegeEscalation).To(HaveValue(BeFalse()))
		}
	},
		Entry("Should create unprivileged containers only", false),
		Entry("Should create an unprivileged container and a privileged one", true),
	)
})
