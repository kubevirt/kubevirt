package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"

	"kubevirt.io/client-go/kubecli"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	cdiImmediateBindAnnotation = "cdi.kubevirt.io/storage.bind.immediate.requested"
)

var _ = Describe("test dataImportCron", func() {
	tests.FlagParse()
	var cli kubecli.KubevirtClient
	ctx := context.TODO()

	cli, err := kubecli.GetKubevirtClient()
	Expect(cli).ToNot(BeNil())
	Expect(err).ToNot(HaveOccurred())

	BeforeEach(func() {
		tests.SkipIfNotOpenShift(cli, "DataImportCronTemplate")
	})

	AfterEach(func() {
		Eventually(func(g Gomega) {
			hc := tests.GetHCO(ctx, cli)

			// make sure there no user-defined DICT
			if len(hc.Spec.DataImportCronTemplates) > 0 {
				hc.APIVersion = "hco.kubevirt.io/v1beta1"
				hc.Kind = "HyperConverged"
				hc.Spec.DataImportCronTemplates = nil

				tests.UpdateHCORetry(ctx, cli, hc)
			}

		}).WithPolling(time.Second * 3).WithTimeout(time.Second * 60).Should(Succeed())

	})

	Context("test annotations", func() {

		It("should add missing annotation in the DICT", func() {
			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)

				hc.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{
					getDICT(),
				}

				tests.UpdateHCORetry(ctx, cli, hc)
				newHC := tests.GetHCO(ctx, cli)

				g.Expect(newHC.Spec.DataImportCronTemplates).To(HaveLen(1))
				g.Expect(newHC.Spec.DataImportCronTemplates[0].Annotations).To(HaveKeyWithValue(cdiImmediateBindAnnotation, "true"), "should add the missing annotation")
			}).WithPolling(time.Second * 3).WithTimeout(time.Second * 60).Should(Succeed())
		})

		It("should not change existing annotation in the DICT", func() {
			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)

				hc.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{
					getDICT(),
				}

				hc.Spec.DataImportCronTemplates[0].Annotations = map[string]string{
					cdiImmediateBindAnnotation: "false",
				}

				tests.UpdateHCORetry(ctx, cli, hc)
				newHC := tests.GetHCO(ctx, cli)

				g.Expect(newHC.Spec.DataImportCronTemplates).To(HaveLen(1))
				g.Expect(newHC.Spec.DataImportCronTemplates[0].Annotations).To(HaveKeyWithValue(cdiImmediateBindAnnotation, "false"), "should not change existing annotation")
			}).WithPolling(time.Second * 3).WithTimeout(time.Second * 60).Should(Succeed())
		})
	})

})

func getDICT() hcov1beta1.DataImportCronTemplate {
	gcType := cdiv1beta1.DataImportCronGarbageCollectOutdated
	pullMethod := cdiv1beta1.RegistryPullNode

	return hcov1beta1.DataImportCronTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "custom",
		},
		Spec: &cdiv1beta1.DataImportCronSpec{
			GarbageCollect:    &gcType,
			ManagedDataSource: "centos7",
			Schedule:          "18 1/12 * * *",
			Template: cdiv1beta1.DataVolume{
				Spec: cdiv1beta1.DataVolumeSpec{
					Source: &cdiv1beta1.DataVolumeSource{
						Registry: &cdiv1beta1.DataVolumeSourceRegistry{
							PullMethod: &pullMethod,
							URL:        pointer.String("docker://quay.io/containerdisks/centos:7-2009"),
						},
					},
					Storage: &cdiv1beta1.StorageSpec{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"storage": resource.MustParse("30Gi"),
							},
						},
					},
				},
			},
		},
	}
}
