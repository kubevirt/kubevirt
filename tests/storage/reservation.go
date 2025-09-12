package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

// The SCSI persistent reservation tests require to run serially because of the
// feature gate PersistentReservation. The enablement/disablement of this
// feature gate redeploys virt-handler pod, and this might interfere with other
// tests.
var _ = Describe(SIG("SCSI persistent reservation", Serial, func() {
	const randLen = 8
	var (
		naa          string
		backendDisk  string
		disk         string
		targetCliPod string
		virtClient   kubecli.KubevirtClient
		node         string
		device       string
		fgDisabled   bool
		pv           *corev1.PersistentVolume
		pvc          *corev1.PersistentVolumeClaim
	)

	// NAA is the Network Address Authority and it is an identifier represented
	// in ASCII-encoded hexadecimal digits
	// More details at:
	//  https://datatracker.ietf.org/doc/html/rfc3980#ref-FC-FS
	generateNaa := func() string {
		const letterBytes = "0123456789abcdef"
		b := make([]byte, 14)
		for i := range b {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}
		// Keep the first 2 digits constants as not all combinations are valid naa
		return "52" + string(b)
	}

	// executeTargetCli executes command targetcli
	executeTargetCli := func(podName string, args []string) {
		cmd := append([]string{"/usr/bin/targetcli"}, args...)
		pod, err := k8s.Client().CoreV1().Pods(testsuite.NamespacePrivileged).Get(context.Background(), podName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(pod, "targetcli", cmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("command='targetcli %v' stdout='%s' stderr='%s'", args, stdout, stderr))
	}

	// createSCSIDisk creates a SCSI using targetcli utility and LinuxIO (see
	// http://linux-iscsi.org/wiki/LIO).
	// For avoiding any confusion, this function doesn't rely on the scsi_debug module
	// as creates a SCSI disk that supports the SCSI protocol. Hence, it can be used to test
	// SCSI commands such as the persistent reservation
	createSCSIDisk := func(podName, pvc string) {
		diskSize := "1G"
		// Create PVC where we store the backend storage for the SCSI disks
		libstorage.CreateFSPVC(pvc, testsuite.NamespacePrivileged, diskSize, libstorage.WithStorageProfile())
		// Create targetcli container
		By("Create targetcli pod")
		pod, err := libpod.Run(libpod.RenderTargetcliPod(podName, pvc), testsuite.NamespacePrivileged)
		Expect(err).ToNot(HaveOccurred())
		node = pod.Spec.NodeName
		// The vm-killer image is built with bazel and the /etc/ld.so.cache isn't built
		// at the package installation. The targetcli utility relies on ctype python package that
		// uses it to find shared library.
		// To fix this issue, we run ldconfig before targetcli
		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(pod, "targetcli", []string{"ldconfig"})
		By(fmt.Sprintf("ldconfig: stdout: %v stderr: %v", stdout, stderr))
		Expect(err).ToNot(HaveOccurred())

		// Create backend file. Let some room for metedata and create a
		// slightly smaller backend image, we use 800M instead of 1G. In
		// this case, the disk size doesn't matter as the disk is used
		// mostly to test the SCSI persistent reservation ioctls.
		executeTargetCli(podName, []string{
			"backstores/fileio",
			"create", backendDisk, "/disks/disk.img", "800M"})
		executeTargetCli(podName, []string{
			"loopback/", "create", naa})
		// Create LUN
		executeTargetCli(podName, []string{
			fmt.Sprintf("loopback/naa.%s/luns", naa),
			"create",
			fmt.Sprintf("/backstores/fileio/%s", backendDisk)})
	}

	// findSCSIdisk returns the first scsi disk that correspond to the model. With targetcli the model name correspond to the name of the storage backend.
	// Example:
	// $ lsblk --scsi -o NAME,MODEL -p -n
	// /dev/sda disk1
	findSCSIdisk := func(podName string, model string) string {
		var device string
		pod, err := k8s.Client().CoreV1().Pods(testsuite.NamespacePrivileged).Get(context.Background(), podName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(pod, "targetcli",
			[]string{"/bin/lsblk", "--scsi", "-o", "NAME,MODEL", "-p", "-n"})
		Expect(err).ToNot(HaveOccurred(), stdout, stderr)
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			if strings.Contains(line, model) {
				line = strings.TrimSpace(line)
				disk := strings.Split(line, " ")
				if len(disk) < 1 {
					continue
				}
				device = disk[0]
				break
			}
		}
		return device

	}

	checkResultCommand := func(vmi *v1.VirtualMachineInstance, cmd, output string) bool {
		res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
			&expect.BSnd{S: fmt.Sprintf("%s\n", cmd)},
			&expect.BExp{R: ""},
		}, 20)
		Expect(err).ToNot(HaveOccurred())
		return strings.Contains(res[0].Output, output)
	}

	waitForVirtHandlerWithPrHelperReadyOnNode := func(node string) {
		ready := false
		fieldSelector, err := fields.ParseSelector("spec.nodeName==" + string(node))
		Expect(err).ToNot(HaveOccurred())
		labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel + "=virt-handler"))
		Expect(err).ToNot(HaveOccurred())
		selector := metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
		Eventually(func() bool {
			pods, err := k8s.Client().CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), selector)
			Expect(err).ToNot(HaveOccurred())
			if len(pods.Items) < 1 {
				return false
			}
			// Virt-handler will be deployed together with the
			// pr-helepr container
			if len(pods.Items[0].Spec.Containers) != 2 {
				return false
			}
			for _, status := range pods.Items[0].Status.ContainerStatuses {
				if status.State.Running != nil {
					ready = true
				} else {
					return false
				}
			}
			return ready

		}, 90*time.Second, 1*time.Second).Should(BeTrue())
	}
	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		fgDisabled = !checks.HasFeature(featuregate.PersistentReservation)
		if fgDisabled {
			config.EnableFeatureGate(featuregate.PersistentReservation)
		}

	})
	AfterEach(func() {
		if fgDisabled {
			config.DisableFeatureGate(featuregate.PersistentReservation)
		}
	})

	Context("Use LUN disk with persistent reservation", func() {
		BeforeEach(func() {
			var err error
			naa = generateNaa()
			backendDisk = "disk" + rand.String(randLen)
			disk = "disk-" + rand.String(randLen)
			targetCliPod = "targetcli-" + rand.String(randLen)
			// Create the scsi disk
			createSCSIDisk(targetCliPod, disk)
			// Avoid races if there is some delay in the device creation
			Eventually(findSCSIdisk, 20*time.Second, 1*time.Second).WithArguments(targetCliPod, backendDisk).ShouldNot(BeEmpty())
			device = findSCSIdisk(targetCliPod, backendDisk)
			Expect(device).ToNot(BeEmpty())
			By(fmt.Sprintf("Create PVC with SCSI disk %s", device))
			pv, pvc, err = CreatePVandPVCwithSCSIDisk(node, device, testsuite.NamespaceTestDefault, "scsi-disks", "scsipv", "scsipvc")
			Expect(err).ToNot(HaveOccurred())
			waitForVirtHandlerWithPrHelperReadyOnNode(node)
			// Switching the PersistentReservation feature gate on/off
			// causes redeployment of all KubeVirt components.
			By("Ensuring all KubeVirt components are ready")
			testsuite.EnsureKubevirtReady()
		})

		AfterEach(func() {
			// Delete the scsi disk
			executeTargetCli(targetCliPod, []string{
				"loopback/", "delete", naa})
			executeTargetCli(targetCliPod, []string{
				"backstores/fileio", "delete", backendDisk})
			Expect(k8s.Client().CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())

		})

		It("Should successfully start a VM with persistent reservation", func() {
			By("Create VMI with the SCSI disk")
			vmi := libvmifact.NewFedora(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithPersistentVolumeClaimLun("lun0", pvc.Name, true),
				libvmi.WithNodeAffinityFor(node),
			)
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(180),
			)
			By("Requesting SCSI persistent reservation")
			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")
			Expect(checkResultCommand(vmi, "sg_persist -i -k /dev/sda",
				"there are NO registered reservation keys")).To(BeTrue())
			Expect(checkResultCommand(vmi, "sg_persist -o -G  --param-sark=12345678 /dev/sda",
				"Peripheral device type: disk")).To(BeTrue())
			Eventually(func(g Gomega) {
				g.Expect(
					checkResultCommand(vmi, "sg_persist -i -k /dev/sda", "1 registered reservation key follows:\r\n    0x12345678\r\n"),
				).To(BeTrue())
			}).
				Within(60 * time.Second).
				WithPolling(10 * time.Second).
				Should(Succeed())

			// Restart virt-handler
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(),
				vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			pod, err := libnode.GetVirtHandlerPod(k8s.Client(), vmi.Status.NodeName)
			Expect(err).ToNot(HaveOccurred())
			err = k8s.Client().CoreV1().Pods(flags.KubeVirtInstallNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			// Wait unti new handler pod is ready
			oldPodName := pod.Name
			Eventually(func(g Gomega) bool {
				pod, err = libnode.GetVirtHandlerPod(k8s.Client(), vmi.Status.NodeName)
				g.Expect(err).To(Or(Succeed(), MatchError("Expected to find one Pod, found 2 Pods")))
				if err != nil {
					return false
				}
				pod, err = k8s.Client().CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				return pod.Name != oldPodName
			}).WithTimeout(time.Minute).WithPolling(time.Second).Should(BeTrue())
			Eventually(matcher.ThisPod(pod)).WithTimeout(30 * time.Second).
				WithPolling(1 * time.Second).Should(matcher.HaveConditionTrue(k8sv1.PodReady))

			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")
			Expect(
				checkResultCommand(vmi, "sg_persist -i -k /dev/sda", "1 registered reservation key follows:\r\n    0x12345678\r\n"),
			).To(BeTrue())
		})

		It("Should successfully start 2 VMs with persistent reservation on the same LUN", func() {
			By("Create 2 VMs with the SCSI disk")
			vmi := libvmifact.NewFedora(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithPersistentVolumeClaimLun("lun0", pvc.Name, true),
				libvmi.WithNodeAffinityFor(node),
			)
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(180),
			)

			vmi2 := libvmifact.NewFedora(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithPersistentVolumeClaimLun("lun0", pvc.Name, true),
				libvmi.WithNodeAffinityFor(node),
			)
			vmi2, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi2)).Create(context.Background(), vmi2, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi2,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(180),
			)

			By("Requesting SCSI persistent reservation from the first VM")
			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")
			Expect(checkResultCommand(vmi, "sg_persist -i -k /dev/sda",
				"there are NO registered reservation keys")).To(BeTrue())
			Expect(checkResultCommand(vmi, "sg_persist -o -G  --param-sark=12345678 /dev/sda",
				"Peripheral device type: disk")).To(BeTrue())
			Eventually(func(g Gomega) {
				g.Expect(
					checkResultCommand(vmi, "sg_persist -i -k /dev/sda", "1 registered reservation key follows:\r\n    0x12345678\r\n"),
				).To(BeTrue())
			}).
				Within(60 * time.Second).
				WithPolling(10 * time.Second).
				Should(Succeed())

			By("Requesting SCSI persistent reservation from the second VM")
			// The second VM should be able to see the reservation key used by the first VM and
			// the reservation with a new key should fail
			Expect(console.LoginToFedora(vmi2)).To(Succeed(), "Should be able to login to the Fedora VM")
			Expect(checkResultCommand(vmi2, "sg_persist -i -k /dev/sda",
				"1 registered reservation key follows:\r\n    0x12345678\r\n")).To(BeTrue())
			Expect(checkResultCommand(vmi2, "sg_persist -o -G  --param-sark=87654321 /dev/sda",
				"Reservation conflict")).To(BeTrue())
		})
	})

	Context("with PersistentReservation feature gate toggled", func() {
		It("should delete and recreate virt-handler", func() {
			config.DisableFeatureGate(featuregate.PersistentReservation)

			Eventually(func() []k8sv1.Container {
				ds, err := k8s.Client().AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.TODO(), "virt-handler", metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return ds.Spec.Template.Spec.Containers
			}, time.Minute*5, time.Second*2).ShouldNot(
				ContainElement((gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Name": Equal("pr-helper")},
				))))

			// Switching the PersistentReservation feature gate on/off
			// causes redeployment of all KubeVirt components.
			By("Ensuring all KubeVirt components are ready")
			testsuite.EnsureKubevirtReady()
		})

		Context("With multipath", func() {
			const mpathSocket = "/proc/1/root/run/multipathd.socket"
			BeforeEach(func() {
				// Check if mulitpathd socket exists on the nodes, if not simulate the existance by creating a mock socket
				nodes := libnode.GetAllSchedulableNodes(k8s.Client())
				for _, node := range nodes.Items {
					_, err := libnode.ExecuteCommandInVirtHandlerPod(node.Name, []string{"ls", mpathSocket})
					if err != nil {
						By(fmt.Sprintf("Create a fake mulitpathd.socket in node %s", node.Name))
						libnode.ExecuteCommandInVirtHandlerPod(node.Name, []string{"touch", mpathSocket})
						DeferCleanup(func() {
							_, err := libnode.ExecuteCommandInVirtHandlerPod(node.Name, []string{"rm", "-f", mpathSocket})
							Expect(err).ToNot(HaveOccurred())
						})
					}
				}
			})

			It("ensure multipath socket is bind mounted and available to the pr-helper daemon", func() {
				nodes := libnode.GetAllSchedulableNodes(k8s.Client())
				for _, node := range nodes.Items {
					Eventually(func(g Gomega) {
						output, err := libnode.ExecuteCommandInVirtHandlerPod(node.Name, []string{"cat", "/proc/mounts"})
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(strings.Count(output, "multipathd.socket")).Should(Equal(1),
							"the multipathd socket should be mounted only once")
					}).
						Within(20 * time.Second).
						WithPolling(1 * time.Second).
						Should(Succeed())
				}
			})
		})
	})

}))
