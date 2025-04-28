package rest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/ptr"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("EvacuateCancel Subresource API", func() {
	const (
		workerNode = "test-worker-01"
		taintKey   = "test-node-drain-key"
	)

	var (
		request    *restful.Request
		response   *restful.Response
		virtClient *kubecli.MockKubevirtClient
		vmClient   *kubecli.MockVirtualMachineInterface
		vmiClient  *kubecli.MockVirtualMachineInstanceInterface
		kubeClient *fake.Clientset
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
						NodeDrainTaintKey: ptr.To(taintKey),
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
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
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		kubeClient = fake.NewClientset()

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachine("").Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		app = NewSubresourceAPIApp(virtClient, backendPort, &tls.Config{InsecureSkipVerify: true}, config)
	})

	newEvacuateCancelBody := func(opts *v1.EvacuateCancelOptions) io.ReadCloser {
		optsJson, _ := json.Marshal(opts)
		return &readCloserWrapper{bytes.NewReader(optsJson)}
	}

	patchVMI := func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
		vmiCopy := vmi.DeepCopy()
		vmiCopy.Status.EvacuationNodeName = ""
		return vmi
	}

	newNode := func() *k8scorev1.Node {
		return &k8scorev1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: workerNode,
			},
		}
	}

	It("Should fail because the node has taint [VMI]", func() {
		vmi := libvmi.New(
			libvmi.WithName(testVMName),
			libvmi.WithNamespace(metav1.NamespaceDefault),
		)
		vmi.Status.EvacuationNodeName = workerNode

		vmiClient.EXPECT().Get(context.Background(), vmi.Name, metav1.GetOptions{}).Return(vmi, nil).AnyTimes()

		kubeClient.Fake.PrependReactor("get", "nodes", func(action testing.Action) (bool, runtime.Object, error) {
			get, ok := action.(testing.GetAction)
			Expect(ok).To(BeTrue())
			Expect(get.GetName()).To(Equal(workerNode))

			node := newNode()
			node.Spec.Taints = append(node.Spec.Taints, k8scorev1.Taint{
				Key:    taintKey,
				Effect: k8scorev1.TaintEffectNoSchedule,
			})

			return true, node, nil
		})

		app.EvacuateCancelHandler(app.FetchVirtualMachineInstance)(request, response)

		Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
	})

	DescribeTable("Should succeed with the expected status code",
		func(evacuateOpts *v1.EvacuateCancelOptions, isVM, isEvacuated bool, getVMErr, getVMIErr, patchVMIErr error, expectStatusCode int) {
			vmi := libvmi.New(
				libvmi.WithName(testVMName),
				libvmi.WithNamespace(metav1.NamespaceDefault),
			)
			if isEvacuated {
				vmi.Status.EvacuationNodeName = workerNode
				kubeClient.Fake.PrependReactor("get", "nodes", func(action testing.Action) (bool, runtime.Object, error) {
					get, ok := action.(testing.GetAction)
					Expect(ok).To(BeTrue())
					Expect(get.GetName()).To(Equal(workerNode))

					return true, newNode(), nil
				})
			}

			if evacuateOpts != nil {
				request.Request.Body = newEvacuateCancelBody(evacuateOpts)
			}

			if isVM {
				vm := libvmi.NewVirtualMachine(vmi)
				vm.Status.Created = true
				vm.UID = "test-vm-uid"

				vmi.OwnerReferences = append(vmi.OwnerReferences, metav1.OwnerReference{UID: "test-vm-uid"})

				vmClient.EXPECT().Get(context.Background(), vm.Name, metav1.GetOptions{}).Return(vm, getVMErr).AnyTimes()
				vmiClient.EXPECT().Get(context.Background(), vmi.Name, metav1.GetOptions{}).Return(vmi, getVMIErr).AnyTimes()
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts metav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						Expect(opts.DryRun).To(BeEquivalentTo(evacuateOpts.DryRun))
						return patchVMI(vmi), patchVMIErr
					}).AnyTimes()

				app.EvacuateCancelHandler(app.FetchVirtualMachineInstanceForVM)(request, response)
			} else {
				vmiClient.EXPECT().Get(context.Background(), vmi.Name, metav1.GetOptions{}).Return(vmi, getVMIErr).AnyTimes()
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts metav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						Expect(opts.DryRun).To(BeEquivalentTo(evacuateOpts.DryRun))
						return patchVMI(vmi), patchVMIErr
					}).AnyTimes()

				app.EvacuateCancelHandler(app.FetchVirtualMachineInstance)(request, response)
			}

			Expect(response.StatusCode()).To(Equal(expectStatusCode))
		},
		Entry("should fail because the VMI is not found [VMI]",
			&v1.EvacuateCancelOptions{}, false, false, nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName), nil, http.StatusNotFound,
		),
		Entry("should fail due to an internal server error [VMI]",
			&v1.EvacuateCancelOptions{}, false, false, nil, fmt.Errorf("some internal error"), nil, http.StatusInternalServerError,
		),

		Entry("should fail because the VM is not found [VM]",
			&v1.EvacuateCancelOptions{}, true, false, errors.NewNotFound(v1.Resource("virtualmachine"), testVMName), nil, nil, http.StatusNotFound,
		),
		Entry("should fail due to an internal server error [VM]",
			&v1.EvacuateCancelOptions{}, true, false, fmt.Errorf("some internal error"), nil, nil, http.StatusInternalServerError,
		),
		Entry("should fail because the VM exists but the VMI is not found [VM]",
			&v1.EvacuateCancelOptions{}, true, false, nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName), nil, http.StatusNotFound,
		),
		Entry("should fail because the VM exists but there is an internal server error [VM]",
			&v1.EvacuateCancelOptions{}, true, false, nil, fmt.Errorf("some internal error"), nil, http.StatusInternalServerError,
		),

		Entry("should fail because the patch operation failed [VMI]",
			&v1.EvacuateCancelOptions{}, false, true, nil, nil, fmt.Errorf("patch failed"), http.StatusInternalServerError,
		),
		Entry("should fail because the patch operation failed [VM]",
			&v1.EvacuateCancelOptions{}, true, true, nil, nil, fmt.Errorf("patch failed"), http.StatusInternalServerError,
		),

		Entry("should succeed because the VMI is not evacuated [VMI]",
			&v1.EvacuateCancelOptions{}, false, false, nil, nil, nil, http.StatusOK,
		),
		Entry("should succeed because the VMI is evacuated [VMI]",
			&v1.EvacuateCancelOptions{}, false, true, nil, nil, nil, http.StatusOK,
		),

		Entry("should succeed because the VM is not evacuated [VM]",
			&v1.EvacuateCancelOptions{}, true, false, nil, nil, nil, http.StatusOK,
		),
		Entry("should succeed because the VM is evacuated [VM]",
			&v1.EvacuateCancelOptions{}, true, true, nil, nil, nil, http.StatusOK,
		),

		Entry("should succeed because the VMI is evacuated with dry-run [VMI]",
			&v1.EvacuateCancelOptions{DryRun: []string{metav1.DryRunAll}}, false, true, nil, nil, nil, http.StatusOK,
		),
		Entry("should succeed because the VM is evacuated with dry-run [VM]",
			&v1.EvacuateCancelOptions{DryRun: []string{metav1.DryRunAll}}, true, true, nil, nil, nil, http.StatusOK,
		),
	)
})
