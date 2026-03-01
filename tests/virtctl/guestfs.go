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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const dummyPodName = "libguestfs-image-checker"

var _ = Describe(SIG("[sig-storage]Guestfs", decorators.SigStorage, func() {
	var (
		pvcClaim string
		done     chan struct{}
	)

	// fakeCreateAttacher simulates the attacher to the pod console. It has to block until the test terminates.
	fakeCreateAttacher := func(_ *guestfs.K8sClient, _ *corev1.Pod, _ string) error {
		<-done
		return nil
	}

	BeforeEach(func() {
		CheckGuestfsImageAvailability()
		guestfs.CreateAttacherFunc = fakeCreateAttacher
		const randNameTail = 5
		pvcClaim = "pvc-" + rand.String(randNameTail)
		done = make(chan struct{})
	})

	AfterEach(func() {
		CleanupGuestfsImageCheckPod()
		guestfs.CreateAttacherFunc = guestfs.CreateAttacher
		close(done)
	})

	Context("[rfe_id:6364]Run libguestfs on PVCs without root", func() {
		// TODO: Always setGroup to true until we have the ability to control how virtctl guestfs is run
		const setGroup = true

		Context("on a FS PVC", func() {
			BeforeEach(func() {
				libstorage.CreateFSPVC(pvcClaim, testsuite.GetTestNamespace(nil), "500Mi", libstorage.WithStorageProfile())
			})

			It("[test_id:11669]Should successfully run libguestfs-test-tool", func() {
				runGuestfsOnPVC(done, pvcClaim, testsuite.GetTestNamespace(nil), setGroup)
				stdout, _, err := execCommandLibguestfsPod(
					getGuestfsPodName(pvcClaim), testsuite.GetTestNamespace(nil), []string{"libguestfs-test-tool"},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("===== TEST FINISHED OK ====="))
			})

			DescribeTable("[posneg:positive]Should successfully run guestfs command on a filesystem-based PVC", func(extraArgs ...string) {
				runGuestfsOnPVC(done, pvcClaim, testsuite.GetTestNamespace(nil), setGroup, extraArgs...)
				verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim), testsuite.GetTestNamespace(nil))
			},
				Entry("[test_id:6480]without extra arguments"),
				Entry("[test_id:11670]setting the uid", "--uid", "1002"),
			)

			It("[posneg:negative][test_id:11671]Should fail to run the guestfs command on a PVC in use", func() {
				runGuestfsOnPVC(done, pvcClaim, testsuite.GetTestNamespace(nil), setGroup)
				cmd := guestfsCmd(pvcClaim, testsuite.GetTestNamespace(nil), setGroup)
				Expect(cmd()).To(MatchError(fmt.Sprintf("PVC %s is used by another pod", pvcClaim)))
			})
		})

		It("[posneg:positive][test_id:6479]Should successfully run guestfs command on a block-based PVC",
			decorators.Conformance, decorators.RequiresBlockStorage, func() {
				libstorage.CreateBlockPVC(pvcClaim, testsuite.GetTestNamespace(nil), "500Mi", libstorage.WithStorageProfile())
				runGuestfsOnPVC(done, pvcClaim, testsuite.GetTestNamespace(nil), setGroup)
				stdout, stderr, err := execCommandLibguestfsPod(
					getGuestfsPodName(pvcClaim), testsuite.GetTestNamespace(nil), []string{"guestfish", "-a", "/dev/vda", "run"},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(stderr).To(BeEmpty())
				Expect(stdout).To(BeEmpty())
			})
	})

	It("[rfe_id:6364]Should successfully run guestfs command on a filesystem-based PVC with root", func() {
		libstorage.CreateFSPVC(pvcClaim, testsuite.NamespacePrivileged, "500Mi", libstorage.WithStorageProfile())
		runGuestfsOnPVC(done, pvcClaim, testsuite.NamespacePrivileged, false, "--root")
		verifyCanRunOnFSPVC(getGuestfsPodName(pvcClaim), testsuite.NamespacePrivileged)
	})
}))

func getGuestfsPodName(pvc string) string {
	return "libguestfs-tools-" + pvc
}

func execCommandLibguestfsPod(podName, namespace string, args []string) (stdout, stderr string, err error) {
	pod, err := kubevirt.Client().CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return exec.ExecuteCommandOnPodWithResults(pod, "libguestfs", args)
}

func guestfsCmd(pvcClaim, namespace string, setGroup bool, extraArgs ...string) func() error {
	args := append([]string{
		"guestfs", pvcClaim,
		"--namespace", namespace,
	}, extraArgs...)
	if setGroup {
		const testGroup = "2000"
		args = append(args, "--fsGroup", testGroup)
	}
	return newRepeatableVirtctlCommand(args...)
}

func runGuestfsOnPVC(done chan struct{}, pvcClaim, namespace string, setGroup bool, extraArgs ...string) {
	go guestfsWithSync(done, guestfsCmd(pvcClaim, namespace, setGroup, extraArgs...))

	podName := getGuestfsPodName(pvcClaim)
	// Waiting until the libguestfs pod is running
	Eventually(func(g Gomega) {
		pod, err := kubevirt.Client().CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(pod).To(matcher.HaveConditionTrue(corev1.ContainersReady))
	}, 90*time.Second, 2*time.Second).Should(Succeed())
	// Verify that the appliance has been extracted before running any tests by checking the done file
	Eventually(func(g Gomega) {
		_, _, err := execCommandLibguestfsPod(podName, namespace, []string{"ls", "/usr/local/lib/guestfs/appliance/done"})
		g.Expect(err).ToNot(HaveOccurred())
	}, 30*time.Second, 2*time.Second).Should(Succeed())
}

func guestfsWithSync(done chan struct{}, cmd func() error) {
	defer GinkgoRecover()
	errChan := make(chan error)
	go func() {
		errChan <- cmd()
	}()
	select {
	case <-done:
	case err := <-errChan:
		Expect(err).ToNot(HaveOccurred())
	}
}

func verifyCanRunOnFSPVC(podName, namespace string) {
	stdout, stderr, err := execCommandLibguestfsPod(podName, namespace, []string{"qemu-img", "create", "/disk/disk.img", "500M"})
	Expect(stderr).To(BeEmpty())
	Expect(stdout).To(ContainSubstring("Formatting"))
	Expect(err).ToNot(HaveOccurred())
	stdout, stderr, err = execCommandLibguestfsPod(podName, namespace, []string{"guestfish", "-a", "/disk/disk.img", "run"})
	Expect(stderr).To(BeEmpty())
	Expect(stdout).To(BeEmpty())
	Expect(err).ToNot(HaveOccurred())
}

func CleanupGuestfsImageCheckPod() {
	kubeClient := kubevirt.Client()
	err := kubeClient.CoreV1().Pods(testsuite.GetTestNamespace(nil)).Delete(context.Background(), dummyPodName, metav1.DeleteOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return
	}
	Expect(err).ToNot(HaveOccurred())
}

// check if the libguestfs-tools image is available for test
func CheckGuestfsImageAvailability() {
	podYaml := ""
	kubeClient := kubevirt.Client()
	guestfsImage, err := guestfs.SetImage(kubeClient)
	Expect(err).ToNot(HaveOccurred())
	Expect(guestfsImage).ToNot(BeEmpty())
	namespace := testsuite.GetTestNamespace(nil)
	pod := CreateDummyPod(guestfsImage, namespace)
	_, err = kubeClient.CoreV1().Pods(testsuite.GetTestNamespace(nil)).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(func(g Gomega) {
		pod, err := kubevirt.Client().CoreV1().Pods(namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		podBytes, err := yaml.Marshal(pod)
		g.Expect(err).ToNot(HaveOccurred())
		podYaml = string(podBytes)
		podYaml = fmt.Sprintf("[debug] failed pod yaml: \n%s\n---\n", podYaml)
		g.Expect(pod).To(matcher.HaveConditionTrue(corev1.ContainersReady), podYaml)
	}, 2*time.Minute, 2*time.Second).Should(Succeed())
}

func CreateDummyPod(img string, ns string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dummyPodName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "testcontainer",
					Image:           img,
					Command:         []string{"sleep", "120"},
					ImagePullPolicy: corev1.PullAlways,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	return pod
}
