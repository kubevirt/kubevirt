package migration

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/migrations"
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
	"kubevirt.io/kubevirt/tests/libpodmutator"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	stressSizeHeavy          = "50%"
	stressSizeLight          = "25%"
	stallDetectionTimeout    = 5 * time.Minute
	migrationCompleteTimeout = 300

	stallDetectionPolicyName    = "stall-detection-env-inject"
	stallDetectionConfigMapName = "stall-detection-config"

	stallDetectedString = "stall detected: bestRemainingBytes=([0-9.]+)Mib outsideWindowMin=([0-9.]+)Mib candidates=([0-9]+)"
	relaxationString    = "relaxed best remaining bytes: oldBest=([0-9.]+)Mib newBest=([0-9.]+)Mib iterElapsedMs=([0-9]+)ms nextPatienceMs=([0-9]+)ms nextDeadlineMs=([0-9]+)ms"
	abortTimeoutString  = "aborting migration due to completion timeout: elapsedSec=([0-9]+) acceptableCompletionSec=([0-9]+)"
)

var forceSwitchoverString = fmt.Sprintf("forcing switchover by setting max downtime to %dms: estimated downtime [0-9.]+ms is a local minima", migrations.QEMUMaxMigrationDowntimeMS)
var completionTimeoutForceSwitchoverString = fmt.Sprintf("completion timeout reached: setting max downtime to %dms to force switchover", migrations.QEMUMaxMigrationDowntimeMS)
var completionTimeoutPostCopyString = "completion timeout reached: starting post-copy mode to force convergence"

func maxDowntimeSetString(maxDowntime uint64, factor float64) string {
	return fmt.Sprintf("max downtime set to %dms: estimated downtime ([0-9]+)ms within tolerable factor %.2fx to max allowed downtime %dms", maxDowntime, factor, maxDowntime)
}
func abortImpossibleString(downtime uint64, factor float64) string {
	return fmt.Sprintf("aborting migration: estimated downtime [0-9]+ms exceeds max allowed downtime %dms by a factor of more than x%.2f", downtime, factor)
}

// migrationTestEnv returns stall-detector and multifd env overrides for e2e.
func migrationTestEnv(overrides map[string]string) map[string]string {
	env := map[string]string{
		// As of QEMU 10.1.0, bandwidth per migration is not correctly respected when using multifd
		//  see https://gitlab.com/qemu-project/qemu/-/work_items/3364
		// This override exists to allow disabling multifd so BandwidthPerMigration is honored.
		migrations.EnvDisableMultifd:            "true",
		migrations.EnvStallProgressTimeout:      "5",
		migrations.EnvSwitchoverTimeout:         "30",
		migrations.EnvStallMargin:               "0.07",
		migrations.EnvEwmaAlpha:                 "0.4",
		migrations.EnvPrecopyPossibleFactor:     "1.5",
		migrations.EnvPatienceWindowDecayFactor: "0.5",
		migrations.EnvSearchLocalMinima:         "true",
		migrations.EnvCompletionTimeoutFactor:   "2",
	}
	for key, value := range overrides {
		env[key] = value
	}
	return env
}

func newStallMigrationPolicy(vmi *v1.VirtualMachineInstance) *migrationsv1.MigrationPolicy {
	policy := GeneratePolicyAndAlignVMI(vmi)
	policy.Spec.BandwidthPerMigration = pointer.P(resource.MustParse("15Mi"))
	policy.Spec.CompletionTimeoutPerGiB = pointer.P(int64(250))
	return policy
}

// startPeriodicDirtyRateStress spawns stress-ng on an interval until the returned stop func is called.
// The stop func is idempotent and waits for the background goroutine to exit.
// Console errors during the ramp are ignored: the guest may migrate away mid-command, and
// stall/relaxation log assertions (plus migration success) are the real success criteria.
func startPeriodicDirtyRateStress(vmi *v1.VirtualMachineInstance, stressSize string, interval time.Duration) func() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = runParticularStressTest(vmi, stressSize, "write64", 0)
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			wg.Wait()
		})
	}
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
	var envPolicy *libpodmutator.EnvInjectionPolicy
	var stallTestNamespace string

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		enabled, err := util.IsMutatingAdmissionPolicyEnabled(virtClient)
		Expect(err).ToNot(HaveOccurred())
		if !enabled {
			Fail("MutatingAdmissionPolicy not available in this cluster")
		}

		stallTestNamespace = testsuite.GetTestNamespace(nil)
		kvconfig.EnableFeatureGate(featuregate.MigrationStallDetection)
		config := getCurrentKvConfig(virtClient)
		if config.DeveloperConfiguration.LogVerbosity == nil {
			config.DeveloperConfiguration.LogVerbosity = &v1.LogVerbosity{}
		}
		config.DeveloperConfiguration.LogVerbosity.VirtLauncher = 4
		kvconfig.UpdateKubeVirtConfigValueAndWait(config)

		libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(nil), stallTestNamespace)
		envPolicy = libpodmutator.SetupEnvInjectionPolicy(libpodmutator.EnvInjectionPolicyOptions{
			Name:          stallDetectionPolicyName,
			ConfigMapName: stallDetectionConfigMapName,
			Namespace:     stallTestNamespace,
		})
	})

	AfterEach(func() {
		libpodmutator.TeardownEnvInjectionPolicy(envPolicy)
		libpodmutator.DeleteEnvConfigMap(virtClient, stallDetectionConfigMapName, stallTestNamespace)
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
			Expect(runParticularStressTest(vmi, stressSizeHeavy, "write64", 1*time.Second)).To(Succeed())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Waiting for stall detection to trigger")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(stallDetectedString))

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
			Expect(runParticularStressTest(vmi, stressSizeHeavy, "write64", 1*time.Second)).To(Succeed())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Waiting for stall detection to trigger")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying forced switchover log message")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(forceSwitchoverString))

			By("Verifying paused migration mode is entered")
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPaused, 5*time.Minute)

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)

			By("Verifying the VMI is unpaused after migration")
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
		})
	})

	Context("with natural convergence after reducing dirty rate", func() {
		It("should detect stall then converge naturally after stress is stopped", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(false)
			policy.Spec.AllowWorkloadDisruption = pointer.P(false)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(400))
			CreateMigrationPolicy(virtClient, policy)
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvPrecopyPossibleFactor: "1000",
			}), stallTestNamespace)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Logging in and running aggressive stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			Expect(runParticularStressTest(vmi, stressSizeHeavy, "write64", 1*time.Second)).To(Succeed())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Waiting for stall detection to trigger")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Killing stress-ng to allow migration to converge")
			// don't validate the command executed since it's possible that the VM would've migrated away by the time we
			// hear back from the command. Therefore, the migration succeeding is implicit validation that this executed.
			_ = console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "pkill -9 stress-ng\n"},
				&expect.BExp{R: ""},
			}, 40)

			By("Verifying natural switchover attempt log message")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(maxDowntimeSetString(400, 1000)))

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})
	})

	Context("with relaxation mechanism", func() {
		It("should relax best remaining bytes when workload ramps up after initial progress", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(false)
			policy.Spec.AllowWorkloadDisruption = pointer.P(true)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(1))
			policy.Spec.CompletionTimeoutPerGiB = pointer.P(int64(500))
			CreateMigrationPolicy(virtClient, policy)
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvPrecopyPossibleFactor:   "10000",
				migrations.EnvStallMargin:             "0.00",
				migrations.EnvCompletionTimeoutFactor: "100",
			}), stallTestNamespace)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Logging in with light initial stress")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			Expect(runParticularStressTest(vmi, stressSizeLight, "write64", 1*time.Second)).To(Succeed())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Ramping up stress while waiting for stall detection")
			stopStressRamp := startPeriodicDirtyRateStress(vmi, "3%", 4*time.Second)
			// Ensure the ramp goroutine exits even if this test fails mid-way.
			DeferCleanup(stopStressRamp)

			By("Waiting for stall detection to trigger")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying relaxation mechanism triggers")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(relaxationString))

			By("Stopping stress ramp so migration can converge")
			stopStressRamp()

			By("Verifying oldBest < newBest in relaxation log")
			logs := getVirtLauncherSourceLogs(Default, virtClient, vmi)
			re := regexp.MustCompile(relaxationString)
			matches := re.FindStringSubmatch(logs)
			Expect(len(matches)).To(BeNumerically(">", 1))
			oldBest, err := strconv.ParseFloat(matches[1], 64)
			Expect(err).ToNot(HaveOccurred())
			newBest, err := strconv.ParseFloat(matches[2], 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(newBest).To(BeNumerically(">", oldBest),
				fmt.Sprintf("newBest (%f) should be greater than oldBest (%f) after relaxation", newBest, oldBest))

			// since we are no longer spawning more stressors, relaxation should eventually catch up and converge
			By("Verifying forced switchover after stall relaxation")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(forceSwitchoverString))

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
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
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvPrecopyPossibleFactor:   "1",
				migrations.EnvCompletionTimeoutFactor: "100000",
			}), stallTestNamespace)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Logging in and running stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			Expect(runParticularStressTest(vmi, stressSizeHeavy, "write64", 1*time.Second)).To(Succeed())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Waiting for stall detection to trigger")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying migration abort log message")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(abortImpossibleString(1, 1)))

			By("Verifying migration enters Failed phase")
			Eventually(matcher.ThisMigration(migration), 60*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed))

			By("Verifying the VMI remains running")
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
		})
	})

	Context("with completion timeout", func() {
		It("should force switchover when completion timeout is reached", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(false)
			policy.Spec.AllowWorkloadDisruption = pointer.P(true)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(1))
			policy.Spec.CompletionTimeoutPerGiB = pointer.P(int64(5))
			CreateMigrationPolicy(virtClient, policy)
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvStallProgressTimeout:    "600",
				migrations.EnvCompletionTimeoutFactor: "100",
			}), stallTestNamespace)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Verifying completion timeout triggers forced switchover")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(completionTimeoutForceSwitchoverString))

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})

		It("should switch to post-copy when completion timeout is reached", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(true)
			policy.Spec.AllowWorkloadDisruption = pointer.P(true)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(1))
			policy.Spec.CompletionTimeoutPerGiB = pointer.P(int64(5))
			CreateMigrationPolicy(virtClient, policy)
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvStallProgressTimeout:    "600",
				migrations.EnvCompletionTimeoutFactor: "100",
			}), stallTestNamespace)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Verifying completion timeout triggers post-copy")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(ContainSubstring(completionTimeoutPostCopyString))

			By("Verifying post-copy mode is entered")
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPostCopy, 5*time.Minute)

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})

		It("should abort migration when completion timeout is reached without convergence options", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			policy := newStallMigrationPolicy(vmi)
			policy.Spec.AllowPostCopy = pointer.P(false)
			policy.Spec.AllowWorkloadDisruption = pointer.P(false)
			policy.Spec.MaxDowntimeMs = pointer.P(uint64(1))
			policy.Spec.CompletionTimeoutPerGiB = pointer.P(int64(5))
			CreateMigrationPolicy(virtClient, policy)
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvStallProgressTimeout:    "600",
				migrations.EnvCompletionTimeoutFactor: "100",
			}), stallTestNamespace)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Verifying completion timeout abort log message")
			Eventually(func(g Gomega) string {
				return getVirtLauncherSourceLogs(g, virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(abortTimeoutString))

			By("Verifying migration enters Failed phase")
			Eventually(matcher.ThisMigration(migration), 60*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed))

			By("Verifying the VMI remains running")
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
		})
	})
}))
