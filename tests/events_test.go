package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	ioerrorPV  = "ioerror-pv"
	ioerrorPVC = "ioerror-pvc"
	deviceName = "errdev0"
	diskName   = "disk0"
)

var _ = Describe("[Serial][sig-storage]K8s IO events", func() {
	var (
		nodeName   string
		virtClient kubecli.KubevirtClient
		pv         *k8sv1.PersistentVolume
		pvc        *k8sv1.PersistentVolumeClaim
	)

	isExpectedIOEvent := func(e corev1.Event, vmiName string) bool {
		if e.Type == "Warning" &&
			e.Reason == "IOerror" &&
			e.Message == "VM Paused due to IO error at the volume: "+diskName &&
			e.InvolvedObject.Kind == "VirtualMachineInstance" &&
			e.InvolvedObject.Name == vmiName {
			return true
		}
		return false
	}

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		nodeName = tests.NodeNameWithHandler()
		tests.CreateFaultyDisk(nodeName, deviceName)
		pv, pvc, err = tests.CreatePVandPVCwithFaultyDisk(nodeName, deviceName, tests.NamespaceTestDefault)
		Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for faulty disk")
	})
	AfterEach(func() {
		tests.RemoveFaultyDisk(nodeName, deviceName)

		err := virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})
	It("[test_id:6225]Should catch the IO error event", func() {
		By("Creating VMI with faulty disk")
		vmi := tests.NewRandomVMIWithPVC(pvc.Name)
		vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).To(BeNil(), "Failed to create vmi")

		tests.WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(vmi, 120)

		By("Expecting  paused event on VMI ")
		Eventually(func() bool {
			events, err := virtClient.CoreV1().Events(tests.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, e := range events.Items {
				if isExpectedIOEvent(e, vmi.Name) {
					return true
				}
			}

			return false
		}, 30*time.Second, 5*time.Second).Should(BeTrue())
		err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
		Expect(err).To(BeNil(), "Failed to delete VMI")
		tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
	})
})
