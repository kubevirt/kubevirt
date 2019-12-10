package tests_test

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	testscore "kubevirt.io/kubevirt/tests"
)

var _ = Describe("Certificates", func() {
	flag.Parse()

	var stopChan chan struct{}

	BeforeEach(func() {
		tests.BeforeEach()
		stopChan = make(chan struct{})
	})

	AfterEach(func() {
		close(stopChan)
	})

	It("should rotate kubemacpool certificates", func() {
		By("getting the kubemacpool-service certificate")
		oldCert, err := GetCertForService("kubemacpool-service", testscore.KubeVirtInstallNamespace, "443")
		Expect(err).ToNot(HaveOccurred())
		Expect(oldCert).ToNot(BeEmpty())

		By("invoking the rotation script")
		Expect(RotateCeritifcates(testscore.KubeVirtInstallNamespace)).To(Succeed())
		By("waiting for all pods to become ready again")
		WaitForPodsToBecomeReady(testscore.KubeVirtInstallNamespace)

		By("getting the ceritifcate again after doing the rotation")
		newCert, err := GetCertForService("kubemacpool-service", testscore.KubeVirtInstallNamespace, "443")
		Expect(newCert).ToNot(BeEmpty())

		By("verifying that the ceritificate indeed changed")
		Expect(newCert).ToNot(Equal(oldCert))
	})

	It("should rotate cdi certificates", func() {
		By("getting the cdi-api certificate")
		oldCDIAPICert, err := GetCertForService("cdi-api", testscore.KubeVirtInstallNamespace, "443")
		Expect(err).ToNot(HaveOccurred())
		Expect(oldCDIAPICert).ToNot(BeEmpty())

		By("getting the cdi-uploadproxy certificate")
		oldCDIUploadproxyCert, err := GetCertForService("cdi-uploadproxy", testscore.KubeVirtInstallNamespace, "443")
		Expect(err).ToNot(HaveOccurred())
		Expect(oldCDIUploadproxyCert).ToNot(BeEmpty())

		By("invoking the rotation script")
		Expect(RotateCeritifcates(testscore.KubeVirtInstallNamespace)).To(Succeed())
		By("waiting for all pods to become ready again")
		WaitForPodsToBecomeReady(testscore.KubeVirtInstallNamespace)

		By("getting the ceritifcate again after doing the rotation")
		newCertAPICert, err := GetCertForService("cdi-api", testscore.KubeVirtInstallNamespace, "443")
		Expect(newCertAPICert).ToNot(BeEmpty())
		newCertUploadproxyCert, err := GetCertForService("cdi-uploadproxy", testscore.KubeVirtInstallNamespace, "443")
		Expect(newCertUploadproxyCert).ToNot(BeEmpty())

		By("verifying that the ceritificate indeed changed")
		Expect(newCertAPICert).ToNot(Equal(oldCDIAPICert))
		Expect(newCertUploadproxyCert).ToNot(Equal(oldCDIUploadproxyCert))
	})

	It("should rotate kubevirt certificates", func() {
		By("getting the virt-api certificate")
		oldVirtAPICert, err := GetCertForPod("kubevirt.io=virt-api", testscore.KubeVirtInstallNamespace, "8443")
		Expect(err).ToNot(HaveOccurred())
		Expect(oldVirtAPICert).ToNot(BeEmpty())

		By("getting the virt-controller certificate")
		oldVirtControllerCert, err := GetCertForPod("kubevirt.io=virt-controller", testscore.KubeVirtInstallNamespace, "8443")
		Expect(err).ToNot(HaveOccurred())
		Expect(oldVirtControllerCert).ToNot(BeEmpty())

		By("getting the virt-handler certificate")
		oldVirtHandlerCert, err := GetCertForPod("kubevirt.io=virt-handler", testscore.KubeVirtInstallNamespace, "8443")
		Expect(err).ToNot(HaveOccurred())
		Expect(oldVirtHandlerCert).ToNot(BeEmpty())

		By("invoking the rotation script")
		Expect(RotateCeritifcates(testscore.KubeVirtInstallNamespace)).To(Succeed())
		By("waiting for all pods to become ready again")
		WaitForPodsToBecomeReady(testscore.KubeVirtInstallNamespace)

		By("getting the ceritifcate again after doing the rotation")
		newVirtAPICert, err := GetCertForPod("kubevirt.io=virt-api", testscore.KubeVirtInstallNamespace, "8443")
		Expect(newVirtAPICert).ToNot(BeEmpty())
		newVirtControllerCert, err := GetCertForPod("kubevirt.io=virt-controller", testscore.KubeVirtInstallNamespace, "8443")
		Expect(newVirtControllerCert).ToNot(BeEmpty())
		newVirtHandlerCert, err := GetCertForPod("kubevirt.io=virt-handler", testscore.KubeVirtInstallNamespace, "8443")
		Expect(newVirtHandlerCert).ToNot(BeEmpty())

		By("verifying that the ceritificate indeed changed")
		Expect(newVirtAPICert).ToNot(Equal(oldVirtAPICert))
		Expect(newVirtControllerCert).ToNot(Equal(oldVirtControllerCert))
		Expect(newVirtHandlerCert).ToNot(Equal(oldVirtHandlerCert))
	})

	It("should rotate SSP certificates", func() {
		tests.SkipIfNotOpenShift("SSP only works on openshift")
		By("getting the virt-template-validator certificate")
		oldCert, err := GetCertForService("virt-template-validator", testscore.KubeVirtInstallNamespace, "443")
		Expect(err).ToNot(HaveOccurred())
		Expect(oldCert).ToNot(BeEmpty())

		By("invoking the rotation script")
		Expect(RotateCeritifcates(testscore.KubeVirtInstallNamespace)).To(Succeed())
		By("waiting for all pods to become ready again")
		WaitForPodsToBecomeReady(testscore.KubeVirtInstallNamespace)

		By("getting the ceritifcate again after doing the rotation")
		newCert, err := GetCertForService("virt-template-validator", testscore.KubeVirtInstallNamespace, "443")
		Expect(newCert).ToNot(BeEmpty())

		By("verifying that the ceritificate indeed changed")
		Expect(newCert).ToNot(Equal(oldCert))
	})

	Context("with an alpine VMI provided via CDI", func() {
		const (
			uploadProxyService   = "cdi-uploadproxy"
			uploadProxyPort      = 443
			localUploadProxyPort = 18443
			imagePath            = "/tmp/alpine.iso"
		)

		pvcName := "alpine-pvc"
		pvcSize := "100Mi"

		virtClient, err := kubecli.GetKubevirtClient()
		testscore.PanicOnError(err)

		BeforeEach(func() {
			By("Downloading alpine image")
			// alpine 3.7.0
			r, err := http.Get("https://storage.googleapis.com/builddeps/5a4b2588afd32e7024dd61d9558b77b03a4f3189cb4c9fc05e9e944fb780acdd")
			Expect(err).ToNot(HaveOccurred())
			defer r.Body.Close()

			file, err := os.Create(imagePath)
			Expect(err).ToNot(HaveOccurred())
			defer file.Close()

			_, err = io.Copy(file, r.Body)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should start the VMI after a certificate rotation", func() {
			By("Rotating the certs first")
			Expect(RotateCeritifcates(testscore.KubeVirtInstallNamespace)).To(Succeed())
			WaitForPodsToBecomeReady(testscore.KubeVirtInstallNamespace)
			jobType := tests.GetJobTypeEnvVar()
			storageClass := tests.KubeVirtStorageClassLocal
			if jobType == "prow" {
				storageClass = ""
			}

			By("Upload image")
			Eventually(func() error {
				stopChan := make(chan struct{})
				defer close(stopChan)
				portMapping := fmt.Sprintf("%d:%d", localUploadProxyPort, uploadProxyPort)
				err := ForwardPortsForService(uploadProxyService, testscore.KubeVirtInstallNamespace, stopChan, []string{portMapping})
				if err != nil {
					return err
				}

				virtctlCmd := testscore.NewRepeatableVirtctlCommand("image-upload",
					"--namespace", testscore.NamespaceTestDefault,
					"--image-path", imagePath,
					"--pvc-name", pvcName,
					"--pvc-size", pvcSize,
					"--uploadproxy-url", fmt.Sprintf("https://127.0.0.1:%d", localUploadProxyPort),
					"--wait-secs", "30",
					"--storage-class", storageClass,
					"--insecure")
				err = virtctlCmd()
				if err != nil {
					return fmt.Errorf("UploadImage Error: %+v\n", err)
				}
				return nil
			}, 40*time.Second, 5*time.Second).Should(Succeed())

			By("Start VM")
			vm := NewRandomVMWithPVC(pvcName)
			vm, err = virtClient.VirtualMachine(testscore.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			// Long timeout, since we don't know if virt-launcher is already pre-pulled
			Eventually(func() v1.VirtualMachineInstancePhase {
				vmi, err := virtClient.VirtualMachineInstance(testscore.NamespaceTestDefault).Get(vm.Name, &k8smetav1.GetOptions{})
				if errors.IsNotFound(err) {
					return ""
				}
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase
			}, 5*time.Minute, 2*time.Second).Should(Equal(v1.Running))
		})
		AfterEach(func() {
			err = os.Remove(imagePath)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func RotateCeritifcates(namespace string) error {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("tools/rotate-certs.sh -n %s", namespace))
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	return cmd.Run()
}

func GetCertForPod(labelSelector string, namespace string, port string) ([]byte, error) {
	randPort := strconv.Itoa(int(4321 + rand.Intn(6000)))
	cli, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	pods, err := cli.CoreV1().Pods(namespace).List(k8smetav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	Expect(pods.Items).ToNot(BeEmpty())

	stopChan := make(chan struct{})
	defer close(stopChan)
	err = testscore.ForwardPorts(&pods.Items[0], []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return GetCert(randPort), nil
}

func GetCertForService(name string, namespace string, port string) ([]byte, error) {
	randPort := strconv.Itoa(int(4321 + rand.Intn(6000)))
	stopChan := make(chan struct{})
	defer close(stopChan)
	err := ForwardPortsForService(name, namespace, stopChan, []string{fmt.Sprintf("%s:%s", randPort, port)})
	if err != nil {
		return nil, err
	}
	return GetCert(randPort), nil
}

func ForwardPortsForService(name string, namespace string, stopChan chan struct{}, ports []string) error {
	client, err := kubecli.GetKubevirtClient()
	testscore.PanicOnError(err)
	service, err := client.CoreV1().Services(namespace).Get(name, k8smetav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return tests.ForwardPortsFromService(service, ports, stopChan, 10*time.Second)
}

func WaitForPodsToBecomeReady(namespace string) {
	client, err := kubecli.GetKubevirtClient()
	testscore.PanicOnError(err)

	Eventually(func() error {
		pods, err := client.CoreV1().Pods(namespace).List(k8smetav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		notReady := []string{}
		for _, pod := range pods.Items {
			if pod.Status.Phase != k8sv1.PodRunning {
				notReady = append(notReady, pod.Name)
				continue
			}
			ready := false
			for _, conditions := range pod.Status.Conditions {
				if conditions.Type == k8sv1.PodReady && conditions.Status == k8sv1.ConditionTrue {
					ready = true
					break
				}
			}
			if !ready {
				notReady = append(notReady, pod.Name)
			}
		}
		if len(notReady) > 0 {
			return fmt.Errorf("Not ready Pods: %v", notReady)
		}
		return nil
	}, 20*time.Minute, 1*time.Second).Should(Succeed())
}

func GetCert(port string) []byte {
	var rawCert []byte
	mutex := &sync.Mutex{}
	conf := &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			mutex.Lock()
			defer mutex.Unlock()
			rawCert = rawCerts[0]
			return nil
		},
	}

	var cert []byte

	EventuallyWithOffset(1, func() []byte {
		conn, err := tls.Dial("tcp4", fmt.Sprintf("localhost:%s", port), conf)
		if err == nil {
			_ = conn.Close()
		}
		fmt.Println(err)
		mutex.Lock()
		defer mutex.Unlock()
		cert = make([]byte, len(rawCert))
		copy(cert, rawCert)
		return cert
	}, 40*time.Second, 1*time.Second).Should(Not(BeEmpty()))

	return cert
}

func NewRandomVMWithPVC(claimName string) *v1.VirtualMachine {
	vmi := testscore.NewRandomVMIWithPVC(claimName)
	t := true
	return &v1.VirtualMachine{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      vmi.Name,
			Namespace: vmi.Namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				Spec: vmi.Spec,
			},
			Running: &t,
		},
	}
}
