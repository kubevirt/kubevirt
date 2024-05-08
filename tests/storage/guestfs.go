package storage

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"
)

type fakeAttacher struct {
	// Channel to unblock the fake attacher for the console
	doneAttacher chan bool
	// Channel to unblock the goroutine for the guestfs command
	doneGuestfs chan bool
}

// fakeCreateAttacher simulates the attacher to the pod console. It has to block until the test terminates.
func (f *fakeAttacher) fakeCreateAttacher(client *guestfs.K8sClient, p *corev1.Pod, command string) error {
	<-f.doneAttacher
	return nil
}

func (f *fakeAttacher) closeChannel() {
	f.doneGuestfs <- true
}

var _ = SIGDescribe("[rfe_id:6364]Guestfs", func() {
	var (
		virtClient kubecli.KubevirtClient
		pvcClaim   string
		setGroup   bool
		testGroup  string
	)

	getGuestfsPodName := func(pvc string) string {
		return "libguestfs-tools-" + pvc
	}

	execCommandLibguestfsPod := func(podName, namespace string, c []string) (string, string, error) {
		pod, err := virtClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return exec.ExecuteCommandOnPodWithResults(pod, "libguestfs", c)
	}

	createPVCFilesystem := func(name, namespace string) {
		quantity, _ := resource.ParseQuantity("500Mi")
		_, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": quantity,
					},
				},
			},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

	}

	createFakeAttacher := func() *fakeAttacher {
		f := &fakeAttacher{}
		f.doneAttacher = make(chan bool, 1)
		f.doneGuestfs = make(chan bool, 1)
		guestfs.SetAttacher(f.fakeCreateAttacher)
		return f
	}

	guestfsWithSync := func(f *fakeAttacher, guestfsCmd *cobra.Command) {
		defer GinkgoRecover()
		errChan := make(chan error)
		go func() {
			errChan <- guestfsCmd.Execute()
		}()
		select {
		case <-f.doneGuestfs:
		case err := <-errChan:
			Expect(err).ToNot(HaveOccurred())
		}
		// Unblock the fake attacher
		f.doneAttacher <- true
	}

	runGuestfsOnPVC := func(f *fakeAttacher, pvcClaim, namespace string, options ...string) {
		podName := getGuestfsPodName(pvcClaim)
		o := append([]string{"guestfs", pvcClaim, "--namespace", namespace}, options...)
		if setGroup {
			o = append(o, "--fsGroup", testGroup)
		}
		guestfsCmd := clientcmd.NewVirtctlCommand(o...)
		go guestfsWithSync(f, guestfsCmd)
		// Waiting until the libguestfs pod is running
		Eventually(func() bool {
			pod, _ := virtClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
			ready := false
			for _, status := range pod.Status.ContainerStatuses {
				if status.State.Running != nil {
					return true
				}
			}
			return ready

		}, 90*time.Second, 2*time.Second).Should(BeTrue())
		// Verify that the appliance has been extracted before running any tests by checking the done file
		Eventually(func() bool {
			_, _, err := execCommandLibguestfsPod(podName, namespace, []string{"ls", "/usr/local/lib/guestfs/appliance/done"})
			if err != nil {
				return false
			}
			return true
		}, 30*time.Second, 2*time.Second).Should(BeTrue())

	}

	verifyCanRunOnFSPVC := func(podName, namespace string) {
		stdout, stderr, err := execCommandLibguestfsPod(podName, namespace, []string{"qemu-img", "create", "/disk/disk.img", "500M"})
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(ContainSubstring("Formatting"))
		Expect(err).ToNot(HaveOccurred())
		stdout, stderr, err = execCommandLibguestfsPod(podName, namespace, []string{"guestfish", "-a", "/disk/disk.img", "run"})
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(BeEmpty())
		Expect(err).ToNot(HaveOccurred())

	}

	Context("Run libguestfs on PVCs", func() {
		var f *fakeAttacher
		var ns string
		BeforeEach(func() {
			virtClient = kubevirt.Client()
			// TODO: Always setGroup to true until we have the ability to control how virtctl guestfs is run
			setGroup = true
			testGroup = "2000"
			f = createFakeAttacher()
			ns = testsuite.GetTestNamespace(nil)
		})

		AfterEach(func() {
			f.closeChannel()
		})

		// libguestfs-test-tool verifies the setup to run libguestfs-tools
		It("Should successfully run libguestfs-test-tool", func() {
			pvcClaim = "pvc-verify"
			createPVCFilesystem(pvcClaim, ns)
			runGuestfsOnPVC(f, pvcClaim, ns)
			output, _, err := execCommandLibguestfsPod(getGuestfsPodName(pvcClaim), ns, []string{"libguestfs-test-tool"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring("===== TEST FINISHED OK ====="))
		})

		It("[posneg:positive][test_id:6480]Should successfully run guestfs command on a filesystem-based PVC", func() {
			pvcClaim = "pvc-fs"
			createPVCFilesystem(pvcClaim, ns)
			runGuestfsOnPVC(f, pvcClaim, ns)
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim), ns)
		})

		It("[posneg:negative][test_id:6480]Should fail to run the guestfs command on a PVC in use", func() {
			pvcClaim = "pvc-fail-to-run-twice"
			createPVCFilesystem(pvcClaim, ns)
			runGuestfsOnPVC(f, pvcClaim, ns)
			options := []string{"guestfs",
				pvcClaim,
				"--namespace", ns}
			if setGroup {
				options = append(options, "--fsGroup", testGroup)
			}
			guestfsCmd := clientcmd.NewVirtctlCommand(options...)
			Expect(guestfsCmd.Execute()).To(HaveOccurred())
		})

		It("[posneg:positive][test_id:6479]Should successfully run guestfs command on a block-based PVC", func() {
			pvcClaim = "pvc-block"
			libstorage.CreateBlockPVC(pvcClaim, ns, "500Mi")
			runGuestfsOnPVC(f, pvcClaim, ns)
			stdout, stderr, err := execCommandLibguestfsPod(getGuestfsPodName(pvcClaim), ns, []string{"guestfish", "-a", "/dev/vda", "run"})
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal(""))
			Expect(err).ToNot(HaveOccurred())

		})
		It("Should successfully run guestfs command on a filesystem-based PVC setting the uid", func() {
			pvcClaim = "pvc-fs-with-different-uid"
			createPVCFilesystem(pvcClaim, ns)
			runGuestfsOnPVC(f, pvcClaim, ns, "--uid", "1002")
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim), ns)
		})
	})
	Context("Run libguestfs on PVCs with root", func() {
		var f *fakeAttacher
		BeforeEach(func() {
			virtClient = kubevirt.Client()
			setGroup = false
			f = createFakeAttacher()
		})

		AfterEach(func() {
			f.closeChannel()
		})
		It("Should successfully run guestfs command on a filesystem-based PVC with root", func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-fs-with-root"
			ns := testsuite.NamespacePrivileged
			createPVCFilesystem(pvcClaim, ns)
			runGuestfsOnPVC(f, pvcClaim, ns, "--root")
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim), ns)
		})
	})

})
