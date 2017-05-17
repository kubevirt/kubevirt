package tests_test

import (
	"flag"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/tests"
	"time"
)

var _ = Describe("Storage", func() {

	flag.Parse()

	coreClient, err := kubecli.Get()
	tests.PanicOnError(err)

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.MustCleanup()
	})

	getTargetLogs := func(tailLines int64) string {
		pods, err := coreClient.CoreV1().Pods(kubev1.NamespaceDefault).List(kubev1.ListOptions{LabelSelector: "app in (iscsi-demo-target)"})
		Expect(err).ToNot(HaveOccurred())

		//FIXME Sometimes pods hang in terminating state, select the pod which does not have a deletion timestamp
		podName := ""
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp == nil {
				podName = pod.ObjectMeta.Name
				break
			}
		}
		Expect(podName).ToNot(BeEmpty())

		logsRaw, err := coreClient.CoreV1().
			Pods("default").
			GetLogs(podName,
				&kubev1.PodLogOptions{TailLines: &tailLines}).
			DoRaw()
		Expect(err).To(BeNil())
		return string(logsRaw)
	}

	BeforeEach(func() {
		// Wait until there is no connection
		logs := func() string { return getTargetLogs(70) }
		Eventually(logs,
			4*time.Second,
			500*time.Millisecond).
			Should(ContainSubstring("I_T nexus information:\n    LUN information:"))
	})

	RunVMAndExpectLaunch := func(vm *v1.VM) {
		obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
		Expect(err).To(BeNil())
		tests.WaitForSuccessfulVMStart(obj)

		// Let's get the IP of the pod of the VM
		pods, err := coreClient.CoreV1().Pods(api.NamespaceDefault).List(services.UnfinishedVMPodSelector(vm))
		//FIXME Sometimes pods hang in terminating state, select the pod which does not have a deletion timestamp
		podIP := ""
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp == nil {
				podIP = pod.Status.PodIP
				break
			}
		}
		Expect(podIP).ToNot(BeEmpty())

		// Periodically check if we now have a connection on the target
		// We don't check against the full pod IP, since depending on the kubernetes proxy mode, we either see the
		// full PodIP or just the proxy IP which connects through different ports
		Eventually(func() string { return getTargetLogs(70) },
			4*time.Second,
			500*time.Millisecond).
			Should(ContainSubstring(fmt.Sprintf("IP Address: %s", podIP[0:8])))
	}

	Context("Given a fresh iSCSI target", func() {

		It("should be available and ready", func() {
			logs := getTargetLogs(70)
			Expect(logs).To(ContainSubstring("Target 1: iqn.2017-01.io.kubevirt:sn.42"))
			Expect(logs).To(ContainSubstring("Driver: iscsi"))
			Expect(logs).To(ContainSubstring("State: ready"))
		})

		It("should not have any connections", func() {
			logs := getTargetLogs(70)
			// Ensure that no connections are listed
			Expect(logs).To(ContainSubstring("I_T nexus information:\n    LUN information:"))
		})
	})

	Context("Given a VM and a directly connected Alpine LUN", func() {

		It("should be successfully started by libvirt", func(done Done) {
			// Start the VM with the LUN attached
			vm := tests.NewRandomVMWithDirectLun(2)
			RunVMAndExpectLaunch(vm)
			close(done)
		}, 50)
	})

	Context("Given a VM and an Alpine PVC", func() {
		It("should be successfully started by libvirt", func(done Done) {
			// Start the VM with the PVC attached
			vm := tests.NewRandomVMWithPVC("disk-alpine")
			RunVMAndExpectLaunch(vm)
			close(done)
		}, 50)
	})

	AfterEach(func() {
		tests.MustCleanup()
	})
})
