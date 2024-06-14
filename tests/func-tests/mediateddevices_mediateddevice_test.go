package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiservererrors "k8s.io/apiserver/pkg/admission/plugin/webhook/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("MediatedDevicesTypes -> MediatedDeviceTypes", Label("MediatedDevices"), func() {
	var (
		cli            client.Client
		initialMDC     *v1beta1.MediatedDevicesConfiguration
		apiServerError = apiservererrors.ToStatusErr(
			util.HcoValidatingWebhook,
			&metav1.Status{
				Message: "mediatedDevicesTypes is deprecated, please use mediatedDeviceTypes instead",
				Reason:  metav1.StatusReasonForbidden,
				Code:    403,
			},
		)
	)

	tests.FlagParse()

	BeforeEach(func(ctx context.Context) {
		cli = tests.GetControllerRuntimeClient()

		tests.BeforeEach(ctx)
		hc := tests.GetHCO(ctx, cli)
		initialMDC = nil
		if hc.Spec.MediatedDevicesConfiguration != nil {
			initialMDC = hc.Spec.MediatedDevicesConfiguration.DeepCopy()
		}
	})

	AfterEach(func(ctx context.Context) {
		hc := tests.GetHCO(ctx, cli)
		hc.Spec.MediatedDevicesConfiguration = initialMDC
		_ = tests.UpdateHCORetry(ctx, cli, hc)
	})

	DescribeTable("should correctly handle MediatedDevicesTypes -> MediatedDeviceTypes transition",
		func(ctx context.Context, mediatedDevicesConfiguration *v1beta1.MediatedDevicesConfiguration, expectedErr error, expectedMediatedDevicesConfiguration *v1beta1.MediatedDevicesConfiguration, expectedKVMediatedDevicesConfiguration *kubevirtcorev1.MediatedDevicesConfiguration) {
			if expectedErr == nil {
				hc := tests.GetHCO(ctx, cli)
				hc.Spec.MediatedDevicesConfiguration = mediatedDevicesConfiguration
				hc = tests.UpdateHCORetry(ctx, cli, hc)
				Expect(hc.Spec.MediatedDevicesConfiguration).To(Equal(expectedMediatedDevicesConfiguration))
				Eventually(func(g Gomega, ctx context.Context) *kubevirtcorev1.MediatedDevicesConfiguration {
					kv := &kubevirtcorev1.KubeVirt{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kubevirt-kubevirt-hyperconverged",
							Namespace: tests.InstallNamespace,
						},
					}

					g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(kv), kv)).To(Succeed())

					return kv.Spec.Configuration.MediatedDevicesConfiguration
				}).WithTimeout(10 * time.Second).
					WithPolling(time.Second).
					WithContext(ctx).
					Should(Equal(expectedKVMediatedDevicesConfiguration))
			} else {
				Eventually(func(ctx context.Context) error {
					hc := tests.GetHCO(ctx, cli)
					hc.Spec.MediatedDevicesConfiguration = mediatedDevicesConfiguration
					_, err := tests.UpdateHCO(ctx, cli, hc)
					return err
				}).WithTimeout(10 * time.Second).
					WithPolling(time.Second).
					WithContext(ctx).
					Should(MatchError(expectedErr))
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
