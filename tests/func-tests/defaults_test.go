package tests_test

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	removePathPatchTmplt = `[{"op": "remove", "path": %q}]`
)

var _ = Describe("Check Default values", Label("defaults"), Serial, func() {
	var (
		cli client.Client
		ctx context.Context
	)

	BeforeEach(func() {
		cli = tests.GetControllerRuntimeClient()

		ctx = context.Background()

		tests.RestoreDefaults(ctx, cli)
	})

	Context("certConfig defaults", func() {
		defaultCertConfig := v1beta1.HyperConvergedCertConfig{
			CA: v1beta1.CertRotateConfigCA{
				Duration:    &metav1.Duration{Duration: time.Hour * 48},
				RenewBefore: &metav1.Duration{Duration: time.Hour * 24},
			},
			Server: v1beta1.CertRotateConfigServer{
				Duration:    &metav1.Duration{Duration: time.Hour * 24},
				RenewBefore: &metav1.Duration{Duration: time.Hour * 12},
			},
		}

		DescribeTable("Check that certConfig defaults are behaving as expected", func(path string) {
			patch := []byte(fmt.Sprintf(removePathPatchTmplt, path))
			Eventually(func() error {
				return tests.PatchHCO(ctx, cli, patch)
			}).WithTimeout(2 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)
				g.Expect(reflect.DeepEqual(hc.Spec.CertConfig, defaultCertConfig)).To(BeTrue(), "certConfig should be equal to default")
			}).WithTimeout(2 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		},
			Entry("when removing /spec/certConfig/ca/duration", "/spec/certConfig/ca/duration"),
			Entry("when removing /spec/certConfig/ca/renewBefore", "/spec/certConfig/ca/renewBefore"),
			Entry("when removing /spec/certConfig/ca", "/spec/certConfig/ca"),
			Entry("when removing /spec/certConfig/server/duration", "/spec/certConfig/server/duration"),
			Entry("when removing /spec/certConfig/server/renewBefore", "/spec/certConfig/server/renewBefore"),
			Entry("when removing /spec/certConfig/server", "/spec/certConfig/server"),
			Entry("when removing /spec/certConfig", "/spec/certConfig"),
			Entry("when removing /spec", "/spec"),
		)
	})

	Context("feature gate defaults", func() {
		defaultFeatureGates := v1beta1.HyperConvergedFeatureGates{
			DownwardMetrics:             ptr.To(false),
			DeployKubeSecondaryDNS:      ptr.To(false),
			DeployTektonTaskResources:   ptr.To(false),
			DeployVMConsoleProxy:        ptr.To(false),
			DisableMDevConfiguration:    ptr.To(false),
			EnableCommonBootImageImport: ptr.To(true),
			PersistentReservation:       ptr.To(false),
			NonRoot:                     ptr.To(true), //nolint SA1019
			WithHostPassthroughCPU:      ptr.To(false),
			EnableManagedTenantQuota:    ptr.To(false),
			AutoResourceLimits:          ptr.To(false),
			AlignCPUs:                   ptr.To(false),
			EnableApplicationAwareQuota: ptr.To(false),
		}

		DescribeTable("Check that featureGates defaults are behaving as expected", func(path string) {
			patch := []byte(fmt.Sprintf(removePathPatchTmplt, path))
			Eventually(func() error {
				return tests.PatchHCO(ctx, cli, patch)
			}).WithTimeout(2 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)
				g.Expect(reflect.DeepEqual(hc.Spec.FeatureGates, defaultFeatureGates)).To(BeTrue(), "featureGates should be equal to default")
			}).WithTimeout(2 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		},
			Entry("when removing /spec/featureGates/downwardMetrics", "/spec/featureGates/downwardMetrics"),
			Entry("when removing /spec/featureGates/deployKubeSecondaryDNS", "/spec/featureGates/deployKubeSecondaryDNS"),
			Entry("when removing /spec/featureGates/deployTektonTaskResources", "/spec/featureGates/deployTektonTaskResources"),
			Entry("when removing /spec/featureGates/deployVmConsoleProxy", "/spec/featureGates/deployVmConsoleProxy"),
			Entry("when removing /spec/featureGates/disableMDevConfiguration", "/spec/featureGates/disableMDevConfiguration"),
			Entry("when removing /spec/featureGates/enableCommonBootImageImport", "/spec/featureGates/enableCommonBootImageImport"),
			Entry("when removing /spec/featureGates/persistentReservation", "/spec/featureGates/persistentReservation"),
			Entry("when removing /spec/featureGates/nonRoot", "/spec/featureGates/nonRoot"),
			Entry("when removing /spec/featureGates/withHostPassthroughCPU", "/spec/featureGates/withHostPassthroughCPU"),
			Entry("when removing /spec/featureGates/enableManagedTenantQuota", "/spec/featureGates/enableManagedTenantQuota"),
			Entry("when removing /spec/featureGates/autoResourceLimits", "/spec/featureGates/autoResourceLimits"),
			Entry("when removing /spec/featureGates/alignCPUs", "/spec/featureGates/alignCPUs"),
			Entry("when removing /spec/featureGates", "/spec/featureGates"),
			Entry("when removing /spec", "/spec"),
		)
	})

	Context("liveMigrationConfig defaults", func() {
		defaultLiveMigrationConfig := v1beta1.LiveMigrationConfigurations{
			AllowAutoConverge:                 ptr.To(false),
			AllowPostCopy:                     ptr.To(false),
			CompletionTimeoutPerGiB:           ptr.To(int64(800)),
			ParallelMigrationsPerCluster:      ptr.To(uint32(5)),
			ParallelOutboundMigrationsPerNode: ptr.To(uint32(2)),
			ProgressTimeout:                   ptr.To(int64(150)),
		}

		DescribeTable("Check that liveMigrationConfig defaults are behaving as expected", func(path string) {
			patch := []byte(fmt.Sprintf(removePathPatchTmplt, path))
			Eventually(func() error {
				return tests.PatchHCO(ctx, cli, patch)
			}).WithTimeout(2 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)
				g.Expect(reflect.DeepEqual(hc.Spec.LiveMigrationConfig, defaultLiveMigrationConfig)).To(BeTrue(), "liveMigrationConfig should be equal to default")
			}).WithTimeout(2 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		},
			Entry("when removing /spec/liveMigrationConfig/allowAutoConverge", "/spec/liveMigrationConfig/allowAutoConverge"),
			Entry("when removing /spec/liveMigrationConfig/allowPostCopy", "/spec/liveMigrationConfig/allowPostCopy"),
			Entry("when removing /spec/liveMigrationConfig/completionTimeoutPerGiB", "/spec/liveMigrationConfig/completionTimeoutPerGiB"),
			Entry("when removing /spec/liveMigrationConfig/parallelMigrationsPerCluster", "/spec/liveMigrationConfig/parallelMigrationsPerCluster"),
			Entry("when removing /spec/liveMigrationConfig/parallelOutboundMigrationsPerNode", "/spec/liveMigrationConfig/parallelOutboundMigrationsPerNode"),
			Entry("when removing /spec/liveMigrationConfig/progressTimeout", "/spec/liveMigrationConfig/progressTimeout"),
			Entry("when removing /spec/liveMigrationConfig", "/spec/liveMigrationConfig"),
			Entry("when removing /spec", "/spec"),
		)
	})

	Context("resourceRequirements defaults", func() {
		defaultResourceRequirements := v1beta1.OperandResourceRequirements{
			VmiCPUAllocationRatio: ptr.To(10),
		}

		DescribeTable("Check that resourceRequirements defaults are behaving as expected", func(path string) {
			patch := []byte(fmt.Sprintf(removePathPatchTmplt, path))
			Eventually(func() error {
				return tests.PatchHCO(ctx, cli, patch)
			}).WithTimeout(20 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)
				g.Expect(reflect.DeepEqual(hc.Spec.ResourceRequirements, &defaultResourceRequirements)).To(BeTrue(), "resourceRequirements should be equal to default")
			}).WithTimeout(2 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		},
			Entry("when removing /spec/resourceRequirements/vmiCPUAllocationRatio", "/spec/resourceRequirements/vmiCPUAllocationRatio"),
			Entry("when removing /spec/resourceRequirements", "/spec/resourceRequirements"),
			Entry("when removing /spec", "/spec"),
		)
	})

	Context("workloadUpdateStrategy defaults", func() {
		defaultWorkloadUpdateStrategy := v1beta1.HyperConvergedWorkloadUpdateStrategy{
			BatchEvictionInterval: &metav1.Duration{Duration: time.Minute},
			BatchEvictionSize:     ptr.To(10),
			WorkloadUpdateMethods: []string{"LiveMigrate"},
		}

		DescribeTable("Check that workloadUpdateStrategy defaults are behaving as expected", func(path string) {
			patch := []byte(fmt.Sprintf(removePathPatchTmplt, path))
			Eventually(func() error {
				return tests.PatchHCO(ctx, cli, patch)
			}).WithTimeout(20 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)
				g.Expect(reflect.DeepEqual(hc.Spec.WorkloadUpdateStrategy, defaultWorkloadUpdateStrategy)).To(BeTrue(), "workloadUpdateStrategy should be equal to default")
			}).WithTimeout(2 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		},
			Entry("when removing /spec/workloadUpdateStrategy/batchEvictionInterval", "/spec/workloadUpdateStrategy/batchEvictionInterval"),
			Entry("when removing /spec/workloadUpdateStrategy/batchEvictionSize", "/spec/workloadUpdateStrategy/batchEvictionSize"),
			Entry("when removing /spec/workloadUpdateStrategy/workloadUpdateMethods", "/spec/workloadUpdateStrategy/workloadUpdateMethods"),
			Entry("when removing /spec/workloadUpdateStrategy", "/spec/workloadUpdateStrategy"),
			Entry("when removing /spec", "/spec"),
		)
	})

	Context("uninstallStrategy defaults", func() {
		const defaultUninstallStrategy = `BlockUninstallIfWorkloadsExist`

		DescribeTable("Check that uninstallStrategy default is behaving as expected", func(path string) {
			patch := []byte(fmt.Sprintf(removePathPatchTmplt, path))
			Eventually(func() error {
				return tests.PatchHCO(ctx, cli, patch)
			}).WithTimeout(2 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)
				g.Expect(hc.Spec.UninstallStrategy).To(Equal(v1beta1.HyperConvergedUninstallStrategy(defaultUninstallStrategy)), "uninstallStrategy should be equal to default")
			}).WithTimeout(2 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		},
			Entry("when removing /spec/uninstallStrategy", "/spec/uninstallStrategy"),
			Entry("when removing /spec", "/spec"),
		)
	})

	Context("VirtualMachineOptions defaults", func() {
		defaultVirtualMachineOptions := &v1beta1.VirtualMachineOptions{
			DisableFreePageReporting: ptr.To(false),
			DisableSerialConsoleLog:  ptr.To(true),
		}

		DescribeTable("Check that featureGates defaults are behaving as expected", func(path string) {
			patch := []byte(fmt.Sprintf(removePathPatchTmplt, path))
			Eventually(func() error {
				return tests.PatchHCO(ctx, cli, patch)
			}).WithTimeout(2 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)
				g.Expect(reflect.DeepEqual(hc.Spec.VirtualMachineOptions, defaultVirtualMachineOptions)).To(BeTrue(), "virtualMachineOptions should be equal to default")
			}).WithTimeout(2 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		},
			Entry("when removing /spec/virtualMachineOptions/disableFreePageReporting", "/spec/virtualMachineOptions/disableFreePageReporting"),
			Entry("when removing /spec/virtualMachineOptions/disableSerialConsoleLog", "/spec/virtualMachineOptions/disableSerialConsoleLog"),
			Entry("when removing /spec/virtualMachineOptions", "/spec/virtualMachineOptions"),
			Entry("when removing /spec", "/spec"),
		)
	})

	Context("HigherWorkloadDensity defaults", func() {
		defaultHigherWorkloadDensity := &v1beta1.HigherWorkloadDensityConfiguration{
			MemoryOvercommitPercentage: 100,
		}

		DescribeTable("Check that HigherWorkloadDensity defaults are behaving as expected", func(path string) {
			patch := []byte(fmt.Sprintf(removePathPatchTmplt, path))
			Eventually(func() error {
				return tests.PatchHCO(ctx, cli, patch)
			}).WithTimeout(2 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())

			Eventually(func(g Gomega) {
				hc := tests.GetHCO(ctx, cli)
				g.Expect(reflect.DeepEqual(hc.Spec.HigherWorkloadDensity, defaultHigherWorkloadDensity)).To(BeTrue(), "HigherWorkloadDensity should be equal to default")
			}).WithTimeout(2 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())
		},
			Entry("when removing /spec/higherWorkloadDensity/memoryOvercommitPercentage", "/spec/higherWorkloadDensity/memoryOvercommitPercentage"),
			Entry("when removing /spec/higherWorkloadDensity", "/spec/higherWorkloadDensity"),
			Entry("when removing /spec", "/spec"),
		)
	})
})
