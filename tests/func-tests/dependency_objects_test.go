package tests_test

import (
	"context"
	"flag"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("[rfe_id:5672][crit:medium][vendor:cnv-qe@redhat.com][level:system]Dependency objects", Label("PriorityClass"), func() {
	flag.Parse()

	var stopChan chan struct{}

	BeforeEach(func() {
		tests.BeforeEach()
		stopChan = make(chan struct{})
	})

	AfterEach(func() {
		close(stopChan)
	})

	It("[test_id:5674]should get the created priority class for critical workloads", Label("test_id:5674"), func() {
		cli := tests.GetK8sClientSet()
		_, err := cli.SchedulingV1().PriorityClasses().Get(context.TODO(), priorityClassName, v1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

})
