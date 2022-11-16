package storage

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/util"
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

var _ = SIGDescribe("[rfe_id:6364][[Serial]Guestfs", func() {
	var (
		virtClient kubecli.KubevirtClient
		pvcClaim   string
		setGroup   bool
		testGroup  string
	)

	getGuestfsPodName := func(pvc string) string {
		return "libguestfs-tools-" + pvc
	}

	execCommandLibguestfsPod := func(podName string, c []string) (string, string, error) {
		pod, err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).Get(context.Background(), podName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return exec.ExecuteCommandOnPodWithResults(virtClient, pod, "libguestfs", c)
	}

	createPVCFilesystem := func(name string) {
		quantity, _ := resource.ParseQuantity("500Mi")
		_, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Create(context.Background(), &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
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

	runGuestfsOnPVC := func(f *fakeAttacher, pvcClaim string, options ...string) {
		podName := getGuestfsPodName(pvcClaim)
		o := append([]string{"guestfs", pvcClaim, "--namespace", util.NamespaceTestDefault}, options...)
		if setGroup {
			o = append(o, "--fsGroup", testGroup)
		}
		guestfsCmd := clientcmd.NewVirtctlCommand(o...)
		go guestfsWithSync(f, guestfsCmd)
		// Waiting until the libguestfs pod is ready
		Eventually(func() bool {
			pod, _ := virtClient.CoreV1().Pods(util.NamespaceTestDefault).Get(context.Background(), podName, metav1.GetOptions{})
			return tests.PodReady(pod) == corev1.ConditionTrue
		}, 90*time.Second, 2*time.Second).Should(BeTrue())

	}

	verifyCanRunOnFSPVC := func(podName string) {
		stdout, stderr, err := execCommandLibguestfsPod(podName, []string{"qemu-img", "create", "/disk/disk.img", "500M"})
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(ContainSubstring("Formatting"))
		Expect(err).ToNot(HaveOccurred())
		stdout, stderr, err = execCommandLibguestfsPod(podName, []string{"guestfish", "-a", "/disk/disk.img", "run"})
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(BeEmpty())
		Expect(err).ToNot(HaveOccurred())

	}

	Context("Run libguestfs on PVCs", func() {
		var f *fakeAttacher
		BeforeEach(func() {
			var err error
			virtClient, err = kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())
			// TODO: Always setGroup to true until we have the ability to control how virtctl guestfs is run
			setGroup = true
			testGroup = "2000"
			f = createFakeAttacher()
		})

		AfterEach(func() {
			f.closeChannel()
		})

		// libguestfs-test-tool verifies the setup to run libguestfs-tools
		It("Should successfully run libguestfs-test-tool", func() {
			pvcClaim = "pvc-verify"
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(f, pvcClaim)
			output, _, err := execCommandLibguestfsPod(getGuestfsPodName(pvcClaim), []string{"libguestfs-test-tool"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring("===== TEST FINISHED OK ====="))
		})

		It("[posneg:positive][test_id:6480]Should successfully run guestfs command on a filesystem-based PVC", func() {
			pvcClaim = "pvc-fs"
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(f, pvcClaim)
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim))
		})

		It("[posneg:negative][test_id:6480]Should fail to run the guestfs command on a PVC in use", func() {
			pvcClaim = "pvc-fail-to-run-twice"
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(f, pvcClaim)
			options := []string{"guestfs",
				pvcClaim,
				"--namespace", util.NamespaceTestDefault}
			if setGroup {
				options = append(options, "--fsGroup", testGroup)
			}
			guestfsCmd := clientcmd.NewVirtctlCommand(options...)
			Expect(guestfsCmd.Execute()).To(HaveOccurred())
		})

		It("[posneg:positive][test_id:6479]Should successfully run guestfs command on a block-based PVC", func() {
			pvcClaim = "pvc-block"
			libstorage.CreateBlockPVC(pvcClaim, "500Mi")
			runGuestfsOnPVC(f, pvcClaim)
			stdout, stderr, err := execCommandLibguestfsPod(getGuestfsPodName(pvcClaim), []string{"guestfish", "-a", "/dev/vda", "run"})
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal(""))
			Expect(err).ToNot(HaveOccurred())

		})
		It("Should successfully run guestfs command on a filesystem-based PVC setting the uid", func() {
			pvcClaim = "pvc-fs-with-different-uid"
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(f, pvcClaim, "--uid", "1002")
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim))
		})
	})
	Context("Run libguestfs on PVCs with root", func() {
		var f *fakeAttacher
		BeforeEach(func() {
			var err error
			virtClient, err = kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())
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
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(f, pvcClaim, "--root")
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim))
		})
	})

})
