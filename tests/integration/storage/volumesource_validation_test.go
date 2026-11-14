package storage_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

// These specs exercise the +kubebuilder:validation:XValidation (CEL) rules on
// ContainerPathVolumeSource (staging/src/kubevirt.io/api/core/v1/schema.go),
// asserting them directly against the generated CRD schema via a real,
// in-process API server. No KubeVirt webhook runs in this environment, so a
// rejection here can only come from the CRD-embedded CEL rule itself.
var _ = Describe("ContainerPathVolumeSource CEL validation", func() {
	var ns string

	BeforeEach(func(ctx SpecContext) {
		ns = f.CreateNamespace(ctx)
	})

	It("should accept an absolute path with readOnly true", func(ctx SpecContext) {
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithFilesystemContainerPath("share", "/data"),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject a relative path", func(ctx SpecContext) {
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithFilesystemContainerPath("share", "data"),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring("path must be absolute")))
	})

	It("should reject a path containing '..'", func(ctx SpecContext) {
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithFilesystemContainerPath("share", "/data/../etc"),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring("path must not contain '..'")))
	})

	It("should reject readOnly set to false", func(ctx SpecContext) {
		vmi := libvmi.New(libvmi.WithMemoryRequest("128Mi"))
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "share",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "share",
			ContainerPath: &v1.ContainerPathVolumeSource{
				Path:     "/data",
				ReadOnly: new(false),
			},
		})

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring("readOnly must be true")))
	})
})
