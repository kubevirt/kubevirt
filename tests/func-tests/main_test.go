package tests_test

import (
	"context"
	"testing"

	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
	kvtutil "kubevirt.io/kubevirt/tests/util"
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
			Name: kvtutil.NamespaceTestDefault,
		},
	}
	opt := metav1.CreateOptions{}
	_, err = virtCli.CoreV1().Namespaces().Create(context.TODO(), ns, opt)
	if !errors.IsAlreadyExists(err) {
		kvtutil.PanicOnError(err)
	}

	tests.BeforeEach()
})

var _ = AfterSuite(func() {
	virtCli, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	opt := metav1.DeleteOptions{}
	kvtutil.PanicOnError(virtCli.CoreV1().Namespaces().Delete(context.TODO(), kvtutil.NamespaceTestDefault, opt))
})
