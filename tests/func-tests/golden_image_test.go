package tests_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/ssp-operator/api/v1beta2"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	defaultImageNamespace      = "kubevirt-os-images"
	cdiImmediateBindAnnotation = "cdi.kubevirt.io/storage.bind.immediate.requested"
)

var (
	expectedImages       = []string{"centos-7-image-cron", "centos-stream8-image-cron", "centos-stream9-image-cron", "centos8-image-cron-is", "fedora-image-cron"}
	imageNamespace       = defaultImageNamespace
	expectedImageStreams = []tests.ImageStreamConfig{
		{
			Name:         "centos8",
			RegistryName: "quay.io/kubevirt/centos8-container-disk-images",
			UsageImages:  []string{"centos8-image-cron-is"},
		},
	}
)

var _ = Describe("golden image test", Label("data-import-cron"), Serial, Ordered, Label(tests.OpenshiftLabel), func() {
	var (
		cli client.Client
		ctx context.Context
	)

	tests.FlagParse()

	if nsFromConfig := tests.GetConfig().DataImportCron.Namespace; len(nsFromConfig) > 0 {
		imageNamespace = nsFromConfig
	}

	if imageNamespaceEnv, ok := os.LookupEnv("IMAGES_NS"); ok && len(imageNamespaceEnv) > 0 {
		imageNamespace = imageNamespaceEnv
	}

	if expectedImagesFromConfig := tests.GetConfig().DataImportCron.ExpectedDataImportCrons; len(expectedImagesFromConfig) > 0 {
		expectedImages = expectedImagesFromConfig
	}
	sort.Strings(expectedImages)

	if expectedISFromConfig := tests.GetConfig().DataImportCron.ExpectedImageStream; len(expectedISFromConfig) > 0 {
		expectedImageStreams = expectedISFromConfig
	}

	BeforeEach(func() {
		cli = tests.GetControllerRuntimeClient()
		ctx = context.Background()

		tests.FailIfNotOpenShift(ctx, cli, "golden image test")
	})

	Context("test image-streams", func() {
		var isEntries []TableEntry
		for _, is := range expectedImageStreams {
			isEntries = append(isEntries, Entry(fmt.Sprintf("check the %s imagestream", is.Name), is))
		}

		DescribeTable("check that imagestream created", func(expectedIS tests.ImageStreamConfig) {
			is := getImageStream(ctx, cli, expectedIS.Name, imageNamespace)

			Expect(is.Spec.Tags[0].From).ToNot(BeNil())
			Expect(is.Spec.Tags[0].From.Kind).To(Equal("DockerImage"))
			Expect(is.Spec.Tags[0].From.Name).To(Equal(expectedIS.RegistryName))
		},
			isEntries,
		)

		DescribeTable("check imagestream reconciliation", func(expectedIS tests.ImageStreamConfig) {
			is := getImageStream(ctx, cli, expectedIS.Name, imageNamespace)

			expectedValue := is.GetLabels()["app.kubernetes.io/part-of"]
			Expect(expectedValue).ToNot(Equal("wrongValue"))

			patchOp := []byte(`[{"op": "replace", "path": "/metadata/labels/app.kubernetes.io~1part-of", "value": "wrong-value"}]`)
			patch := client.RawPatch(types.JSONPatchType, patchOp)

			Eventually(func() error {
				return cli.Patch(ctx, is, patch)
			}).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Succeed())

			Eventually(func(g Gomega) string {
				is = getImageStream(ctx, cli, expectedIS.Name, imageNamespace)
				return is.GetLabels()["app.kubernetes.io/part-of"]
			}).WithTimeout(time.Second * 15).WithPolling(time.Millisecond * 100).Should(Equal(expectedValue))
		},
			isEntries,
		)
	})

	It("make sure the feature gate is set", func() {
		hco := tests.GetHCO(ctx, cli)
		Expect(hco.Spec.FeatureGates.EnableCommonBootImageImport).To(HaveValue(BeTrue()))
	})

	Context("check default golden images", func() {
		It("should propagate the DICT to SSP", func() {
			Eventually(func(g Gomega) []string {
				ssp := getSSP(ctx, cli)
				g.Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(len(expectedImages)))

				imageNames := make([]string, len(expectedImages))
				for i, image := range ssp.Spec.CommonTemplates.DataImportCronTemplates {
					imageNames[i] = image.Name
				}
				sort.Strings(imageNames)
				return imageNames
			}).WithTimeout(10 * time.Second).WithPolling(100 * time.Millisecond).Should(Equal(expectedImages))
		})

		It("should have all the images in the HyperConverged status", func() {
			Eventually(func(g Gomega) []string {
				hco := tests.GetHCO(ctx, cli)

				g.Expect(hco.Status.DataImportCronTemplates).To(HaveLen(len(expectedImages)))

				imageNames := make([]string, len(expectedImages))
				for i, image := range hco.Status.DataImportCronTemplates {
					imageNames[i] = image.Name
				}

				sort.Strings(imageNames)
				return imageNames
			}).WithTimeout(10 * time.Second).WithPolling(100 * time.Millisecond).Should(Equal(expectedImages))
		})

		It("should have all the DataImportCron resources", func() {
			Eventually(func(g Gomega) []string {
				dicList := &cdiv1beta1.DataImportCronList{}
				Expect(cli.List(ctx, dicList, client.InNamespace(imageNamespace))).To(Succeed())

				g.Expect(dicList.Items).To(HaveLen(len(expectedImages)))

				imageNames := make([]string, len(expectedImages))
				for i, image := range dicList.Items {
					imageNames[i] = image.Name
				}

				sort.Strings(imageNames)
				return imageNames
			}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Equal(expectedImages))
		})
	})

	Context("check imagestream images", func() {
		var isUsageEntries []TableEntry
		for _, is := range expectedImageStreams {
			for _, image := range is.UsageImages {
				isUsageEntries = append(isUsageEntries, Entry(fmt.Sprintf("%s should have imageStream source", image), image, is.Name))
			}
		}

		DescribeTable("check the images that use image streams", func(imageName, streamName string) {
			dic := &cdiv1beta1.DataImportCron{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imageName,
					Namespace: imageNamespace,
				},
			}

			Expect(cli.Get(ctx, client.ObjectKeyFromObject(dic), dic)).To(Succeed())

			Expect(dic.Spec.Template.Spec.Source).ToNot(BeNil())
			Expect(dic.Spec.Template.Spec.Source.Registry).ToNot(BeNil())
			Expect(dic.Spec.Template.Spec.Source.Registry.ImageStream).To(HaveValue(Equal(streamName)))
			Expect(dic.Spec.Template.Spec.Source.Registry.PullMethod).To(HaveValue(Equal(cdiv1beta1.RegistryPullNode)))
		}, isUsageEntries)
	})

	Context("disable the feature", func() {
		It("Should set the FG to false", func() {
			patch := []byte(`[{ "op": "replace", "path": "/spec/featureGates/enableCommonBootImageImport", "value": false }]`)
			Eventually(tests.PatchHCO).WithArguments(ctx, cli, patch).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		})

		var isEntries []TableEntry
		for _, is := range expectedImageStreams {
			isEntries = append(isEntries, Entry(fmt.Sprintf("check the %s imagestream", is.Name), is))
		}

		if len(isEntries) > 0 {
			DescribeTable("imageStream should be removed", func(expectedIS tests.ImageStreamConfig) {
				Eventually(func() error {
					is := &v1.ImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      expectedIS.Name,
							Namespace: imageNamespace,
						},
					}

					return cli.Get(ctx, client.ObjectKeyFromObject(is), is)
				}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(MatchError(errors.IsNotFound, "not found error"))
			}, isEntries)
		}

		It("should empty the DICT in SSP", func() {
			Eventually(func(g Gomega) []v1beta2.DataImportCronTemplate {
				ssp := getSSP(ctx, cli)
				return ssp.Spec.CommonTemplates.DataImportCronTemplates
			}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(BeEmpty())
		})

		It("should have no images in the HyperConverged status", func() {
			Eventually(func() []hcov1beta1.DataImportCronTemplateStatus {
				hco := tests.GetHCO(ctx, cli)
				return hco.Status.DataImportCronTemplates
			}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(BeEmpty())
		})

		It("should have no images", func() {
			Eventually(func(g Gomega) []v1.ImageStream {
				isList := &v1.ImageStreamList{}
				Expect(cli.List(ctx, isList, client.InNamespace(imageNamespace))).To(Succeed())

				return isList.Items
			}).WithTimeout(5 * time.Minute).WithPolling(time.Second).Should(BeEmpty())
		})
	})

	Context("enable the feature again", func() {
		It("Should set the FG to false", func() {
			patch := []byte(`[{ "op": "replace", "path": "/spec/featureGates/enableCommonBootImageImport", "value": true }]`)
			Eventually(tests.PatchHCO).WithArguments(ctx, cli, patch).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		})

		var isEntries []TableEntry
		for _, is := range expectedImageStreams {
			isEntries = append(isEntries, Entry(fmt.Sprintf("check the %s imagestream", is.Name), is))
		}

		if len(isEntries) > 0 {
			DescribeTable("imageStream should be recovered", func(expectedIS tests.ImageStreamConfig) {
				Eventually(func(g Gomega) error {
					is := v1.ImageStream{
						ObjectMeta: metav1.ObjectMeta{
							Name:      expectedIS.Name,
							Namespace: imageNamespace,
						},
					}
					return cli.Get(ctx, client.ObjectKeyFromObject(&is), &is)
				}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).ShouldNot(HaveOccurred())
			}, isEntries)
		}

		It("should propagate the DICT in SSP", func() {
			Eventually(func(g Gomega) []v1beta2.DataImportCronTemplate {
				ssp := getSSP(ctx, cli)
				return ssp.Spec.CommonTemplates.DataImportCronTemplates
			}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(HaveLen(len(expectedImages)))
		})

		It("should have all the images in the HyperConverged status", func() {
			Eventually(func() []hcov1beta1.DataImportCronTemplateStatus {
				hco := tests.GetHCO(ctx, cli)
				return hco.Status.DataImportCronTemplates
			}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(HaveLen(len(expectedImages)))
		})

		It("should restore all the DataImportCron resources", func() {
			Eventually(func(g Gomega) []cdiv1beta1.DataImportCron {
				dicList := &cdiv1beta1.DataImportCronList{}
				Expect(cli.List(ctx, dicList, client.InNamespace(imageNamespace))).To(Succeed())

				return dicList.Items
			}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(HaveLen(len(expectedImages)))
		})
	})

	Context("test annotations", func() {

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
	return hcov1beta1.DataImportCronTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "custom",
		},
		Spec: &cdiv1beta1.DataImportCronSpec{
			GarbageCollect:    ptr.To(cdiv1beta1.DataImportCronGarbageCollectOutdated),
			ManagedDataSource: "centos7",
			Schedule:          "18 1/12 * * *",
			Template: cdiv1beta1.DataVolume{
				Spec: cdiv1beta1.DataVolumeSpec{
					Source: &cdiv1beta1.DataVolumeSource{
						Registry: &cdiv1beta1.DataVolumeSourceRegistry{
							PullMethod: ptr.To(cdiv1beta1.RegistryPullNode),
							URL:        ptr.To("docker://quay.io/containerdisks/centos:7-2009"),
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

func getSSP(ctx context.Context, cli client.Client) *v1beta2.SSP {
	ssp := &v1beta2.SSP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ssp-kubevirt-hyperconverged",
			Namespace: tests.InstallNamespace,
		},
	}

	Expect(cli.Get(ctx, client.ObjectKeyFromObject(ssp), ssp)).To(Succeed())
	return ssp
}

func getImageStream(ctx context.Context, cli client.Client, name, namespace string) *v1.ImageStream {
	is := &v1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	Expect(cli.Get(ctx, client.ObjectKeyFromObject(is), is)).To(Succeed())

	return is
}
