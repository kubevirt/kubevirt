package tests_test

import (
	"flag"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Storage", func() {

	fmt.Printf("")
	flag.Parse()

	coreClient, err := kubecli.Get()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.MustCleanup()
	})

	Context("Given a fresh iSCSI target", func() {
		tailLines := int64(70)
		var logs string

		It("should provide logs", func() {
			logsRaw, err := coreClient.CoreV1().
				Pods("default").
				GetLogs("iscsi-demo-target-tgtd",
					&kubev1.PodLogOptions{TailLines: &tailLines}).
				DoRaw()
			Expect(err).To(BeNil())
			logs = string(logsRaw)
		})

		It("should be available and ready", func() {
			Expect(logs).To(ContainSubstring("Target 1: iqn.2017-01.io.kubevirt:sn.42"))
			Expect(logs).To(ContainSubstring("Driver: iscsi"))
			Expect(logs).To(ContainSubstring("State: ready"))
		})

		It("should not have any connections", func() {
			// Ensure that no connections are listed
			Expect(logs).To(ContainSubstring("I_T nexus information:\n    LUN information:"))
		})
	})

	AfterEach(func() {
		tests.MustCleanup()
	})
})
