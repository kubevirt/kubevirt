package reporter

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("VM log reporter", func() {
	Context("when collecting VM logs for a spec", func() {
		const namespace = "vm-log-reporter"

		var (
			virtClient *kubecli.MockKubevirtClient
			vmiClient  *kubecli.MockVirtualMachineInstanceInterface
			k8sClient  *k8sfake.Clientset
			vmi        virtv1.VirtualMachineInstance
			specReport types.SpecReport
			logOptions []*v1.PodLogOptions
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

			originalNamespaces := testsuite.TestNamespaces
			testsuite.TestNamespaces = []string{namespace}
			DeferCleanup(func() {
				testsuite.TestNamespaces = originalNamespaces
			})

			specReport = types.SpecReport{
				ContainerHierarchyTexts: []string{"VM log checks"},
				LeafNodeText:            "current spec",
				StartTime:               time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC),
			}
			vmi = virtv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-vmi",
					Namespace:   namespace,
					UID:         "test-vmi-uid",
					Annotations: map[string]string{testAnnotationKey: "VM log checks current spec"},
				},
			}
			k8sClient = k8sfake.NewSimpleClientset(
				&v1.Pod{ObjectMeta: metav1.ObjectMeta{
					Name:      "virt-launcher-test-vmi",
					Namespace: namespace,
					Labels:    map[string]string{virtv1.CreatedByLabel: "test-vmi-uid"},
				}},
				&v1.Pod{ObjectMeta: metav1.ObjectMeta{
					Name:      "virt-launcher-other-vmi",
					Namespace: namespace,
					Labels:    map[string]string{virtv1.CreatedByLabel: "other-vmi-uid"},
				}},
			)
			logOptions = nil
			k8sClient.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
				if action.GetSubresource() != "log" {
					return false, nil, nil
				}
				options := action.(k8stesting.GenericAction).GetValue().(*v1.PodLogOptions)
				logOptions = append(logOptions, options.DeepCopy())
				return true, &runtime.Unknown{Raw: []byte(`{"level":"error","msg":"unexpected launcher failure"}`)}, nil
			})
		})

		It("includes unexpected errors from VMIs created by the current spec", func() {
			virtClient.EXPECT().VirtualMachineInstance(namespace).Return(vmiClient)
			vmiClient.EXPECT().List(context.Background(), metav1.ListOptions{}).Return(
				&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{vmi}}, nil,
			)
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()

			Expect(getVMILogErrors(virtClient, &specReport)).To(ConsistOf(`[test-vmi] {"level":"error","msg":"unexpected launcher failure"}`))
			Expect(logOptions).To(ConsistOf(&v1.PodLogOptions{Container: "compute"}))
		})

		It("excludes VMIs created by another ordinary spec", func() {
			vmi.Annotations[testAnnotationKey] = "VM log checks previous spec"
			virtClient.EXPECT().VirtualMachineInstance(namespace).Return(vmiClient)
			vmiClient.EXPECT().List(context.Background(), metav1.ListOptions{}).Return(
				&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{vmi}}, nil,
			)

			Expect(getVMILogErrors(virtClient, &specReport)).To(BeEmpty())
			Expect(logOptions).To(BeEmpty())
		})

		It("includes errors since the current spec started for a reused VMI", func() {
			specReport.ContainerHierarchyLabels = [][]string{decorators.OncePerOrderedCleanup}
			vmi.Annotations[testAnnotationKey] = "VM log checks previous spec"
			virtClient.EXPECT().VirtualMachineInstance(namespace).Return(vmiClient)
			vmiClient.EXPECT().List(context.Background(), metav1.ListOptions{}).Return(
				&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{vmi}}, nil,
			)
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()

			Expect(getVMILogErrors(virtClient, &specReport)).To(ConsistOf(`[test-vmi] {"level":"error","msg":"unexpected launcher failure"}`))
			Expect(logOptions).To(ConsistOf(&v1.PodLogOptions{
				Container: "compute",
				SinceTime: &metav1.Time{Time: specReport.StartTime},
			}))
		})

		It("keeps startup logs for a VMI created by the current shared spec", func() {
			specReport.ContainerHierarchyLabels = [][]string{decorators.OncePerOrderedCleanup}
			virtClient.EXPECT().VirtualMachineInstance(namespace).Return(vmiClient)
			vmiClient.EXPECT().List(context.Background(), metav1.ListOptions{}).Return(
				&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{vmi}}, nil,
			)
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()

			Expect(getVMILogErrors(virtClient, &specReport)).To(ConsistOf(`[test-vmi] {"level":"error","msg":"unexpected launcher failure"}`))
			Expect(logOptions).To(ConsistOf(&v1.PodLogOptions{Container: "compute"}))
		})

		It("excludes unannotated VMIs from shared specs", func() {
			specReport.ContainerHierarchyLabels = [][]string{decorators.OncePerOrderedCleanup}
			vmi.Annotations = map[string]string{"unrelated-annotation": "present"}
			virtClient.EXPECT().VirtualMachineInstance(namespace).Return(vmiClient)
			vmiClient.EXPECT().List(context.Background(), metav1.ListOptions{}).Return(
				&virtv1.VirtualMachineInstanceList{Items: []virtv1.VirtualMachineInstance{vmi}}, nil,
			)

			Expect(getVMILogErrors(virtClient, &specReport)).To(BeEmpty())
			Expect(logOptions).To(BeEmpty())
		})
	})
})
