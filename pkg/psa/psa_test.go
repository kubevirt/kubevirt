package psa

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PSA", func() {
	var (
		privilegedNamespace *k8sv1.Namespace
		restrictedNamespace *k8sv1.Namespace
	)

	BeforeEach(func() {
		privilegedNamespace = newNamespace("privileged")
		restrictedNamespace = newNamespace("restricted")
	})

	Context("should report correct PSA level", func() {
		DescribeTable("when inspecting namespace", func(namespace *k8sv1.Namespace, privileged bool) {
			Expect(IsNamespacePrivileged(namespace)).To(Equal(privileged))
		},
			Entry("privileged", privilegedNamespace, true),
			Entry("restricted", restrictedNamespace, false),
			Entry("with no label", &k8sv1.Namespace{}, false),
		)
	})
})

func newNamespace(level string) *k8sv1.Namespace {
	return &k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				PSALabel: level,
			},
		},
	}
}
