package tests_test

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"

	"kubevirt.io/kubevirt/tests/flags"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("nonRoot -> root FG", func() {
	tests.FlagParse()
	var cli kubecli.KubevirtClient
	ctx := context.TODO()

	cli, err := kubecli.GetKubevirtClient()
	Expect(cli).ToNot(BeNil())
	Expect(err).ToNot(HaveOccurred())
	var initialNonRoot *bool
	var initialRoot *bool

	el := field.ErrorList{field.Invalid(field.NewPath("spec", "featureGates"), "object", "nonRoot FG is deprecated, please use root FG with opposite logic")}
	invalidHCOError := errors.NewInvalid(schema.GroupKind{Kind: util.HyperConvergedKind, Group: util.APIVersionGroup}, util.HyperConvergedName, el)
	invalidHCOError.ErrStatus.APIVersion = "v1"
	invalidHCOError.ErrStatus.Kind = "Status"

	const kvRoot = "Root"

	BeforeEach(func() {
		tests.BeforeEach()
		hc := tests.GetHCO(ctx, cli)
		initialNonRoot = hc.Spec.FeatureGates.NonRoot //nolint SA1019
		initialRoot = hc.Spec.FeatureGates.Root
	})

	AfterEach(func() {
		hc := tests.GetHCO(ctx, cli)
		hc.Spec.FeatureGates.NonRoot = initialNonRoot //nolint SA1019
		hc.Spec.FeatureGates.Root = initialRoot
		_ = tests.UpdateHCORetry(ctx, cli, hc)
	})

	DescribeTable("should correctly handle nonRoot -> root FG transition",
		func(nonRootFG *bool, rootFG *bool, expectedErr error, expectedNonRootFG *bool, expectedRootFG *bool) {
			if expectedErr == nil {
				hc := tests.GetHCO(ctx, cli)
				hc.Spec.FeatureGates.NonRoot = nonRootFG //nolint SA1019
				hc.Spec.FeatureGates.Root = rootFG
				hc = tests.UpdateHCORetry(ctx, cli, hc)
				Expect(hc.Spec.FeatureGates.NonRoot).To(Equal(expectedNonRootFG)) //nolint SA1019
				Expect(hc.Spec.FeatureGates.Root).To(Equal(expectedRootFG))

				Eventually(func() bool {
					kubevirt, err := cli.KubeVirt(flags.KubeVirtInstallNamespace).Get("kubevirt-kubevirt-hyperconverged", &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return slices.Contains(kubevirt.Spec.Configuration.DeveloperConfiguration.FeatureGates, kvRoot)
				}, 10*time.Second, time.Second).Should(Equal(*expectedRootFG))

			} else {
				Eventually(func() error {
					hc := tests.GetHCO(ctx, cli)
					hc.Spec.FeatureGates.NonRoot = nonRootFG //nolint SA1019
					hc.Spec.FeatureGates.Root = rootFG
					_, err = tests.UpdateHCO(ctx, cli, hc)
					return err
				}, 10*time.Second, time.Second).Should(MatchError(expectedErr))
			}
		},
		// NonRoot: nil
		Entry("NonRoot: nil, Root: nil -> NonRoot: nil, Root: false",
			nil,
			nil,
			nil,
			nil,
			pointer.Bool(false),
		),
		Entry("NonRoot: nil, Root: false -> NonRoot: nil, Root: false",
			nil,
			pointer.Bool(false),
			nil,
			nil,
			pointer.Bool(false),
		),
		Entry("NonRoot: nil, Root: true -> NonRoot: false, Root: true",
			nil,
			pointer.Bool(true),
			nil,
			nil,
			pointer.Bool(true),
		),
		// NonRoot: false
		Entry("NonRoot: false, Root: nil -> NonRoot: false, Root: true",
			pointer.Bool(false),
			nil,
			nil,
			pointer.Bool(false),
			pointer.Bool(true),
		),
		Entry("NonRoot: false, Root: false -> error",
			pointer.Bool(false),
			pointer.Bool(false),
			invalidHCOError,
			nil,
			nil,
		),
		Entry("NonRoot: false, Root: true -> NonRoot: false, Root: true",
			pointer.Bool(false),
			pointer.Bool(true),
			nil,
			pointer.Bool(false),
			pointer.Bool(true),
		),
		// NonRoot: true
		Entry("NonRoot: true, Root: nil -> NonRoot: true, Root: false",
			pointer.Bool(true),
			nil,
			nil,
			pointer.Bool(true),
			pointer.Bool(false),
		),
		Entry("NonRoot: true, Root: false -> NonRoot: true, Root: false",
			pointer.Bool(true),
			pointer.Bool(false),
			nil,
			pointer.Bool(true),
			pointer.Bool(false),
		),
		Entry("NonRoot: true, Root: true -> error",
			pointer.Bool(true),
			pointer.Bool(true),
			invalidHCOError,
			nil,
			nil,
		),
	)

})
