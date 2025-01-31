package annotations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"

	"kubevirt.io/kubevirt/pkg/instancetype/annotations"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Annotations", func() {
	var (
		vm   *v1.VirtualMachine
		meta *metav1.ObjectMeta
	)

	const instancetypeName = "instancetype-name"

	BeforeEach(func() {
		vm = libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithInstancetype(instancetypeName))

		meta = &metav1.ObjectMeta{}
	})

	It("should add instancetype name annotation", func() {
		vm.Spec.Instancetype.Kind = apiinstancetype.SingularResourceName

		annotations.Set(vm, meta)

		Expect(meta.Annotations[v1.InstancetypeAnnotation]).To(Equal(instancetypeName))
		Expect(meta.Annotations[v1.ClusterInstancetypeAnnotation]).To(Equal(""))
	})

	It("should add cluster instancetype name annotation", func() {
		vm.Spec.Instancetype.Kind = apiinstancetype.ClusterSingularResourceName

		annotations.Set(vm, meta)

		Expect(meta.Annotations[v1.InstancetypeAnnotation]).To(Equal(""))
		Expect(meta.Annotations[v1.ClusterInstancetypeAnnotation]).To(Equal(instancetypeName))
	})

	It("should add cluster name annotation, if instancetype.kind is empty", func() {
		vm.Spec.Instancetype.Kind = ""

		annotations.Set(vm, meta)

		Expect(meta.Annotations[v1.InstancetypeAnnotation]).To(Equal(""))
		Expect(meta.Annotations[v1.ClusterInstancetypeAnnotation]).To(Equal(instancetypeName))
	})
})
