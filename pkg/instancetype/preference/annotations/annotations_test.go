package annotations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/annotations"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Preferences - annotations", func() {
	var (
		vm   *v1.VirtualMachine
		meta *metav1.ObjectMeta
	)

	const preferenceName = "preference-name"

	BeforeEach(func() {
		vm = libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithPreference(preferenceName))

		meta = &metav1.ObjectMeta{}
	})

	It("should add preference name annotation", func() {
		vm.Spec.Preference.Kind = apiinstancetype.SingularPreferenceResourceName

		annotations.Set(vm, meta)

		Expect(meta.Annotations).To(HaveKeyWithValue(v1.PreferenceAnnotation, preferenceName))
		Expect(meta.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
	})

	It("should add cluster preference name annotation", func() {
		vm.Spec.Preference.Kind = apiinstancetype.ClusterSingularPreferenceResourceName

		annotations.Set(vm, meta)

		Expect(meta.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
		Expect(meta.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, preferenceName))
	})

	It("should add cluster name annotation, if preference.kind is empty", func() {
		vm.Spec.Preference.Kind = ""

		annotations.Set(vm, meta)

		Expect(meta.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
		Expect(meta.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, preferenceName))
	})
})
