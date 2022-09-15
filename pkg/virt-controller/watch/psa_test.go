package watch

import (
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("PSA", func() {
	var (
		namespaceStore cache.Store
		client         *kubecli.MockKubevirtClient
		kubeClient     *fake.Clientset
		ctrl           *gomock.Controller
		notOnOpenshift = false
	)

	BeforeEach(func() {
		namespaceStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		ctrl = gomock.NewController(GinkgoT())
		client = kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = fake.NewSimpleClientset()
		client.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
	})

	Context("should patch namespace with enforce level", func() {
		var (
			onOpenshift          = true
			psaLabels            = HaveKeyWithValue(PSALabel, "privileged")
			psaLabelsOnOpenshift = And(HaveKeyWithValue(PSALabel, "privileged"), HaveKeyWithValue(OpenshiftPSAsync, "false"))
		)

		expectLabels := func(expectedLabels types.GomegaMatcher) {
			kubeClient.Fake.PrependReactor("patch", "namespaces",
				func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					patchAction, ok := action.(testing.PatchAction)
					Expect(ok).To(BeTrue())
					patchBytes := patchAction.GetPatch()
					namespace := &k8sv1.Namespace{}
					Expect(json.Unmarshal(patchBytes, namespace)).To(Succeed(), string(patchBytes))

					Expect(namespace.Labels).To(expectedLabels)
					return true, nil, nil
				})
		}

		DescribeTable("when label is missing", func(expectedLabels types.GomegaMatcher, onOpenshift bool) {
			expectLabels(expectedLabels)
			namespace := &k8sv1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			}
			Expect(namespaceStore.Add(namespace)).NotTo(HaveOccurred())

			Expect(escalateNamespace(namespaceStore, client, "test", onOpenshift)).To(Succeed())
		},
			Entry("on plain Kubernetes", psaLabels, notOnOpenshift),
			Entry("on Openshift", psaLabelsOnOpenshift, onOpenshift),
		)

		DescribeTable("when enforce label is not privileged", func(expectedLabels types.GomegaMatcher, onOpenshift bool) {
			expectLabels(expectedLabels)
			namespace := &k8sv1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						PSALabel: "restricted",
					},
				},
			}
			Expect(namespaceStore.Add(namespace)).NotTo(HaveOccurred())

			Expect(escalateNamespace(namespaceStore, client, "test", onOpenshift)).To(Succeed())
		},
			Entry("on plain Kubernetes", psaLabels, notOnOpenshift),
			Entry("on Openshift", psaLabelsOnOpenshift, onOpenshift),
		)
	})
	It("should not patch namespace when enforce label is set to privileged", func() {
		namespace := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Labels: map[string]string{
					PSALabel: "privileged",
				},
			},
		}
		Expect(namespaceStore.Add(namespace)).NotTo(HaveOccurred())
		kubeClient.Fake.PrependReactor("patch", "namespaces",
			func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				Expect("Patch namespaces is not expected").To(BeEmpty())
				return true, nil, nil
			})
		Expect(escalateNamespace(namespaceStore, client, "test", notOnOpenshift)).To(Succeed())
	})

})
