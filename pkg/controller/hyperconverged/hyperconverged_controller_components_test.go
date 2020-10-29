package hyperconverged

import (
	"os"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	consolev1 "github.com/openshift/api/console/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"context"
	"fmt"

	"k8s.io/client-go/tools/reference"
)

var _ = Describe("HyperConverged Components", func() {

	Context("Manage IMS Config", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("CONVERSION_CONTAINER", "new-conversion-container-value")
			os.Setenv("VMWARE_CONTAINER", "new-vmware-container-value")
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should error if environment vars not specified", func() {
			os.Unsetenv("CONVERSION_CONTAINER")
			os.Unsetenv("VMWARE_CONTAINER")

			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)
			res := r.ensureIMSConfig(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(BeNil())
		})

		It("should create if not present", func() {
			expectedResource := newIMSConfigForCR(hco, namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)
			res := r.ensureIMSConfig(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := newIMSConfigForCR(hco, namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			res := r.ensureIMSConfig(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		// in an ideal world HCO should be managing the whole config map,
		// now due to a bad design only a few values of this config map are
		// really managed by HCO while others are managed by other entities
		// TODO: fix this bad design splitting the config map into two distinct objects and reconcile the whole object here
		It("should (partially!!!) reconcile according to env values", func() {
			convk := "v2v-conversion-image"
			vmwarek := "kubevirt-vmware-image"
			updatableKeys := [...]string{convk, vmwarek}
			unupdatableKeyValues := map[string]string{
				"ext_key_1": "ext_value_1",
				"ext_key_2": "ext_value_2",
			}

			expectedResource := newIMSConfigForCR(hco, namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			outdatedResource := newIMSConfigForCR(hco, namespace)
			outdatedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", outdatedResource.Namespace, outdatedResource.Name)
			// values we should update
			outdatedResource.Data[convk] = "old-conversion-container-value-we-have-to-update"
			outdatedResource.Data[vmwarek] = "old-vmware-container-value-we-have-to-update"
			// add values we should not touch
			for k, v := range unupdatableKeyValues {
				outdatedResource.Data[k] = v
			}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, outdatedResource})
			r := initReconciler(cl)

			res := r.ensureIMSConfig(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())

			for _, k := range updatableKeys {
				Expect(foundResource.Data[k]).To(Not(Equal(outdatedResource.Data[k])))
				Expect(foundResource.Data[k]).To(Equal(expectedResource.Data[k]))
			}

			for k, v := range unupdatableKeyValues {
				Expect(foundResource.Data).To(HaveKeyWithValue(k, v))
			}

		})

	})

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
			r := initReconciler(cl)

			err := r.ensureConsoleCLIDownload(req)
			Expect(err).To(BeNil())

			foundResource := &consolev1.ConsoleCLIDownload{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := hco.NewConsoleCLIDownload()
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/consoleclidownloads/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			err := r.ensureConsoleCLIDownload(req)
			Expect(err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		DescribeTable("should update if something changed", func(modifiedResource *consolev1.ConsoleCLIDownload) {
			os.Setenv(hcoutil.KubevirtVersionEnvV, "100")
			cl := commonTestUtils.InitClient([]runtime.Object{modifiedResource})
			r := initReconciler(cl)
			err := r.ensureConsoleCLIDownload(req)
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
				}),
		)
	})
})
