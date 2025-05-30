package rest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("EvacuateCancel Subresource API", func() {
	const (
		workerNode          = "test-worker-01"
		workerNodeWithTaint = "test-worker-02"
		taintKey            = "test-node-drain-key"
	)
	var (
		request  *restful.Request
		response *restful.Response

		virtClient *kubecli.MockKubevirtClient
		kubeClient *k8sfake.Clientset
		app        *SubresourceAPIApp

		kv = &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{},
					MigrationConfiguration: &v1.MigrationConfiguration{
						NodeDrainTaintKey: pointer.P(taintKey),
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeployed,
			},
		}
	)

	config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

	BeforeEach(func() {
		request = restful.NewRequest(&http.Request{})
		request.PathParameters()["name"] = testVMName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault
		recorder := httptest.NewRecorder()
		response = restful.NewResponse(recorder)

		backend := ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		Expect(err).ToNot(HaveOccurred())
		ctrl := gomock.NewController(GinkgoT())

		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = k8sfake.NewClientset(
			&k8scorev1.Node{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: workerNode,
				},
			},
			&k8scorev1.Node{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: workerNodeWithTaint,
				},
				Spec: k8scorev1.NodeSpec{
					Taints: []k8scorev1.Taint{
						{
							Key:    taintKey,
							Effect: k8scorev1.TaintEffectNoSchedule,
						},
					},
				},
			},
		)

		fakeKubevirtClients := fake.NewSimpleClientset().KubevirtV1()
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(fakeKubevirtClients.VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(fakeKubevirtClients.VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(fakeKubevirtClients.VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(fakeKubevirtClients.VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		app = NewSubresourceAPIApp(virtClient, backendPort, &tls.Config{InsecureSkipVerify: true}, config)
	})

	newEvacuateCancelBody := func(opts *v1.EvacuateCancelOptions) io.ReadCloser {
		optsJson, _ := json.Marshal(opts)
		return &readCloserWrapper{bytes.NewReader(optsJson)}
	}

	newInvalidBody := func() io.ReadCloser {
		return &readCloserWrapper{bytes.NewReader([]byte("invalid options"))}
	}

	newVMI := func(isEvacuated, withVM bool, node string) (*v1.VirtualMachineInstance, *v1.VirtualMachine) {
		vmi := libvmi.New(
			libvmi.WithName(testVMName),
			libvmi.WithNamespace(metav1.NamespaceDefault),
		)
		vmi.Status.NodeName = node
		if isEvacuated {
			vmi.Status.EvacuationNodeName = node
		}

		var vm *v1.VirtualMachine = nil
		if withVM {
			vm = libvmi.NewVirtualMachine(vmi)
			vm.Status.Created = true
			vm.UID = "test-vm-uid"
			vmi.OwnerReferences = append(vmi.OwnerReferences, metav1.OwnerReference{UID: vm.UID})
		}

		return vmi, vm
	}

	createVMI := func(isEvacuated, withVM bool, node string) (*v1.VirtualMachineInstance, *v1.VirtualMachine) {
		var err error
		vmi, vm := newVMI(isEvacuated, withVM, node)
		if vm != nil {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		return vmi, vm
	}

	When("Request to VirtualMachine", func() {
		DescribeTable("Should succeed", func(isEvacuated, dryRun bool) {
			vmi, _ := createVMI(isEvacuated, true, workerNode)

			if dryRun {
				request.Request.Body = newEvacuateCancelBody(&v1.EvacuateCancelOptions{DryRun: []string{metav1.DryRunAll}})
			}

			app.EvacuateCancelHandler(app.FetchVirtualMachineInstanceForVM)(request, response)
			Expect(response.StatusCode()).To(Equal(http.StatusOK))

			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			if !dryRun {
				Expect(vmi.Status.EvacuationNodeName).To(BeEmpty())
			}
		},
			Entry("should succeed because the VM is evacuated", true, false),
			Entry("should succeed because the VM is not evacuated", false, false),
			Entry("should succeed because the VM is evacuated with dry-run", true, true),
		)

		DescribeTable("Should fail because not found resource", func(vmExists bool) {
			if vmExists {
				_, vm := newVMI(false, true, workerNode)
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			app.EvacuateCancelHandler(app.FetchVirtualMachineInstanceForVM)(request, response)
			Expect(response.StatusCode()).To(Equal(http.StatusNotFound))
		},
			Entry("should fail because vm is not found", false),
			Entry("should fail because vmi is not found", true),
		)

		It("should fail because the node has taint", func() {
			createVMI(true, true, workerNodeWithTaint)
			app.EvacuateCancelHandler(app.FetchVirtualMachineInstanceForVM)(request, response)
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
		})

		It("Should fail because opts is invalid", func() {
			createVMI(true, true, workerNode)
			request.Request.Body = newInvalidBody()
			app.EvacuateCancelHandler(app.FetchVirtualMachineInstanceForVM)(request, response)
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
		})
	})

	When("Request to VirtualMachineInstance", func() {
		DescribeTable("Should succeed with the expected status code", func(isEvacuated, dryRun bool) {
			vmi, _ := createVMI(isEvacuated, false, workerNode)

			if dryRun {
				request.Request.Body = newEvacuateCancelBody(&v1.EvacuateCancelOptions{DryRun: []string{metav1.DryRunAll}})
			}

			app.EvacuateCancelHandler(app.FetchVirtualMachineInstance)(request, response)
			Expect(response.StatusCode()).To(Equal(http.StatusOK))

			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			if !dryRun {
				Expect(vmi.Status.EvacuationNodeName).To(BeEmpty())
			}
		},
			Entry("should succeed because the VMI is evacuated", true, false),
			Entry("should succeed because the VMI is not evacuated", false, false),
			Entry("should succeed because the VMI is evacuated with dry-run", true, true),
		)

		It("should fail because vmi is not found", func() {
			app.EvacuateCancelHandler(app.FetchVirtualMachineInstance)(request, response)
			Expect(response.StatusCode()).To(Equal(http.StatusNotFound))
		})

		It("should fail because the node has taint", func() {
			createVMI(false, false, workerNodeWithTaint)
			app.EvacuateCancelHandler(app.FetchVirtualMachineInstance)(request, response)
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
		})

		It("Should fail because opts is invalid", func() {
			createVMI(true, false, workerNode)
			request.Request.Body = newInvalidBody()
			app.EvacuateCancelHandler(app.FetchVirtualMachineInstance)(request, response)
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
		})
	})
})
