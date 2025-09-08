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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/remotecommand"

	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	execute "kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	pvcSize = "100Mi"
)

var _ = Describe(SIG("[sig-storage]ImageUpload", decorators.SigStorage, Serial, func() {
	const (
		timeout      = 180
		randNameTail = 5
	)

	var (
		virtClient kubecli.KubevirtClient
		imagePath  string
		targetName string
		kubectlCmd *exec.Cmd
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		imagePath = copyAlpineDisk()
		targetName = "alpine-" + rand.String(randNameTail)

		config, err := virtClient.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), "config", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if config.Status.UploadProxyURL == nil {
			By("Setting up port forwarding")
			_, kubectlCmd, err = clientcmd.CreateCommandWithNS(
				flags.ContainerizedDataImporterNamespace, "kubectl", "port-forward", "svc/cdi-uploadproxy", "18443:443",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(kubectlCmd.Start()).To(Succeed())
		}
	})

	AfterEach(func() {
		if kubectlCmd != nil {
			Expect(kubectlCmd.Process.Kill()).To(Succeed())
			Expect(kubectlCmd.Wait()).To(Succeed())
		}
	})

	DescribeTable("[test_id:4621]Upload an imag start a VMI should succeed", decorators.RequiresBlockStorage,
		func(resource string, validateFn func(string, string), diskFn func(string, string, ...libvmi.DiskOption) libvmi.Option) {
			sc, exists := libstorage.GetRWOBlockStorageClass()
			if !exists {
				Fail("Fail test when RWOBlock storage class is not present")
			}

			By("Upload image")
			stdout, stderr, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), "virtctl", "image-upload",
				resource, targetName,
				"--image-path", imagePath,
				"--size", pvcSize,
				"--storage-class", sc,
				"--force-bind",
				"--volume-mode", "block",
				"--insecure",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(MatchRegexp(`\d{1,3}\.?\d{1,2}%`), "progress missing from stdout")
			Expect(stderr).To(BeEmpty())

			By("Validating uploaded image")
			validateFn(targetName, sc)

			By("Start VMI")
			vmi := libvmi.New(
				libvmi.WithMemoryRequest("256Mi"),
				diskFn("disk0", targetName),
			)
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(timeout),
			)
		},
		Entry("DataVolume", decorators.StorageCritical, "dv", validateDataVolume, libvmi.WithDataVolume),
		Entry("PVC", "pvc", validatePVC, libvmi.WithPersistentVolumeClaim),
	)

	DescribeTable("[test_id:11655]Create upload volume with force-bind flag should succeed",
		decorators.RequiresWFFCStorageClass, func(resource string, validateFn func(string)) {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists || !libstorage.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
				Fail("Fail no wffc storage class available")
			}

			By("Upload image")
			err := runImageUploadCmd(
				resource, targetName,
				"--image-path", imagePath,
				"--storage-class", sc,
				"--access-mode", "ReadWriteOnce",
				"--force-bind",
			)
			Expect(err).ToNot(HaveOccurred())

			By("Validating uploaded image")
			validateFn(targetName)
		},
		Entry("DataVolume", "dv", validateDataVolumeForceBind),
		Entry("PVC", "pvc", validatePVCForceBind),
	)

	DescribeTable("Create upload volume using volume-mode flag should succeed", func(volumeMode string, scFunc func() (string, bool)) {
		sc, exists := scFunc()
		if !exists {
			Fail(fmt.Sprintf("Fail test, %s storage class is not present", volumeMode))
		}

		By("Upload image")
		err := runImageUploadCmd(
			"dv", targetName,
			"--image-path", imagePath,
			"--storage-class", sc,
			"--force-bind",
			"--volume-mode", volumeMode,
		)
		Expect(err).ToNot(HaveOccurred())

		By("Validating uploaded image")
		validateDataVolume(targetName, sc)
	},
		Entry("[test_id:10671]block volumeMode", decorators.RequiresBlockStorage, "block", libstorage.GetRWOBlockStorageClass),
		Entry("[test_id:10672]filesystem volumeMode", "filesystem", libstorage.GetRWOFileSystemStorageClass),
	)

	It("[test_id:11656]Upload fails when DV is in WFFC/PendingPopulation phase but uploads after consumer is created",
		decorators.RequiresWFFCStorageClass, func() {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists || !libstorage.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
				Fail("Fail test, no wffc storage class available")
			}

			args := []string{
				"dv", targetName,
				"--image-path", imagePath,
				"--storage-class", sc,
				"--access-mode", "ReadWriteOnce",
			}

			By("Upload image")
			err := runImageUploadCmd(args...)
			Expect(err).To(MatchError(ContainSubstring("make sure the PVC is Bound, or use force-bind flag")))

			By("Start VMI")
			vmi := libvmi.New(
				libvmi.WithMemoryRequest("256Mi"),
				libvmi.WithDataVolume("disk0", targetName),
			)
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Wait for DV to be in UploadReady phase")
			dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vmi.Namespace).Get(context.Background(), targetName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dv, timeout, matcher.BeInPhase(cdiv1.UploadReady))

			By("Upload image, now should succeed")
			Expect(runImageUploadCmd(args...)).To(Succeed())

			By("Validating uploaded image")
			validateDataVolume(targetName, sc)
		})

	Context("Create upload archive volume", func() {
		var archivePath string

		BeforeEach(func() {
			By("Creating an archive")
			archivePath = createArchive(imagePath)
		})

		DescribeTable("[test_id:11657]Should succeed", func(resource string, uploadDV bool) {
			By("Upload archive content")
			err := runImageUploadCmd(
				resource, targetName,
				"--archive-path", archivePath,
				"--force-bind",
			)
			Expect(err).ToNot(HaveOccurred())

			if uploadDV {
				By("Get DataVolume")
				var dv *cdiv1.DataVolume
				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).
					Get(context.Background(), targetName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(dv.Spec.ContentType).To(Equal(cdiv1.DataVolumeArchive))
			} else {
				By("Validate no DataVolume")
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).
					Get(context.Background(), targetName, metav1.GetOptions{})
				Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
			}

			By("Get PVC")
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).
				Get(context.Background(), targetName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pvc.Annotations).To(HaveKeyWithValue("cdi.kubevirt.io/storage.contentType", string(cdiv1.DataVolumeArchive)))
		},
			Entry("DataVolume", "dv", true),
			Entry("PVC", "pvc", false),
		)

		DescribeTable("[test_id:11658]fails when provisioning fails", func(resource, expected string) {
			sc := "invalid-sc-" + rand.String(randNameTail)
			libstorage.CreateStorageClass(sc, nil)
			err := runImageUploadCmd(
				resource, "alpine-archive-"+rand.String(randNameTail),
				"--archive-path", archivePath,
				"--storage-class", sc,
				"--force-bind",
			)
			Expect(err).To(MatchError(ContainSubstring(expected)))
			libstorage.DeleteStorageClass(sc)
		},
			Entry("DataVolume", "dv", "Claim not valid"),
			Entry("PVC", "pvc", "Provisioning failed"),
		)
	})
}))

func copyAlpineDisk() string {
	virtClient := kubevirt.Client()
	By("Getting the disk image provider pod")
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).
		List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=disks-images-provider"})
	Expect(err).ToNot(HaveOccurred())
	Expect(pods.Items).ToNot(BeEmpty())

	path := filepath.Join(GinkgoT().TempDir(), "alpine.iso")
	file, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		Expect(file.Close()).To(Succeed())
	}()

	var stderr bytes.Buffer
	err = execute.ExecuteCommandOnPodWithOptions(&pods.Items[0], "target", []string{"cat", "/images/alpine/disk.img"},
		remotecommand.StreamOptions{
			Stdout: file,
			Stderr: &stderr,
			Tty:    false,
		},
	)
	Expect(err).ToNot(HaveOccurred())
	Expect(stderr.Len()).To(BeZero())

	return path
}

func createArchive(sourceFilesNames ...string) string {
	path := filepath.Join(GinkgoT().TempDir(), "archive.tar")
	file, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		Expect(file.Close()).To(Succeed())
	}()

	libstorage.ArchiveToFile(file, sourceFilesNames...)

	return path
}

func runImageUploadCmd(args ...string) error {
	_args := append([]string{
		"image-upload",
		"--namespace", testsuite.GetTestNamespace(nil),
		"--size", pvcSize,
		"--insecure",
	}, args...)
	return newRepeatableVirtctlCommand(_args...)()
}

func validateDataVolume(targetName, _ string) {
	virtClient := kubevirt.Client()
	By("Get DataVolume")
	_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).
		Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func validatePVC(targetName, sc string) {
	virtClient := kubevirt.Client()
	By("Validate no DataVolume")
	_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).
		Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

	By("Get PVC")
	pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).
		Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(*pvc.Spec.StorageClassName).To(Equal(sc))
}

func validateDataVolumeForceBind(targetName string) {
	virtClient := kubevirt.Client()
	By("Get DataVolume")
	dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).
		Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	_, found := dv.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"]
	Expect(found).To(BeTrue())
}

func validatePVCForceBind(targetName string) {
	virtClient := kubevirt.Client()
	By("Validate no DataVolume")
	_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).
		Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

	By("Get PVC")
	pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).
		Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	_, found := pvc.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"]
	Expect(found).To(BeTrue())
}
