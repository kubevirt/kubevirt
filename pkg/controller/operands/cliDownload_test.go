package operands

import (
	"context"
	"fmt"
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
	"os"
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
			expectedResource := hco.NewConsoleCLIDownload()
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
			expectedResource := hco.NewConsoleCLIDownload()
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

		DescribeTable("should update if something changed", func(modifiedResource *consolev1.ConsoleCLIDownload) {
			os.Setenv(hcoutil.KubevirtVersionEnvV, "100")
			cl := commonTestUtils.InitClient([]runtime.Object{modifiedResource})
			handler := &CLIDownloadHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			err := handler.Ensure(req)
			Expect(err).To(BeNil())
			expectedResource := hco.NewConsoleCLIDownload()
			key, err := client.ObjectKeyFromObject(expectedResource)
			Expect(err).ToNot(HaveOccurred())
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
	})
})
