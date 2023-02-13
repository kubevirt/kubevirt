package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"

	"kubevirt.io/client-go/kubecli"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests/flags"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	cdiImmediateBindAnnotation = "cdi.kubevirt.io/storage.bind.immediate.requested"
)

var _ = Describe("test dataImportCron", func() {
	tests.FlagParse()
	var cli kubecli.KubevirtClient

	s := scheme.Scheme
	_ = hcov1beta1.AddToScheme(s)
	s.AddKnownTypes(hcov1beta1.SchemeGroupVersion)

	BeforeEach(func() {
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		tests.SkipIfNotOpenShift(virtCli, "DataImportCronTemplate")

		cli, err = kubecli.GetKubevirtClientFromRESTConfig(virtCli.Config())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Eventually(func(g Gomega) {
			hc := &hcov1beta1.HyperConverged{}
			Expect(cli.RestClient().
				Get().
				Resource("hyperconvergeds").
				Name("kubevirt-hyperconverged").
				Namespace(flags.KubeVirtInstallNamespace).
				AbsPath("/apis", hcov1beta1.SchemeGroupVersion.Group, hcov1beta1.SchemeGroupVersion.Version).
				Timeout(10 * time.Second).
				Do(context.TODO()).
				Into(hc),
			).To(Succeed())

			// make sure there no user-defined DICT
			if len(hc.Spec.DataImportCronTemplates) > 0 {
				hc.APIVersion = "hco.kubevirt.io/v1beta1"
				hc.Kind = "HyperConverged"
				hc.Spec.DataImportCronTemplates = nil

				res := cli.RestClient().Put().
					Resource("hyperconvergeds").
					Name(hcov1beta1.HyperConvergedName).
					Namespace(flags.KubeVirtInstallNamespace).
					AbsPath("/apis", hcov1beta1.SchemeGroupVersion.Group, hcov1beta1.SchemeGroupVersion.Version).
					Timeout(10 * time.Second).
					Body(hc).Do(context.TODO())

				g.Expect(res.Error()).ToNot(HaveOccurred())
			}

		}).WithPolling(time.Second * 3).WithTimeout(time.Second * 60).Should(Succeed())

	})

	Context("test annotations", func() {

		It("should add missing annotation in the DICT", func() {
			Eventually(func(g Gomega) {
				hc := &hcov1beta1.HyperConverged{}
				Expect(cli.RestClient().
					Get().
					Resource("hyperconvergeds").
					Name("kubevirt-hyperconverged").
					Namespace(flags.KubeVirtInstallNamespace).
					AbsPath("/apis", hcov1beta1.SchemeGroupVersion.Group, hcov1beta1.SchemeGroupVersion.Version).
					Timeout(10 * time.Second).
					Do(context.TODO()).
					Into(hc),
				).To(Succeed())

				hc.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{
					getDICT(),
				}

				hc.APIVersion = "hco.kubevirt.io/v1beta1"
				hc.Kind = "HyperConverged"

				res := cli.RestClient().Put().
					Resource("hyperconvergeds").
					Name(hcov1beta1.HyperConvergedName).
					Namespace(flags.KubeVirtInstallNamespace).
					AbsPath("/apis", hcov1beta1.SchemeGroupVersion.Group, hcov1beta1.SchemeGroupVersion.Version).
					Timeout(10 * time.Second).
					Body(hc).Do(context.TODO())

				g.Expect(res.Error()).ToNot(HaveOccurred())
				newHC := &hcov1beta1.HyperConverged{}
				err := res.Into(newHC)
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(newHC.Spec.DataImportCronTemplates).To(HaveLen(1))
				g.Expect(newHC.Spec.DataImportCronTemplates[0].Annotations).To(HaveKeyWithValue(cdiImmediateBindAnnotation, "true"), "should add the missing annotation")
			}).WithPolling(time.Second * 3).WithTimeout(time.Second * 60).Should(Succeed())
		})

		It("should not change existing annotation in the DICT", func() {
			Eventually(func(g Gomega) {
				hc := &hcov1beta1.HyperConverged{}
				Expect(cli.RestClient().
					Get().
					Resource("hyperconvergeds").
					Name("kubevirt-hyperconverged").
					Namespace(flags.KubeVirtInstallNamespace).
					AbsPath("/apis", hcov1beta1.SchemeGroupVersion.Group, hcov1beta1.SchemeGroupVersion.Version).
					Timeout(10 * time.Second).
					Do(context.TODO()).
					Into(hc),
				).To(Succeed())

				hc.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{
					getDICT(),
				}

				hc.Spec.DataImportCronTemplates[0].Annotations = map[string]string{
					cdiImmediateBindAnnotation: "false",
				}

				hc.APIVersion = "hco.kubevirt.io/v1beta1"
				hc.Kind = "HyperConverged"

				res := cli.RestClient().Put().
					Resource("hyperconvergeds").
					Name(hcov1beta1.HyperConvergedName).
					Namespace(flags.KubeVirtInstallNamespace).
					AbsPath("/apis", hcov1beta1.SchemeGroupVersion.Group, hcov1beta1.SchemeGroupVersion.Version).
					Timeout(10 * time.Second).
					Body(hc).Do(context.TODO())

				g.Expect(res.Error()).ToNot(HaveOccurred())
				newHC := &hcov1beta1.HyperConverged{}
				err := res.Into(newHC)
				g.Expect(err).ShouldNot(HaveOccurred())
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
