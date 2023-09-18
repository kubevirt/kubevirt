package monitoring

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/tests/framework/checks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/migrations/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("[Serial][sig-monitoring]VM Migration Monitoring", Serial, decorators.SigMonitoring, decorators.RequiresTwoSchedulableNodes, func() {
	var virtClient kubecli.KubevirtClient
	var vmi *v1.VirtualMachineInstance

	Context("Migration metrics", func() {
		createVmi := func() *v1.VirtualMachineInstance {
			By("Starting the VirtualMachineInstance")
			opts := append(
				libvmi.WithMasqueradeNetworking(),
				libvmi.WithEmptyDisk("emptydisk1", v1.DiskBusUSB, resource.MustParse("2Gi")),
			)

			vmi := libvmi.NewCirros(opts...)
			vmi.Namespace = testsuite.GetTestNamespace(nil)

			return tests.RunVMIAndExpectLaunch(vmi, 240)
		}

		createMigrationPolicy := func(vmi *v1.VirtualMachineInstance, bandwidthPerMigration resource.Quantity) {
			By("Creating a migration policy")
			_, err := virtClient.MigrationPolicy().Create(context.TODO(), &v1alpha1.MigrationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "migration-policy-" + vmi.Name,
				},
				Spec: v1alpha1.MigrationPolicySpec{
					BandwidthPerMigration: &bandwidthPerMigration,
					Selectors: &v1alpha1.Selectors{
						NamespaceSelector: map[string]string{
							"kubernetes.io/metadata.name": vmi.Namespace,
						},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		deleteMigrationPolicy := func(vmi *v1.VirtualMachineInstance) {
			By("Deleting a migration policy")
			err := virtClient.MigrationPolicy().Delete(context.TODO(), "migration-policy-"+vmi.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		getTransferRateMetric := func(vmi *v1.VirtualMachineInstance) float64 {
			labels := map[string]string{
				"name":      vmi.Name,
				"namespace": vmi.Namespace,
			}
			i, err := getMetricValueWithLabels(virtClient, "kubevirt_migrate_vmi_disk_transfer_rate_bytes", labels)
			if err != nil {
				return -1
			}
			return i
		}

		BeforeEach(func() {
			virtClient = kubevirt.Client()
			checks.SkipIfPrometheusRuleIsNotEnabled(virtClient)
		})

		AfterEach(func() {
			if vmi != nil {
				deleteMigrationPolicy(vmi)
			}
		})

		It("should show kubevirt_migrate_vmi_disk_transfer_rate_bytes", func() {
			oneMi, err := resource.ParseQuantity("1Mi")
			Expect(err).ToNot(HaveOccurred())

			vmi = createVmi()
			createMigrationPolicy(vmi, oneMi)

			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)

			Eventually(func() float64 {
				return getTransferRateMetric(vmi)
			}, 2*time.Minute, 20*time.Second).ShouldNot(Equal(-1.0))
			Eventually(func() float64 {
				return getTransferRateMetric(vmi)
			}, 3*time.Minute, 20*time.Second).Should(BeNumerically("~", oneMi.Value(), oneMi.Value()/10))

			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
		})
	})
})
