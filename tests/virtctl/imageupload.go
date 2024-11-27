package virtctl

import (
	"bytes"
	"context"
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

var _ = Describe("[sig-storage][virtctl]ImageUpload", decorators.SigStorage, func() {
	var (
		virtClient kubecli.KubevirtClient
		imagePath  string
		kubectlCmd *exec.Cmd
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		imagePath = copyAlpineDisk()

		config, err := virtClient.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), "config", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if config.Status.UploadProxyURL == nil {
			By("Setting up port forwarding")
			_, kubectlCmd, err = clientcmd.CreateCommandWithNS(flags.ContainerizedDataImporterNamespace, "kubectl", "port-forward", "svc/cdi-uploadproxy", "18443:443")
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

	DescribeTable("[test_id:4621]Upload an image and start a VMI should succeed", func(resource string, validateFn func(string, string), diskFn func(string, string) libvmi.Option) {
		sc, exists := libstorage.GetRWOBlockStorageClass()
		if !exists {
			Fail("Fail test when RWOBlock storage class is not present")
		}

		By("Upload image")
		targetName := "alpine-" + rand.String(12)
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
			libvmi.WithResourceMemory("256Mi"),
			diskFn("disk0", targetName),
		)
		vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		libwait.WaitForSuccessfulVMIStart(vmi,
			libwait.WithFailOnWarnings(false),
			libwait.WithTimeout(180),
		)
	},
		Entry("DataVolume", "dv", validateDataVolume, libvmi.WithDataVolume),
		Entry("PVC", "pvc", validatePVC, libvmi.WithPersistentVolumeClaim),
	)

	DescribeTable("Create upload volume with force-bind flag should succeed", func(resource string, validateFn func(string)) {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists || !libstorage.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
			Fail("Fail no wffc storage class available")
		}

		By("Upload image")
		targetName := "alpine-" + rand.String(12)
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

	DescribeTable("Create upload volume using volume-mode flag should succeed", func(volumeMode string) {
		sc, exists := libstorage.GetRWOBlockStorageClass()
		if !exists {
			Fail("Fail test when RWOBlock storage class is not present")
		}

		By("Upload image")
		targetName := "alpine-" + rand.String(12)
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
		Entry("[test_id:10671]block volumeMode", "block"),
		Entry("[test_id:10672]filesystem volumeMode", "filesystem"),
	)

	It("Upload fails when DV is in WFFC/PendingPopulation phase but uploads after consumer is created", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists || !libstorage.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
			Skip("Skip no wffc storage class available")
		}

		targetName := "alpine-" + rand.String(12)
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
			libvmi.WithResourceMemory("256Mi"),
			libvmi.WithDataVolume("disk0", targetName),
		)
		vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Wait for DV to be in UploadReady phase")
		dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		libstorage.EventuallyDV(dv, 240, matcher.BeInPhase(cdiv1.UploadReady))

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

		DescribeTable("Should succeed", func(resource string, uploadDV bool) {
			By("Upload archive content")
			targetName := "alpine-" + rand.String(12)
			err := runImageUploadCmd(
				resource, targetName,
				"--archive-path", archivePath,
				"--force-bind",
			)
			Expect(err).ToNot(HaveOccurred())

			if uploadDV {
				if !libstorage.IsDataVolumeGC(virtClient) {
					By("Get DataVolume")
					dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(dv.Spec.ContentType).To(Equal(cdiv1.DataVolumeArchive))
				}
			} else {
				By("Validate no DataVolume")
				_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
				Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
			}

			By("Get PVC")
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pvc.Annotations).To(HaveKeyWithValue("cdi.kubevirt.io/storage.contentType", string(cdiv1.DataVolumeArchive)))
		},
			Entry("DataVolume", "dv", true),
			Entry("PVC", "pvc", false),
		)

		DescribeTable("fails when provisioning fails", func(resource, expected string) {
			sc := "invalid-sc-" + rand.String(12)
			libstorage.CreateStorageClass(sc, nil)
			err := runImageUploadCmd(
				resource, "alpine-archive-"+rand.String(12),
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
})

func copyAlpineDisk() string {
	virtClient := kubevirt.Client()
	By("Getting the disk image provider pod")
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=disks-images-provider"})
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
	return clientcmd.NewRepeatableVirtctlCommand(_args...)()
}

func validateDataVolume(targetName string, _ string) {
	virtClient := kubevirt.Client()
	if libstorage.IsDataVolumeGC(virtClient) {
		_, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return
	}
	By("Get DataVolume")
	_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func validatePVC(targetName string, sc string) {
	virtClient := kubevirt.Client()
	By("Validate no DataVolume")
	_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

	By("Get PVC")
	pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(*pvc.Spec.StorageClassName).To(Equal(sc))
}

func validateDataVolumeForceBind(targetName string) {
	virtClient := kubevirt.Client()
	if libstorage.IsDataVolumeGC(virtClient) {
		return
	}
	By("Get DataVolume")
	dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	_, found := dv.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"]
	Expect(found).To(BeTrue())
}

func validatePVCForceBind(targetName string) {
	virtClient := kubevirt.Client()
	By("Validate no DataVolume")
	_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

	By("Get PVC")
	pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), targetName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	_, found := pvc.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"]
	Expect(found).To(BeTrue())
}
