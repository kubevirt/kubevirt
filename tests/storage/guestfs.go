package storage

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/util"
)

const libguestsTools = "libguestfs-tools-"

type fakeAttacher struct {
	done chan bool
}

// fakeCreateAttacher simulates the attacher to the pod console. It has to block until the test terminates.
func (f *fakeAttacher) fakeCreateAttacher(client *guestfs.K8sClient, p *corev1.Pod, command string) error {
	<-f.done
	return nil
}

func (f *fakeAttacher) closeChannel() {
	f.done <- true
}

var _ = SIGDescribe("[rfe_id:6364][[Serial]Guestfs", func() {
	var (
		virtClient kubecli.KubevirtClient
		pvcClaim   string
		setGroup   bool
		testGroup  string
	)
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
		f.done = make(chan bool, 1)
		guestfs.SetAttacher(f.fakeCreateAttacher)
		return f
	}

	runGuestfsOnPVC := func(pvcClaim string, options ...string) {
		podName := libguestsTools + pvcClaim
		o := append([]string{"guestfs", pvcClaim, "--namespace", util.NamespaceTestDefault}, options...)
		if setGroup {
			o = append(o, "--fsGroup", testGroup)
		}
		guestfsCmd := clientcmd.NewVirtctlCommand(o...)
		go func() {
			defer GinkgoRecover()
			Expect(guestfsCmd.Execute()).ToNot(HaveOccurred())
		}()
		// Waiting until the libguestfs pod is running
		Eventually(func() bool {
			pod, _ := virtClient.CoreV1().Pods(util.NamespaceTestDefault).Get(context.Background(), podName, metav1.GetOptions{})
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
			_, _, err := execCommandLibguestfsPod(podName, []string{"ls", "/usr/local/lib/guestfs/appliance/done"})
			if err != nil {
				return false
			}
			return true
		}, 30*time.Second, 2*time.Second).Should(BeTrue())

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
		BeforeEach(func() {
			var err error
			virtClient, err = kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())
			// TODO: Always setGroup to true until we have the ability to control how virtctl guestfs is run
			setGroup = true
			testGroup = "2000"
		})

		AfterEach(func() {
			err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Delete(context.Background(), pvcClaim, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		// libguestfs-test-tool verifies the setup to run libguestfs-tools
		It("Should successfully run libguestfs-test-tool", func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-verify"
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(pvcClaim)
			output, _, err := execCommandLibguestfsPod(libguestsTools+pvcClaim, []string{"libguestfs-test-tool"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring("===== TEST FINISHED OK ====="))

		})

		It("[posneg:positive][test_id:6480]Should successfully run guestfs command on a filesystem-based PVC", func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-fs"
			podName := libguestsTools + pvcClaim
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(pvcClaim)
			verifyCanRunOnFSPVC(podName)
		})

		It("[posneg:negative][test_id:6480]Should fail to run the guestfs command on a PVC in use", func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-fail-to-run-twice"
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(pvcClaim)
			guestfsCmd := clientcmd.NewVirtctlCommand("guestfs",
				pvcClaim,
				"--namespace", util.NamespaceTestDefault)
			Expect(guestfsCmd.Execute()).To(HaveOccurred())
		})

		It("[posneg:positive][test_id:6479]Should successfully run guestfs command on a block-based PVC", func() {
			f := createFakeAttacher()
			defer f.closeChannel()

			pvcClaim = "pvc-block"
			podName := libguestsTools + pvcClaim
			libstorage.CreateBlockPVC(pvcClaim, "500Mi")
			runGuestfsOnPVC(pvcClaim)
			stdout, stderr, err := execCommandLibguestfsPod(podName, []string{"guestfish", "-a", "/dev/vda", "run"})
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal(""))
			Expect(err).ToNot(HaveOccurred())

		})
		It("Should successfully run guestfs command on a filesystem-based PVC setting the uid", func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-fs-with-different-uid"
			podName := libguestsTools + pvcClaim
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(pvcClaim, "--uid", "1002")
			verifyCanRunOnFSPVC(podName)
		})
	})
	Context("Run libguestfs on PVCs with root", func() {
		BeforeEach(func() {
			var err error
			virtClient, err = kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())
			setGroup = false
		})

		AfterEach(func() {
			err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Delete(context.Background(), pvcClaim, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should successfully run guestfs command on a filesystem-based PVC with root", func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-fs-with-different-uid"
			podName := libguestsTools + pvcClaim
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(pvcClaim, "--root")
			verifyCanRunOnFSPVC(podName)
		})
	})

})
