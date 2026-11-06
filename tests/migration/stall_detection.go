package migration

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

const (
	stressSize               = "50%"
	stallDetectionTimeout    = 5 * time.Minute
	migrationCompleteTimeout = 300

	stallDetectedString = "stall detected: bestRemainingBytes=([0-9.]+)Mib outsideWindowMin=([0-9.]+)Mib candidates=([0-9]+)"
)

func newStallMigrationPolicy(vmi *v1.VirtualMachineInstance) *migrationsv1.MigrationPolicy {
	policy := GeneratePolicyAndAlignVMI(vmi)
	policy.Spec.BandwidthPerMigration = pointer.P(resource.MustParse("15Mi"))
	policy.Spec.CompletionTimeoutPerGiB = pointer.P(int64(250))
	return policy
}

func getVirtLauncherSourceLogs(g Gomega, virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	GinkgoHelper()
	freshVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	g.Expect(err).ToNot(HaveOccurred(), "should get VMI")
	g.Expect(freshVMI.Status.MigrationState).ToNot(BeNil())
	sourcePod := freshVMI.Status.MigrationState.SourcePod
	g.Expect(sourcePod).ToNot(BeEmpty(), "source pod name must be populated during migration")
	logsRaw, err := virtClient.CoreV1().
		Pods(vmi.Namespace).
		GetLogs(sourcePod, &k8sv1.PodLogOptions{Container: "compute"}).
		DoRaw(context.Background())
	g.Expect(err).ToNot(HaveOccurred(), "should get virt-launcher source pod logs")
	return string(logsRaw)
}

var _ = Describe(SIG("Migration Stall Detection", Serial, decorators.RequiresTwoSchedulableNodes, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		enabled, err := util.IsMutatingAdmissionPolicyEnabled(virtClient)
		Expect(err).ToNot(HaveOccurred())
		if !enabled {
			Fail("MutatingAdmissionPolicy not available in this cluster")
		}

		kvconfig.EnableFeatureGate(featuregate.MigrationStallDetection)
		config := getCurrentKvConfig(virtClient)
		if config.DeveloperConfiguration.LogVerbosity == nil {
			config.DeveloperConfiguration.LogVerbosity = &v1.LogVerbosity{}
		}
		config.DeveloperConfiguration.LogVerbosity.VirtLauncher = 4
		kvconfig.UpdateKubeVirtConfigValueAndWait(config)
	})

	Context("with post-copy enabled", func() {
		It("should detect stall and switch to post-copy", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(true)
			policy.Spec.AllowWorkloadDisruption = pointer.P(true)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(1))
			CreateMigrationPolicy(virtClient, policy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Logging in and running stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			Expect(runParticularStressTest(vmi, stressSize, "write64", 1*time.Second)).To(Succeed())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Waiting for stall detection to trigger")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying post-copy mode is entered")
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPostCopy, 5*time.Minute)

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})
	})

	Context("with workload disruption allowed and post-copy disabled", func() {
		It("should detect stall and force switchover via max downtime", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(false)
			policy.Spec.AllowWorkloadDisruption = pointer.P(true)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(1))
			CreateMigrationPolicy(virtClient, policy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Logging in and running stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			Expect(runParticularStressTest(vmi, stressSize, "write64", 1*time.Second)).To(Succeed())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Waiting for stall detection to trigger")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying paused migration mode is entered")
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPaused, 5*time.Minute)

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)

			By("Verifying the VMI is unpaused after migration")
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
		})
	})

	Context("with post-copy and workload disruption both disabled", func() {
		It("should detect stall and abort migration when convergence is impossible", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(false)
			policy.Spec.AllowWorkloadDisruption = pointer.P(false)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(1))
			CreateMigrationPolicy(virtClient, policy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Logging in and running stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			Expect(runParticularStressTest(vmi, stressSize, "write64", 1*time.Second)).To(Succeed())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Waiting for stall detection to trigger")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying impossible convergence conditions cause migration to abort")
			libmigration.ExpectMigrationAbortStatusToBe(migration, v1.MigrationAbortSucceeded, 1*time.Minute)

			By("Verifying migration enters Failed phase")
			Eventually(matcher.ThisMigration(migration), 60*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed))

			By("Verifying the VMI remains running")
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
		})
	})

	Context("with completion timeout", func() {
		It("should abort migration when completion timeout is reached without convergence options", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(false)
			policy.Spec.AllowWorkloadDisruption = pointer.P(false)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(1))
			policy.Spec.CompletionTimeoutPerGiB = pointer.P(int64(5))
			CreateMigrationPolicy(virtClient, policy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Verifying completion timeout causes migration to abort")
			libmigration.ExpectMigrationAbortStatusToBe(migration, v1.MigrationAbortSucceeded, stallDetectionTimeout)

			By("Verifying migration enters Failed phase")
			Eventually(matcher.ThisMigration(migration), 60*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed))

			By("Verifying the VMI remains running")
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
		})
	})
}))
