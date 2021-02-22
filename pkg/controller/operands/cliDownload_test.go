package operands

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/meta"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CLI Download", func() {
	Context("ConsoleCLIDownload", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewConsoleCLIDownload(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := &CLIDownloadHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			err := handler.Ensure(req)
			Expect(err).To(BeNil())

			foundResource := &consolev1.ConsoleCLIDownload{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := NewConsoleCLIDownload(hco)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/consoleclidownloads/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := &CLIDownloadHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			err := handler.Ensure(req)
			Expect(err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("Should override links' url and text by the env vars", func() {
			origUrl := os.Getenv("VIRTCTL_DOWNLOAD_URL")
			origText := os.Getenv("VIRTCTL_DOWNLOAD_TEXT")
			defer os.Setenv("VIRTCTL_DOWNLOAD_URL", origUrl)
			defer os.Setenv("VIRTCTL_DOWNLOAD_TEXT", origText)

			_ = os.Setenv("VIRTCTL_DOWNLOAD_URL", "https://test-url:8443")
			_ = os.Setenv("VIRTCTL_DOWNLOAD_TEXT", "link text")
			expectedResource := NewConsoleCLIDownload(hco)
			Expect(expectedResource.Spec.Links).Should(HaveLen(1))
			Expect(expectedResource.Spec.Links).Should(ContainElement(consolev1.CLIDownloadLink{
				Text: "link text",
				Href: "https://test-url:8443",
			}))

		})

		DescribeTable("should update if something changed", func(modifiedResource *consolev1.ConsoleCLIDownload) {
			os.Setenv(hcoutil.KubevirtVersionEnvV, "100")
			cl := commonTestUtils.InitClient([]runtime.Object{modifiedResource})
			handler := &CLIDownloadHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			err := handler.Ensure(req)
			Expect(err).To(BeNil())
			expectedResource := NewConsoleCLIDownload(hco)
			key := client.ObjectKeyFromObject(expectedResource)
			foundResource := &consolev1.ConsoleCLIDownload{}
			Expect(cl.Get(context.TODO(), key, foundResource))
			Expect(foundResource.Spec.Links[0].Href).To(Equal(expectedResource.Spec.Links[0].Href))
			Expect(foundResource.Spec.Links[0].Text).To(Equal(expectedResource.Spec.Links[0].Text))
		},
			Entry("with modified download link",
				&consolev1.ConsoleCLIDownload{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "console.openshift.io/v1",
						Kind:       "ConsoleCLIDownload",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "virtctl-clidownloads-kubevirt-hyperconverged",
					},

					Spec: consolev1.ConsoleCLIDownloadSpec{
						Links: []consolev1.CLIDownloadLink{
							{
								Href: "https://dummy.url1.com",
								Text: "KubeVirt 100 release downloads",
							},
						},
					},
				}),
			Entry("with modified download text",
				&consolev1.ConsoleCLIDownload{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "console.openshift.io/v1",
						Kind:       "ConsoleCLIDownload",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "virtctl-clidownloads-kubevirt-hyperconverged",
					},
					Spec: consolev1.ConsoleCLIDownloadSpec{
						Links: []consolev1.CLIDownloadLink{
							{
								Href: "https://github.com/kubevirt/kubevirt/releases/100",
								Text: "dummy text 1",
							},
						},
					},
				},
			),
		)

		It("should return error if ConsoleCLIDownload was not found", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := &CLIDownloadHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}

			cl.InitiateCreateErrors(func(obj client.Object) error {
				if _, ok := obj.(*consolev1.ConsoleCLIDownload); ok {
					return &meta.NoResourceMatchError{}
				}
				return nil
			})
			err := handler.Ensure(req)
			Expect(err).To(HaveOccurred())
			Expect(meta.IsNoMatchError(err)).To(BeTrue())
		})

		It("should return error when update fails", func() {
			expectedResource := NewConsoleCLIDownload(hco)
			expectedResource.Spec.Links[0].Text = "wrong text"
			cl := commonTestUtils.InitClient([]runtime.Object{expectedResource})
			fakeErr := fmt.Errorf("fake ConsoleCLIDownload update error")
			cl.InitiateUpdateErrors(func(obj client.Object) error {
				if _, ok := obj.(*consolev1.ConsoleCLIDownload); ok {
					return fakeErr
				}
				return nil
			})
			handler := &CLIDownloadHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			err := handler.Ensure(req)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(fakeErr))
		})
	})
})
