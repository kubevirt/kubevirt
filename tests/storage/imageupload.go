package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests/util"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

const (
	uploadProxyService   = "svc/cdi-uploadproxy"
	uploadProxyPort      = 443
	localUploadProxyPort = 18443
	imagePath            = "/tmp/alpine.iso"
)

var _ = SIGDescribe("[Serial]ImageUpload", func() {
	var kubectlCmd *exec.Cmd

	pvcSize := "100Mi"

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error

		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	BeforeEach(func() {
		By("Getting the disk image provider pod")
		pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=disks-images-provider"})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).ToNot(BeEmpty())

		stderr, err := tests.CopyFromPod(virtClient, &pods.Items[0], "target", "/images/alpine/disk.img", imagePath)
		log.DefaultLogger().Info(stderr)
		Expect(err).ToNot(HaveOccurred())

		config, err := virtClient.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), "config", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if config.Status.UploadProxyURL == nil {
			By("Setting up port forwarding")
			portMapping := fmt.Sprintf("%d:%d", localUploadProxyPort, uploadProxyPort)
			_, kubectlCmd, err = tests.CreateCommandWithNS(flags.ContainerizedDataImporterNamespace, "kubectl", "port-forward", uploadProxyService, portMapping)
			Expect(err).ToNot(HaveOccurred())

			err = kubectlCmd.Start()
			Expect(err).ToNot(HaveOccurred())
		}
	})

	validateDataVolume := func(targetName string, _ string) {
		By("Get DataVolume")
		_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(util.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	deletePVC := func(targetName string) {
		err := virtClient.CoreV1().PersistentVolumeClaims((util.NamespaceTestDefault)).Delete(context.Background(), targetName, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			_, err = virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			Expect(err).ToNot(HaveOccurred())
			return false
		}, 90*time.Second, 2*time.Second).Should(BeTrue())

		Eventually(func() bool {
			pvList, err := virtClient.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, pv := range pvList.Items {
				if ref := pv.Spec.ClaimRef; ref != nil {
					if ref.Name == targetName {
						return false
					}
				}
			}
			return true
		}, 120*time.Second, 2*time.Second).Should(BeTrue())
	}

	deleteDataVolume := func(targetName string) {
		err := virtClient.CdiClient().CdiV1beta1().DataVolumes(util.NamespaceTestDefault).Delete(context.Background(), targetName, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(util.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			Expect(err).ToNot(HaveOccurred())
			return false
		}, 90*time.Second, 2*time.Second).Should(BeTrue())

		deletePVC(targetName)
	}

	validatePVC := func(targetName string, storageClass string) {
		By("Don't DataVolume")
		_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(util.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(errors.IsNotFound(err)).To(BeTrue())

		By("Get PVC")
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(*pvc.Spec.StorageClassName).To(Equal(storageClass))
	}

	Context("[rook-ceph] Upload an image and start a VMI with PVC", func() {
		DescribeTable("[test_id:4621] Should succeed", func(resource, targetName string, validateFunc func(string, string), deleteFunc func(string), startVM bool) {
			sc, exists := tests.GetCephStorageClass()
			if !exists {
				Skip("Skip OCS tests when Ceph is not present")
			}
			defer deleteFunc(targetName)

			By("Upload image")
			virtctlCmd := tests.NewRepeatableVirtctlCommand("image-upload",
				resource, targetName,
				"--namespace", util.NamespaceTestDefault,
				"--image-path", imagePath,
				"--size", pvcSize,
				"--storage-class", sc,
				"--block-volume",
				"--insecure")
			err := virtctlCmd()
			if err != nil {
				fmt.Printf("UploadImage Error: %+v\n", err)
				Expect(err).ToNot(HaveOccurred())
			}

			validateFunc(targetName, sc)

			if startVM {
				By("Start VM")
				vmi := tests.NewRandomVMIWithDataVolume(targetName)
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}()
				tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		},
			Entry("DataVolume", "dv", "alpine-dv-"+rand.String(12), validateDataVolume, deleteDataVolume, true),
			Entry("PVC", "pvc", "alpine-pvc-"+rand.String(12), validatePVC, deletePVC, false),
		)
	})

	validateDataVolumeForceBind := func(targetName string) {
		By("Get DataVolume")
		dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(util.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		_, found := dv.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"]
		Expect(found).To(BeTrue())
	}

	validatePVCForceBind := func(targetName string) {
		By("Don't DataVolume")
		_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(util.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(errors.IsNotFound(err)).To(BeTrue())

		By("Get PVC")
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, found := pvc.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"]
		Expect(found).To(BeTrue())
	}

	Context("Create upload volume with force-bind flag", func() {
		DescribeTable("Should succeed", func(resource, targetName string, validateFunc func(string), deleteFunc func(string)) {
			defer deleteFunc(targetName)

			By("Upload image")
			virtctlCmd := tests.NewRepeatableVirtctlCommand("image-upload",
				resource, targetName,
				"--namespace", util.NamespaceTestDefault,
				"--image-path", imagePath,
				"--size", pvcSize,
				"--force-bind",
				"--block-volume",
				"--insecure")

			Expect(virtctlCmd()).To(Succeed())
			validateFunc(targetName)
		},
			Entry("DataVolume", "dv", "alpine-dv-"+rand.String(12), validateDataVolumeForceBind, deleteDataVolume),
			Entry("PVC", "pvc", "alpine-pvc-"+rand.String(12), validatePVCForceBind, deletePVC),
		)
	})

	AfterEach(func() {
		if kubectlCmd != nil {
			kubectlCmd.Process.Kill()
			kubectlCmd.Wait()
		}

		err := os.Remove(imagePath)
		Expect(err).ToNot(HaveOccurred())
	})
})
