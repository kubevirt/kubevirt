package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	timeout         = 10 * time.Minute
	pollingInterval = 10 * time.Second
)

var _ = Describe("[rfe_id:273][crit:critical][vendor:cnv-qe@redhat.com][level:system]Virtual Machine", Serial, Label("vm"), func() {
	tests.FlagParse()

	var (
		cli client.Client
	)

	BeforeEach(func(ctx context.Context) {
		cli = tests.GetControllerRuntimeClient()
		tests.BeforeEach(ctx)
	})

	It("[test_id:5696] should create, verify and delete VMIs", Label("test_id:5696"), func(ctx context.Context) {
		vmiName := verifyVMICreation(ctx, cli)
		verifyVMIRunning(ctx, cli, vmiName)
		verifyVMIDeletion(ctx, cli, vmiName)
	})
})

func verifyVMICreation(ctx context.Context, cli client.Client) string {
	By("Creating VMI...")
	vmi := createVMIObject("testvmi")
	vmi.Spec.Domain.Resources.Requests = corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("128Mi")}
	vmi.Spec.Domain.Devices.Interfaces = []kubevirtcorev1.Interface{
		{
			Name: kubevirtcorev1.DefaultPodNetwork().Name,
			InterfaceBindingMethod: kubevirtcorev1.InterfaceBindingMethod{
				Masquerade: &kubevirtcorev1.InterfaceMasquerade{},
			},
		},
	}
	vmi.Spec.Networks = []kubevirtcorev1.Network{*kubevirtcorev1.DefaultPodNetwork()}

	EventuallyWithOffset(1, func() error {
		return cli.Create(ctx, vmi)
	}).WithTimeout(timeout).WithPolling(pollingInterval).Should(Succeed(), "failed to create a vmi")
	return vmi.Name
}

func verifyVMIRunning(ctx context.Context, cli client.Client, vmiName string) *kubevirtcorev1.VirtualMachineInstance {
	By("Verifying VMI is running")
	var vmi *kubevirtcorev1.VirtualMachineInstance
	EventuallyWithOffset(1, func(g Gomega, ctx context.Context) kubevirtcorev1.VirtualMachineInstancePhase {
		vmi = createVMIObject(vmiName)

		g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(vmi), vmi)).To(Succeed())
		Expect(vmi.Status.Phase).ToNot(Equal(kubevirtcorev1.Failed), "vmi scheduling failed: %s\n", vmi2JSON(vmi))

		return vmi.Status.Phase
	}).WithTimeout(timeout).WithPolling(pollingInterval).WithContext(ctx).Should(Equal(kubevirtcorev1.Running), "failed to get the vmi Running")

	return vmi
}

func verifyVMIDeletion(ctx context.Context, cli client.Client, vmiName string) {
	By("Verifying node placement of VMI")
	vmi := createVMIObject(vmiName)

	EventuallyWithOffset(1, func(ctx context.Context) error {
		return cli.Delete(ctx, vmi)
	}).WithTimeout(timeout).WithPolling(pollingInterval).WithContext(ctx).Should(Succeed(), "failed to delete a vmi")
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

func createVMIObject(vmiName string) *kubevirtcorev1.VirtualMachineInstance {
	return &kubevirtcorev1.VirtualMachineInstance{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      vmiName,
			Namespace: tests.TestNamespace,
		},
	}
}
