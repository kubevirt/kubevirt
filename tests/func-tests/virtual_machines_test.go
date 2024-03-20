package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvlibvmi "kubevirt.io/kubevirt/tests/libvmi"
	kvtutil "kubevirt.io/kubevirt/tests/util"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	timeout         = 10 * time.Minute
	pollingInterval = 10 * time.Second
)

var _ = Describe("[rfe_id:273][crit:critical][vendor:cnv-qe@redhat.com][level:system]Virtual Machine", Serial, func() {
	tests.FlagParse()

	var client kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		client, err = kubecli.GetKubevirtClient()
		kvtutil.PanicOnError(err)
		tests.BeforeEach()
	})

	It("[test_id:5696] should create, verify and delete VMIs", func() {
		vmiName := verifyVMICreation(client)
		verifyVMIRunning(client, vmiName)
		verifyVMIDeletion(client, vmiName)
	})
})

func verifyVMICreation(client kubecli.KubevirtClient) string {
	By("Creating VMI...")
	vmi := kvlibvmi.New(
		kvlibvmi.WithResourceMemory("128Mi"),
		kvlibvmi.WithInterface(kvlibvmi.InterfaceDeviceWithMasqueradeBinding()),
		kvlibvmi.WithNetwork(kubevirtcorev1.DefaultPodNetwork()),
	)
	EventuallyWithOffset(1, func() error {
		_, err := client.VirtualMachineInstance(kvtutil.NamespaceTestDefault).Create(context.Background(), vmi)
		return err
	}, timeout, pollingInterval).Should(Succeed(), "failed to create a vmi")
	return vmi.Name
}

func verifyVMIRunning(client kubecli.KubevirtClient, vmiName string) *kubevirtcorev1.VirtualMachineInstance {
	By("Verifying VMI is running")
	var vmi *kubevirtcorev1.VirtualMachineInstance
	EventuallyWithOffset(1, func(g Gomega) bool {
		var err error
		vmi, err = client.VirtualMachineInstance(kvtutil.NamespaceTestDefault).Get(context.Background(), vmiName, &k8smetav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).ToNot(Equal(kubevirtcorev1.Failed), "vmi scheduling failed: %s\n", vmi2JSON(vmi))
		return vmi.Status.Phase == kubevirtcorev1.Running
	}, timeout, pollingInterval).Should(BeTrue(), "failed to get the vmi Running")

	return vmi
}

func verifyVMIDeletion(client kubecli.KubevirtClient, vmiName string) {
	By("Verifying node placement of VMI")
	EventuallyWithOffset(1, func() error {
		return client.VirtualMachineInstance(kvtutil.NamespaceTestDefault).Delete(context.Background(), vmiName, &k8smetav1.DeleteOptions{})
	}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to delete a vmi")
}

func vmi2JSON(vmi *kubevirtcorev1.VirtualMachineInstance) string {
	buff := &bytes.Buffer{}
	enc := json.NewEncoder(buff)
	enc.SetIndent("", "  ")
	err := enc.Encode(vmi)
	if err != nil {
		GinkgoWriter.Println("failed to encode VMI. returning a golang struct string instead")
		return fmt.Sprintf("%#v", vmi)
	}

	return buff.String()
}
