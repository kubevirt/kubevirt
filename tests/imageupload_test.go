package tests_test

import (
	"flag"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	uploadProxyService   = "svc/cdi-uploadproxy"
	uploadProxyPort      = 443
	localUploadProxyPort = 18443
	imagePath            = "./vendor/kubevirt.io/containerized-data-importer/tests/images/cirros-qcow2.img"
)

var _ = Describe("ImageUpload", func() {

	flag.Parse()

	namespace := tests.NamespaceTestDefault
	pvcName := "cirros-pvc"
	pvcSize := "100Mi"

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	Context("Upload an image and start a VMI", func() {
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
				"--namespace", namespace,
				"--image-path", imagePath,
				"--pvc-name", pvcName,
				"--pvc-size", pvcSize,
				"--uploadproxy-url", fmt.Sprintf("https://127.0.0.1:%d", localUploadProxyPort),
				"--wait-secs", "30",
				"--storage-class", "local",
				"--insecure")
			err = virtctlCmd()
			if err != nil {
				fmt.Printf("UploadImage Error: %+v\n", err)
				Expect(err).ToNot(HaveOccurred())
			}

			By("Start VM")
			vmi := tests.NewRandomVMIWithPVC(pvcName)
			vmi, err = virtClient.VirtualMachineInstance(namespace).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.ObjectMeta.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
