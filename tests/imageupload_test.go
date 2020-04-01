package tests_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	uploadProxyService   = "svc/cdi-uploadproxy"
	uploadProxyPort      = 443
	localUploadProxyPort = 18443
	imagePath            = "/tmp/alpine.iso"
)

var _ = Describe("ImageUpload", func() {

	tests.FlagParse()

	namespace := tests.NamespaceTestDefault
	dvName := "alpine-dv"
	pvcSize := "100Mi"

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		By("Getting CDI HTTP import server pod")
		pods, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{LabelSelector: "kubevirt.io=cdi-http-import-server"})
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
	})

	Context("Upload an image and start a VMI with PVC", func() {
		It("Should succeed", func() {
			By("Setting up port forwarding")
			portMapping := fmt.Sprintf("%d:%d", localUploadProxyPort, uploadProxyPort)
			_, kubectlCmd, err := tests.CreateCommandWithNS(tests.ContainerizedDataImporterNamespace, "kubectl", "port-forward", uploadProxyService, portMapping)
			Expect(err).ToNot(HaveOccurred())

			err = kubectlCmd.Start()
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(2 * time.Second)
			Expect(kubectlCmd.ProcessState).To(BeNil())
			defer func() {
				kubectlCmd.Process.Kill()
				kubectlCmd.Wait()
			}()

			By("Upload image")

			virtctlCmd := tests.NewRepeatableVirtctlCommand("image-upload",
				"dv", dvName,
				"--namespace", namespace,
				"--image-path", imagePath,
				"--size", pvcSize,
				"--uploadproxy-url", fmt.Sprintf("https://127.0.0.1:%d", localUploadProxyPort),
				"--wait-secs", "30",
				"--storage-class", tests.Config.StorageClassLocal,
				"--insecure")
			err = virtctlCmd()
			if err != nil {
				fmt.Printf("UploadImage Error: %+v\n", err)
				Expect(err).ToNot(HaveOccurred())
			}

			By("Get DataVolume")
			_, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(dvName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Start VM")
			vmi := tests.NewRandomVMIWithPVC(dvName)
			vmi, err = virtClient.VirtualMachineInstance(namespace).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.ObjectMeta.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	AfterEach(func() {
		err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Delete(dvName, &metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())

		err = os.Remove(imagePath)
		Expect(err).ToNot(HaveOccurred())
	})
})
