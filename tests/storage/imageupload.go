package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
		tests.PanicOnError(err)
	})

	BeforeEach(func() {
		By("Getting CDI HTTP import server pod")
		pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=cdi-http-import-server"})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).ToNot(BeEmpty())

		stopChan := make(chan struct{})
		err = tests.ForwardPorts(&pods.Items[0], []string{"65432:80"}, stopChan, 10*time.Second)
		Expect(err).ToNot(HaveOccurred())

		By("Downloading alpine image")
		r, err := http.Get("http://localhost:65432/images/alpine.iso")
		Expect(err).ToNot(HaveOccurred())
		defer r.Body.Close()

		file, err := os.Create(imagePath)
		Expect(err).ToNot(HaveOccurred())
		defer file.Close()

		_, err = io.Copy(file, r.Body)
		Expect(err).ToNot(HaveOccurred())

		close(stopChan)

		By("Setting up port forwarding")
		portMapping := fmt.Sprintf("%d:%d", localUploadProxyPort, uploadProxyPort)
		_, kubectlCmd, err = tests.CreateCommandWithNS(flags.ContainerizedDataImporterNamespace, "kubectl", "port-forward", uploadProxyService, portMapping)
		Expect(err).ToNot(HaveOccurred())

		err = kubectlCmd.Start()
		Expect(err).ToNot(HaveOccurred())
	})

	validateDataVolume := func(targetName string, _ string) {
		By("Get DataVolume")
		_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(tests.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	deletePVC := func(targetName string) {
		err := virtClient.CoreV1().PersistentVolumeClaims((tests.NamespaceTestDefault)).Delete(context.Background(), targetName, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			_, err = virtClient.CoreV1().PersistentVolumeClaims(tests.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			Expect(err).ToNot(HaveOccurred())
			return false
		}, 90*time.Second, 2*time.Second).Should(BeTrue())
	}

	deleteDataVolume := func(targetName string) {
		err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(tests.NamespaceTestDefault).Delete(context.Background(), targetName, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			_, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(tests.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
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
		_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(tests.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(errors.IsNotFound(err)).To(BeTrue())

		By("Get PVC")
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(tests.NamespaceTestDefault).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(*pvc.Spec.StorageClassName).To(Equal(storageClass))
	}

	Context("Upload an image and start a VMI with PVC on rook-ceph", func() {
		DescribeTable("[test_id:4621] Should succeed", func(resource, targetName string, validateFunc func(string, string), deleteFunc func(string), startVM bool) {
			sc, exists := tests.GetCephStorageClass()
			if !exists {
				Skip("Skip OCS tests when Ceph is not present")
			}
			defer deleteFunc(targetName)

			By("Upload image")
			virtctlCmd := tests.NewRepeatableVirtctlCommand("image-upload",
				resource, targetName,
				"--namespace", tests.NamespaceTestDefault,
				"--image-path", imagePath,
				"--size", pvcSize,
				"--uploadproxy-url", fmt.Sprintf("https://127.0.0.1:%d", localUploadProxyPort),
				"--wait-secs", "60",
				"--storage-class", sc,
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
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}()
				tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		},
			Entry("DataVolume", "dv", "alpine-dv", validateDataVolume, deleteDataVolume, true),
			Entry("PVC", "pvc", "alpine-pvc", validatePVC, deletePVC, false),
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
