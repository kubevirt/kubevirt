package webhooks

import (
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Validating KubeVirtCreate Admitter", func() {
	var admitter *kubeVirtCreateAdmitter
	var kvInterface *kubecli.MockKubeVirtInterface

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kubevirtClient := kubecli.NewMockKubevirtClient(ctrl)
		kvInterface = kubecli.NewMockKubeVirtInterface(ctrl)
		kubevirtClient.EXPECT().KubeVirt(gomock.Any()).Return(kvInterface).AnyTimes()

		admitter = NewKubeVirtCreateAdmitter(kubevirtClient)
	})

	It("should prevent creating another Kubevirt resource", func() {
		alreadyExistingKv := v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "Different",
				Name:      "Existing",
			},
		}
		kvInterface.EXPECT().List(gomock.Any()).
			Return(&v1.KubeVirtList{Items: []v1.KubeVirt{alreadyExistingKv}}, nil).AnyTimes()

		newKv := v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name: "New",
			},
		}

		b, err := json.Marshal(newKv)
		Expect(err).ToNot(HaveOccurred())
		review := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Namespace: "test",
				Name:      "kubevirt",
				Object: runtime.RawExtension{
					Raw: b,
				},
			},
		}

		response := admitter.Admit(review)
		Expect(response.Allowed).To(BeFalse(), "Additional attempts to create Kubevirt should fail")
	})

	It("should allow creating new Kubevirt resource", func() {
		kvInterface.EXPECT().List(gomock.Any()).
			Return(&v1.KubeVirtList{Items: []v1.KubeVirt{}}, nil).AnyTimes()

		newKv := v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name: "New",
			},
		}

		b, err := json.Marshal(newKv)
		Expect(err).ToNot(HaveOccurred())
		review := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Namespace: "test",
				Name:      "kubevirt",
				Object: runtime.RawExtension{
					Raw: b,
				},
			},
		}

		response := admitter.Admit(review)
		Expect(response.Allowed).To(BeTrue(), "Create Kubevirt should be allowed")
	})
})
