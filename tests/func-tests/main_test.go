package tests_test

import (
	"testing"

	"github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"
	testscore "kubevirt.io/kubevirt/tests"
	flags "kubevirt.io/kubevirt/tests/flags"
)

func TestTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tests Suite")
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

	jobType := tests.GetJobTypeEnvVar()
	if jobType == "prow" {
		kubevirtCfg, err := virtCli.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(tests.KubevirtCfgMap, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		kubevirtCfg.Data["debug.useEmulation"] = "true"
		_, err = virtCli.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Update(kubevirtCfg)
		Expect(err).ToNot(HaveOccurred())
	}
})

var _ = AfterSuite(func() {
	virtCli, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	testscore.PanicOnError(virtCli.CoreV1().Namespaces().Delete(testscore.NamespaceTestDefault, &metav1.DeleteOptions{}))
})
