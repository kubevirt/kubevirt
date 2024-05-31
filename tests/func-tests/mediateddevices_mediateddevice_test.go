package tests_test

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/flags"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	apiservererrors "k8s.io/apiserver/pkg/admission/plugin/webhook/errors"
)

var _ = Describe("MediatedDevicesTypes -> MediatedDeviceTypes", func() {
	tests.FlagParse()
	var cli kubecli.KubevirtClient
	ctx := context.TODO()

	var initialMDC *v1beta1.MediatedDevicesConfiguration

	apiServerError := apiservererrors.ToStatusErr(
		util.HcoValidatingWebhook,
		&metav1.Status{
			Message: "mediatedDevicesTypes is deprecated, please use mediatedDeviceTypes instead",
			Reason:  metav1.StatusReasonForbidden,
			Code:    403,
		})
	apiServerError.ErrStatus.APIVersion = "v1"
	apiServerError.ErrStatus.Kind = "Status"

	BeforeEach(func() {
		var err error
		cli, err = kubecli.GetKubevirtClient()
		Expect(cli).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())
		tests.BeforeEach()
		hc := tests.GetHCO(ctx, cli)
		initialMDC = nil
		if hc.Spec.MediatedDevicesConfiguration != nil {
			initialMDC = hc.Spec.MediatedDevicesConfiguration.DeepCopy()
		}
	})

	AfterEach(func() {
		hc := tests.GetHCO(ctx, cli)
		hc.Spec.MediatedDevicesConfiguration = initialMDC
		_ = tests.UpdateHCORetry(ctx, cli, hc)
	})

	DescribeTable("should correctly handle MediatedDevicesTypes -> MediatedDeviceTypes transition",
		func(mediatedDevicesConfiguration *v1beta1.MediatedDevicesConfiguration, expectedErr error, expectedMediatedDevicesConfiguration *v1beta1.MediatedDevicesConfiguration, expectedKVMediatedDevicesConfiguration *kubevirtcorev1.MediatedDevicesConfiguration) {
			if expectedErr == nil {
				hc := tests.GetHCO(ctx, cli)
				hc.Spec.MediatedDevicesConfiguration = mediatedDevicesConfiguration
				hc = tests.UpdateHCORetry(ctx, cli, hc)
				Expect(hc.Spec.MediatedDevicesConfiguration).To(Equal(expectedMediatedDevicesConfiguration))
				Eventually(func() *kubevirtcorev1.MediatedDevicesConfiguration {
					kubevirt, err := cli.KubeVirt(flags.KubeVirtInstallNamespace).Get(ctx, "kubevirt-kubevirt-hyperconverged", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return kubevirt.Spec.Configuration.MediatedDevicesConfiguration
				}, 10*time.Second, time.Second).Should(Equal(expectedKVMediatedDevicesConfiguration))
			} else {
				Eventually(func() error {
					hc := tests.GetHCO(ctx, cli)
					hc.Spec.MediatedDevicesConfiguration = mediatedDevicesConfiguration
					_, err := tests.UpdateHCO(ctx, cli, hc)
					return err
				}, 10*time.Second, time.Second).Should(MatchError(expectedErr))
			}
		},
		Entry("should do nothing when mediatedDevicesConfiguration is not present",
			nil,
			nil,
			nil,
			nil,
		),
		Entry("should migrate to mediatedDeviceTypes when deprecated are used",
			&v1beta1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
				NodeMediatedDeviceTypes: []v1beta1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDevicesTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDevicesTypes: []string{ //nolint SA1019
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
			nil,
			&v1beta1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
				MediatedDeviceTypes:  []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []v1beta1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDevicesTypes: []string{ //nolint SA1019
							"nvidia-223",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDevicesTypes: []string{ //nolint SA1019
							"nvidia-229",
							"nvidia-232",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
			&kubevirtcorev1.MediatedDevicesConfiguration{
				MediatedDeviceTypes: []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []kubevirtcorev1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
		),
		Entry("should do nothing when mediatedDeviceTypes and deprecated APIs are used consistently",
			&v1beta1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
				MediatedDeviceTypes:  []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []v1beta1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDevicesTypes: []string{
							"nvidia-223",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDevicesTypes: []string{ //nolint SA1019
							"nvidia-229",
							"nvidia-232",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
			nil,
			&v1beta1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
				MediatedDeviceTypes:  []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []v1beta1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDevicesTypes: []string{ //nolint SA1019
							"nvidia-223",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDevicesTypes: []string{ //nolint SA1019
							"nvidia-229",
							"nvidia-232",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
			&kubevirtcorev1.MediatedDevicesConfiguration{
				MediatedDeviceTypes: []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []kubevirtcorev1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
		),
		Entry("should do nothing when only mediatedDeviceTypes is used",
			&v1beta1.MediatedDevicesConfiguration{
				MediatedDeviceTypes: []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []v1beta1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
			nil,
			&v1beta1.MediatedDevicesConfiguration{
				MediatedDeviceTypes: []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []v1beta1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
			&kubevirtcorev1.MediatedDevicesConfiguration{
				MediatedDeviceTypes: []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []kubevirtcorev1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
		),
		Entry("should refuse inconsistent values between mediatedDeviceTypes and deprecated APIs - spec.mediatedDevicesConfiguration.mediatedDeviceTypes",
			&v1beta1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
				MediatedDeviceTypes:  []string{"nvidia-232"},
				NodeMediatedDeviceTypes: []v1beta1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDevicesTypes: []string{
							"nvidia-223",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDevicesTypes: []string{ //nolint SA1019
							"nvidia-229",
							"nvidia-232",
						},
						MediatedDeviceTypes: []string{
							"nvidia-229",
							"nvidia-232",
						},
					},
				},
			},
			apiServerError,
			nil,
			nil,
		),
		Entry("should refuse inconsistent values between mediatedDeviceTypes and deprecated APIs - spec.mediatedDevicesConfiguration.nodeMediatedDeviceTypes[1].mediatedDeviceTypes",
			&v1beta1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
				MediatedDeviceTypes:  []string{"nvidia-222", "nvidia-230"},
				NodeMediatedDeviceTypes: []v1beta1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDevicesTypes: []string{
							"nvidia-223",
						},
						MediatedDeviceTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDevicesTypes: []string{ //nolint SA1019
							"nvidia-229",
							"nvidia-232",
						},
						MediatedDeviceTypes: []string{
							"nvidia-218",
						},
					},
				},
			},
			apiServerError,
			nil,
			nil,
		),
	)

})
