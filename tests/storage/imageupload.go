package storage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/remotecommand"

	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/errorhandling"
	execute "kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
)

const (
	uploadProxyService   = "svc/cdi-uploadproxy"
	uploadProxyPort      = 443
	localUploadProxyPort = 18443
	imagePath            = "/tmp/alpine.iso"
	getDataVolume        = "Get DataVolume"
	getPVC               = "Get PVC"
	imageUploadCmd       = "image-upload"
	namespaceArg         = "--namespace"
	sizeArg              = "--size"
	insecureArg          = "--insecure"
)

var _ = SIGDescribe("[Serial]ImageUpload", Serial, func() {
	var kubectlCmd *exec.Cmd

	pvcSize := "100Mi"

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	BeforeEach(func() {
		By("Getting the disk image provider pod")
		pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=disks-images-provider"})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).ToNot(BeEmpty())

		stderr, err := copyFromPod(&pods.Items[0], "target", "/images/alpine/disk.img", imagePath)
		log.DefaultLogger().Info(stderr)
		Expect(err).ToNot(HaveOccurred())

		config, err := virtClient.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), "config", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if config.Status.UploadProxyURL == nil {
			By("Setting up port forwarding")
			portMapping := fmt.Sprintf("%d:%d", localUploadProxyPort, uploadProxyPort)
			_, kubectlCmd, err = clientcmd.CreateCommandWithNS(flags.ContainerizedDataImporterNamespace, "kubectl", "port-forward", uploadProxyService, portMapping)
			Expect(err).ToNot(HaveOccurred())

			err = kubectlCmd.Start()
			Expect(err).ToNot(HaveOccurred())
		}
	})

	validateDataVolume := func(targetName string, _ string) {
		if libstorage.IsDataVolumeGC(virtClient) {
			_, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return
		}
		By(getDataVolume)
		_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	validatePVC := func(targetName string, storageClass string) {
		By("Validate no DataVolume")
		_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By(getPVC)
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(*pvc.Spec.StorageClassName).To(Equal(storageClass))
	}

	Context("[storage-req] Upload an image and start a VMI with PVC", decorators.StorageReq, func() {
		DescribeTable("[test_id:4621] Should succeed", func(resource, targetName string, validateFunc func(string, string), startVM bool) {
			sc, exists := libstorage.GetRWOBlockStorageClass()
			if !exists {
				Skip("Skip test when RWOBlock storage class is not present")
			}

			By("Upload image")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				resource, targetName,
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--image-path", imagePath,
				sizeArg, pvcSize,
				"--storage-class", sc,
				"--block-volume",
				insecureArg)
			err := virtctlCmd()
			if err != nil {
				fmt.Printf("UploadImage Error: %+v\n", err)
				Expect(err).ToNot(HaveOccurred())
			}

			validateFunc(targetName, sc)

			if startVM {
				By("Start VM")
				vmi := tests.NewRandomVMIWithDataVolume(targetName)
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}()
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				)
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		},
			Entry("DataVolume", "dv", "alpine-dv-"+rand.String(12), validateDataVolume, true),
			Entry("PVC", "pvc", "alpine-pvc-"+rand.String(12), validatePVC, false),
		)
	})

	validateDataVolumeForceBind := func(targetName string) {
		if libstorage.IsDataVolumeGC(virtClient) {
			return
		}
		By(getDataVolume)
		dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		_, found := dv.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"]
		Expect(found).To(BeTrue())
	}

	validatePVCForceBind := func(targetName string) {
		By("Validate no DataVolume")
		_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By(getPVC)
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, found := pvc.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"]
		Expect(found).To(BeTrue())
	}

	Context("Create upload volume with force-bind flag", func() {
		DescribeTable("Should succeed", func(resource, targetName string, validateFunc func(string)) {
			storageClass, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists || !libstorage.IsStorageClassBindingModeWaitForFirstConsumer(storageClass) {
				Skip("Skip no wffc storage class available")
			}

			By("Upload image")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				resource, targetName,
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--image-path", imagePath,
				sizeArg, pvcSize,
				"--storage-class", storageClass,
				"--access-mode", "ReadWriteOnce",
				"--force-bind",
				insecureArg)

			Expect(virtctlCmd()).To(Succeed())
			validateFunc(targetName)
		},
			Entry("DataVolume", "dv", "alpine-dv-"+rand.String(12), validateDataVolumeForceBind),
			Entry("PVC", "pvc", "alpine-pvc-"+rand.String(12), validatePVCForceBind),
		)
	})

	Context("Create upload volume using volume-mode flag", func() {
		DescribeTable("Should succeed", func(volumeMode string) {
			sc, exists := libstorage.GetRWOBlockStorageClass()
			if !exists {
				Skip("Skip test when RWOBlock storage class is not present")
			}
			targetName := "alpine-dv-" + rand.String(12)

			By("Upload image")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				"dv", targetName,
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--image-path", imagePath,
				sizeArg, pvcSize,
				"--storage-class", sc,
				"--volume-mode", volumeMode,
				insecureArg)
			err := virtctlCmd()
			if err != nil {
				fmt.Printf("UploadImage Error: %+v\n", err)
				Expect(err).ToNot(HaveOccurred())
			}
			validateDataVolume(targetName, sc)
		},
			Entry("[test_id:10671]block volumeMode", "block"),
			Entry("[test_id:10672]filesystem volumeMode", "filesystem"),
		)

		It("[test_id:10674]Should fail with invalid volume-mode", func() {
			targetName := "alpine-bad-dv-" + rand.String(12)

			By("Upload image")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				"dv", targetName,
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--image-path", imagePath,
				sizeArg, pvcSize,
				"--volume-mode", "test",
				insecureArg)
			err := virtctlCmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid volume mode 'test'. Valid values are 'block' and 'filesystem'"))
		})
	})

	Context("Upload fails when DV is in WFFC/PendingPopulation phase", func() {
		It("but uploads after consumer is created", func() {
			storageClass, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists || !libstorage.IsStorageClassBindingModeWaitForFirstConsumer(storageClass) {
				Skip("Skip no wffc storage class available")
			}

			By("Upload image")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				"dv", "target-dv",
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--image-path", imagePath,
				sizeArg, pvcSize,
				"--storage-class", storageClass,
				"--access-mode", "ReadWriteOnce",
				insecureArg)

			err := virtctlCmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("make sure the PVC is Bound, or use force-bind flag"))

			By("Start VM")
			vmi := tests.NewRandomVMIWithDataVolume("target-dv")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}()

			By("Wait for DV to be in UploadReady phase")
			dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), "target-dv", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dataVolume, 240, matcher.BeInPhase(cdiv1.UploadReady))

			By("Upload image, now should succeed")
			virtctlCmd = clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				"dv", "target-dv",
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--image-path", imagePath,
				sizeArg, pvcSize,
				"--storage-class", storageClass,
				"--access-mode", "ReadWriteOnce",
				insecureArg)

			Expect(virtctlCmd()).To(Succeed())
			validateDataVolume("target-dv", storageClass)
		})
	})

	Context("Create upload archive volume", func() {
		var archivePath string

		BeforeEach(func() {
			archivePath = createArchive("archive", os.TempDir(), imagePath)
		})

		AfterEach(func() {
			err := os.Remove(archivePath)
			Expect(err).ToNot(HaveOccurred())
		})

		validateArchiveUpload := func(targetName string, uploadDV bool) {
			if uploadDV {
				if !libstorage.IsDataVolumeGC(virtClient) {
					By(getDataVolume)
					dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(dv.Spec.ContentType).To(Equal(cdiv1.DataVolumeArchive))
				}
			} else {
				By("Validate no DataVolume")
				_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
				Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
			}

			By(getPVC)
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			contentType, found := pvc.Annotations["cdi.kubevirt.io/storage.contentType"]
			Expect(found).To(BeTrue())
			Expect(contentType).To(Equal(string(cdiv1.DataVolumeArchive)))
		}

		DescribeTable("Should succeed", func(resource, targetName string, uploadDV bool) {
			By("Upload archive content")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				resource, targetName,
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--archive-path", archivePath,
				sizeArg, pvcSize,
				"--force-bind",
				insecureArg)

			Expect(virtctlCmd()).To(Succeed())
			validateArchiveUpload(targetName, uploadDV)
		},
			Entry("DataVolume", "dv", "alpine-archive-dv-"+rand.String(12), true),
			Entry("PVC", "pvc", "alpine-archive-pvc-"+rand.String(12), false),
		)
	})

	Context("Upload fails", func() {
		var archivePath string
		invalidStorageClass := "no-sc"

		BeforeEach(func() {
			archivePath = createArchive("archive", os.TempDir(), imagePath)
		})

		AfterEach(func() {
			err := os.Remove(archivePath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Upload fails creating a DV when using a non-existent storageClass", func() {
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				"dv", "alpine-archive-dv-"+rand.String(12),
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--archive-path", archivePath,
				"--storage-class", invalidStorageClass,
				sizeArg, pvcSize,
				"--force-bind",
				insecureArg)

			err := virtctlCmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storageclasses.storage.k8s.io \"no-sc\" not found"))
		})

		It("Upload fails creating a PVC when using a non-existent storageClass", func() {
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				"pvc", "alpine-archive-"+rand.String(12),
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--archive-path", archivePath,
				"--storage-class", invalidStorageClass,
				sizeArg, pvcSize,
				"--force-bind",
				insecureArg)

			err := virtctlCmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storageclasses.storage.k8s.io \"no-sc\" not found"))
		})

		It("Upload doesn't succeed when DV provisioning fails", func() {
			libstorage.CreateStorageClass(invalidStorageClass, nil)
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				"dv", "alpine-archive-dv-"+rand.String(12),
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--archive-path", archivePath,
				"--storage-class", invalidStorageClass,
				sizeArg, pvcSize,
				"--force-bind",
				insecureArg)

			err := virtctlCmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Claim not valid"))
			libstorage.DeleteStorageClass(invalidStorageClass)
		})

		It("Upload doesn't succeed when PVC provisioning fails", func() {
			libstorage.CreateStorageClass(invalidStorageClass, nil)
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(imageUploadCmd,
				"pvc", "alpine-archive-pvc-"+rand.String(12),
				namespaceArg, testsuite.GetTestNamespace(nil),
				"--archive-path", archivePath,
				"--storage-class", invalidStorageClass,
				sizeArg, pvcSize,
				"--force-bind",
				insecureArg)

			err := virtctlCmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Provisioning failed"))
			libstorage.DeleteStorageClass(invalidStorageClass)
		})
	})

	AfterEach(func() {
		if kubectlCmd != nil {
			Expect(kubectlCmd.Process.Kill()).To(Succeed())
			Expect(kubectlCmd.Wait()).To(Succeed())
		}

		err := os.Remove(imagePath)
		Expect(err).ToNot(HaveOccurred())
	})
})

func createArchive(targetFile, tgtDir string, sourceFilesNames ...string) string {
	tgtPath := filepath.Join(tgtDir, filepath.Base(targetFile)+".tar")
	tgtFile, err := os.Create(tgtPath)
	Expect(err).ToNot(HaveOccurred())
	defer errorhandling.SafelyCloseFile(tgtFile)

	tests.ArchiveToFile(tgtFile, sourceFilesNames...)

	return tgtPath
}

func copyFromPod(pod *k8sv1.Pod, containerName, sourceFile, targetFile string) (stderr string, err error) {
	var (
		stderrBuf bytes.Buffer
	)
	file, err := os.Create(targetFile)
	if err != nil {
		Expect(err).ToNot(HaveOccurred())
	}
	defer func() {
		if err := file.Close(); err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
	}()

	options := remotecommand.StreamOptions{
		Stdout: file,
		Stderr: &stderrBuf,
		Tty:    false,
	}
	err = execute.ExecuteCommandOnPodWithOptions(pod, containerName, []string{"cat", sourceFile}, options)
	return stderrBuf.String(), err
}
