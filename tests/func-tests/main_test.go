package tests_test

import (
	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"
	"testing"

	"github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"
	testscore "kubevirt.io/kubevirt/tests"
)

func TestTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Tests Suite", NewReporters())
}

// NewReporters is a function to gather new ginkgo test reporters
func NewReporters() []Reporter {
	reporters := make([]Reporter, 0)
	if ginkgo_reporters.Polarion.Run {
		reporters = append(reporters, &ginkgo_reporters.Polarion)
	}
	if ginkgo_reporters.JunitOutput != "" {
		reporters = append(reporters, ginkgo_reporters.NewJunitReporter())
	}
	return reporters
}

var _ = BeforeSuite(func() {

	virtCli, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testscore.NamespaceTestDefault,
		},
	}
	_, err = virtCli.CoreV1().Namespaces().Create(ns)
	if !errors.IsAlreadyExists(err) {
		testscore.PanicOnError(err)
	}

	tests.BeforeEach()
})

var _ = AfterSuite(func() {
	virtCli, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	testscore.PanicOnError(virtCli.CoreV1().Namespaces().Delete(testscore.NamespaceTestDefault, &metav1.DeleteOptions{}))
})
