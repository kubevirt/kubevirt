package tests_test

import (
	"flag"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Dependency objects", func() {
	flag.Parse()

	var stopChan chan struct{}

	BeforeEach(func() {
		tests.BeforeEach()
		stopChan = make(chan struct{})
	})

	AfterEach(func() {
		close(stopChan)
	})

	It("should list priority classes", func() {
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		_, err = virtCli.SchedulingV1().PriorityClasses().Get("kubevirt-cluster-critical", v1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

})
