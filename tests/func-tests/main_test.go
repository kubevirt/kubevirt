package tests_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

func TestTests(t *testing.T) {
	GinkgoWriter.Printf("Start running the HCO functional tests; go version: %s; platform: %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	RegisterFailHandler(Fail)
	RunSpecs(t, "HyperConverged cluster E2E Test suite")
}

var _ = BeforeSuite(func(ctx context.Context) {
	cli := tests.GetK8sClientSet()

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: tests.TestNamespace,
		},
	}

	opt := metav1.CreateOptions{}
	_, err := cli.CoreV1().Namespaces().Create(ctx, ns, opt)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			panic(err)
		}
	}

	controllerCli := tests.GetControllerRuntimeClient()
	isOpenshift, err := tests.IsOpenShift(ctx, controllerCli)
	if err != nil {
		Fail("can't tell the cluster type; " + err.Error())
	}

	if isOpenshift {
		By("adding temporary route")
		Eventually(ctx, func(g Gomega) error {
			return tests.CreateTempRoute(ctx, controllerCli)
		}).WithTimeout(time.Second * 60).
			WithPolling(time.Second).
			WithContext(ctx).
			Should(Succeed())
	}

	tests.BeforeEach(ctx)

	DeferCleanup(func(ctx context.Context) {
		if isOpenshift {
			By("removing the temporary route")
			Eventually(ctx, func() error {
				return tests.DeleteTempRoute(ctx, controllerCli)
			}).WithTimeout(time.Second * 60).
				WithPolling(time.Second).
				WithContext(ctx).
				Should(Succeed())
		}
	})
})

var _ = AfterSuite(func(ctx context.Context) {
	cli := tests.GetK8sClientSet()
	err := cli.CoreV1().Namespaces().Delete(ctx, tests.TestNamespace, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		panic(err)
	}
})
