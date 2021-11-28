package storage

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/util"
)

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
	)
	execCommandLibguestfsPod := func(podName string, c []string) (string, string, error) {
		pod, err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).Get(context.Background(), podName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return tests.ExecuteCommandOnPodV2(virtClient, pod, "libguestfs", c)
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

	runGuestfsOnPVC := func(pvcClaim string) {
		podName := "libguestfs-tools-" + pvcClaim
		guestfsCmd := tests.NewVirtctlCommand("guestfs",
			pvcClaim,
			"--namespace", util.NamespaceTestDefault)
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
			_, _, err := execCommandLibguestfsPod(podName, []string{"ls", "/usr/local/lib/guestfs/done"})
			if err != nil {
				return false
			}
			return true
		}, 30*time.Second, 2*time.Second).Should(BeTrue())

	}

	Context("Run libguestfs on PVCs", func() {
		BeforeEach(func() {
			var err error
			virtClient, err = kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())

		}, 120)

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
			output, _, err := execCommandLibguestfsPod("libguestfs-tools-"+pvcClaim, []string{"libguestfs-test-tool"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring("===== TEST FINISHED OK ====="))

		})

		It("[posneg:positive][test_id:6480]Should successfully run guestfs command on a filesystem-based PVC", func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-fs"
			podName := "libguestfs-tools-" + pvcClaim
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(pvcClaim)
			stdout, stderr, err := execCommandLibguestfsPod(podName, []string{"qemu-img", "create", "/disk/disk.img", "500M"})
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(ContainSubstring("Formatting"))
			Expect(err).ToNot(HaveOccurred())
			stdout, stderr, err = execCommandLibguestfsPod(podName, []string{"guestfish", "-a", "/disk/disk.img", "run"})
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal(""))
			Expect(err).ToNot(HaveOccurred())

		})

		It("[posneg:negative][test_id:6480]Should fail to run the guestfs command on a PVC in use", func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-fail-to-run-twice"
			createPVCFilesystem(pvcClaim)
			runGuestfsOnPVC(pvcClaim)
			guestfsCmd := tests.NewVirtctlCommand("guestfs",
				pvcClaim,
				"--namespace", util.NamespaceTestDefault)
			Expect(guestfsCmd.Execute()).To(HaveOccurred())
		})

		It("[posneg:positive][test_id:6479]Should successfully run guestfs command on a block-based PVC", func() {
			f := createFakeAttacher()
			defer f.closeChannel()

			pvcClaim = "pvc-block"
			podName := "libguestfs-tools-" + pvcClaim
			size, _ := resource.ParseQuantity("500Mi")
			tests.CreateCephPVC(virtClient, pvcClaim, size)
			runGuestfsOnPVC(pvcClaim)
			stdout, stderr, err := execCommandLibguestfsPod(podName, []string{"guestfish", "-a", "/dev/vda", "run"})
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal(""))
			Expect(err).ToNot(HaveOccurred())

		})

	})
})
