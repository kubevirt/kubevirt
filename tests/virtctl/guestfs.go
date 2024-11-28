/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors
 *
 */

package virtctl

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const testGroup = "2000"

var _ = Describe("[sig-storage][virtctl][rfe_id:6364]Guestfs", decorators.SigStorage, Label("guestfs"), func() {
	var (
		pvcClaim string
		setGroup bool
	)

	Context("Run libguestfs on PVCs", Label("guestfs"), func() {
		var (
			f  *fakeAttacher
			ns string
		)

		BeforeEach(func() {
			// TODO: Always setGroup to true until we have the ability to control how virtctl guestfs is run
			setGroup = true
			f = createFakeAttacher()
			ns = testsuite.GetTestNamespace(nil)
		})

		AfterEach(func() {
			f.closeChannel()
		})

		// libguestfs-test-tool verifies the setup to run libguestfs-tools
		It("Should successfully run libguestfs-test-tool", Label("guestfs", "FileSystem"), func() {
			pvcClaim = "pvc-verify"
			libstorage.CreateFSPVC(pvcClaim, ns, "500Mi", nil)
			runGuestfsOnPVC(f, pvcClaim, ns, setGroup)
			output, _, err := execCommandLibguestfsPod(getGuestfsPodName(pvcClaim), ns, []string{"libguestfs-test-tool"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring("===== TEST FINISHED OK ====="))
		})

		It("[posneg:positive][test_id:6480]Should successfully run guestfs command on a filesystem-based PVC", Label("guestfs", "FileSystem"), func() {
			pvcClaim = "pvc-fs"
			libstorage.CreateFSPVC(pvcClaim, ns, "500Mi", nil)
			runGuestfsOnPVC(f, pvcClaim, ns, setGroup)
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim), ns)
		})

		It("[posneg:negative][test_id:6480]Should fail to run the guestfs command on a PVC in use", Label("guestfs", "FileSystem"), func() {
			pvcClaim = "pvc-fail-to-run-twice"
			libstorage.CreateFSPVC(pvcClaim, ns, "500Mi", nil)
			runGuestfsOnPVC(f, pvcClaim, ns, setGroup)
			options := []string{"guestfs",
				pvcClaim,
				"--namespace", ns}
			if setGroup {
				options = append(options, "--fsGroup", testGroup)
			}
			guestfsCmd := clientcmd.NewVirtctlCommand(options...)
			Expect(guestfsCmd.Execute()).To(HaveOccurred())
		})

		It("[posneg:positive][test_id:6479]Should successfully run guestfs command on a block-based PVC", Label("guestfs", "Block"), func() {
			pvcClaim = "pvc-block"
			libstorage.CreateBlockPVC(pvcClaim, ns, "500Mi")
			runGuestfsOnPVC(f, pvcClaim, ns, setGroup)
			stdout, stderr, err := execCommandLibguestfsPod(getGuestfsPodName(pvcClaim), ns, []string{"guestfish", "-a", "/dev/vda", "run"})
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal(""))
			Expect(err).ToNot(HaveOccurred())

		})
		It("Should successfully run guestfs command on a filesystem-based PVC setting the uid", Label("guestfs", "FileSystem"), func() {
			pvcClaim = "pvc-fs-with-different-uid"
			libstorage.CreateFSPVC(pvcClaim, ns, "500Mi", nil)
			runGuestfsOnPVC(f, pvcClaim, ns, setGroup, "--uid", "1002")
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim), ns)
		})
	})

	Context("Run libguestfs on PVCs with root", func() {
		var f *fakeAttacher

		BeforeEach(func() {
			setGroup = false
			f = createFakeAttacher()
		})

		AfterEach(func() {
			f.closeChannel()
		})

		It("Should successfully run guestfs command on a filesystem-based PVC with root", Label("guestfs", "Filesystem"), func() {
			f := createFakeAttacher()
			defer f.closeChannel()
			pvcClaim = "pvc-fs-with-root"
			ns := testsuite.NamespacePrivileged
			libstorage.CreateFSPVC(pvcClaim, ns, "500Mi", nil)
			runGuestfsOnPVC(f, pvcClaim, ns, setGroup, "--root")
			verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim), ns)
		})
	})
})

func getGuestfsPodName(pvc string) string {
	return "libguestfs-tools-" + pvc
}

func execCommandLibguestfsPod(podName, namespace string, c []string) (string, string, error) {
	pod, err := kubevirt.Client().CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return exec.ExecuteCommandOnPodWithResults(pod, "libguestfs", c)
}

func runGuestfsOnPVC(f *fakeAttacher, pvcClaim, namespace string, setGroup bool, options ...string) {
	podName := getGuestfsPodName(pvcClaim)
	o := append([]string{"guestfs", pvcClaim, "--namespace", namespace}, options...)
	if setGroup {
		o = append(o, "--fsGroup", testGroup)
	}
	guestfsCmd := clientcmd.NewVirtctlCommand(o...)
	go guestfsWithSync(f, guestfsCmd)
	// Waiting until the libguestfs pod is running
	Eventually(func() bool {
		pod, _ := kubevirt.Client().CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
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
		return err == nil
	}, 30*time.Second, 2*time.Second).Should(BeTrue())
}

func guestfsWithSync(f *fakeAttacher, guestfsCmd *cobra.Command) {
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

func verifyCanRunOnFSPVC(podName, namespace string) {
	stdout, stderr, err := execCommandLibguestfsPod(podName, namespace, []string{"qemu-img", "create", "/disk/disk.img", "500M"})
	Expect(stderr).To(Equal(""))
	Expect(stdout).To(ContainSubstring("Formatting"))
	Expect(err).ToNot(HaveOccurred())
	stdout, stderr, err = execCommandLibguestfsPod(podName, namespace, []string{"guestfish", "-a", "/disk/disk.img", "run"})
	Expect(stderr).To(BeEmpty())
	Expect(stdout).To(BeEmpty())
	Expect(err).ToNot(HaveOccurred())
}

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

func createFakeAttacher() *fakeAttacher {
	f := &fakeAttacher{}
	f.doneAttacher = make(chan bool, 1)
	f.doneGuestfs = make(chan bool, 1)
	guestfs.CreateAttacherFunc = f.fakeCreateAttacher
	return f
}
