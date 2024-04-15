package tests_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	kvtutil "kubevirt.io/kubevirt/tests/util"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

func TestTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HyperConverged cluster E2E Test suite")
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
