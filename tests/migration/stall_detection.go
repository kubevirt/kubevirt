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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpodmutator"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

const (
	stressSizeHeavy          = "50%"
	stressSizeLight          = "25%"
	stallDetectionTimeout    = 5 * time.Minute
	migrationCompleteTimeout = 300

	stallDetectionWebhookName       = "test-stall-detection-mutator"
	stallDetectionWebhookSecretName = "webhook-certs-stall-detection"
	stallDetectionWebhookPort       = 8443
	stallDetectionConfigMapName     = "stall-detection-config"

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

// migrationTestEnv returns the base migration and stall-detector env configuration for these
// tests, merged with any per-test overrides.
func migrationTestEnv(overrides map[string]string) map[string]string {
	env := map[string]string{
		// As of QEMU 10.1.0, bandwidth per migration is not correctly respected when using multifd
		//  see https://gitlab.com/qemu-project/qemu/-/work_items/3364
		// By setting parallel threads to 0, we disable multifd and thus bandwidth per migration is respected.
		migrations.EnvParallelMigrationThreads:  "0",
		migrations.EnvBandwidthPerMigration:     "15Mi",
		migrations.EnvCompletionTimeoutPerGiB:   "250",
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

// startPeriodicDirtyRateStress spawns stress-ng on an interval until the returned stop func is called.
// The stop func is idempotent, waits for the background goroutine to exit, and returns any stress error.
func startPeriodicDirtyRateStress(vmi *v1.VirtualMachineInstance, stressSize string, interval time.Duration) func() error {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	var stressErr error
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := runParticularStressTest(vmi, stressSize, "write64", 0); err != nil && ctx.Err() == nil {
					stressErr = err
				}
			}
		}
	}()
	var once sync.Once
	return func() error {
		once.Do(func() {
			cancel()
			wg.Wait()
		})
		return stressErr
	}
}

func getVirtLauncherSourceLogs(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	GinkgoHelper()
	freshVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred(), "should get VMI")
	Expect(freshVMI.Status.MigrationState).ToNot(BeNil())
	sourcePod := freshVMI.Status.MigrationState.SourcePod
	Expect(sourcePod).ToNot(BeEmpty(), "source pod name must be populated during migration")
	logsRaw, err := virtClient.CoreV1().
		Pods(vmi.Namespace).
		GetLogs(sourcePod, &k8sv1.PodLogOptions{Container: "compute"}).
		DoRaw(context.Background())
	Expect(err).ToNot(HaveOccurred(), "should get virt-launcher source pod logs")
	return string(logsRaw)
}

var _ = Describe(SIG("Migration Stall Detection", Serial, decorators.RequiresTwoSchedulableNodes, func() {
	var virtClient kubecli.KubevirtClient
	var envWebhook *libpodmutator.Webhook
	var stallTestNamespace string

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		stallTestNamespace = testsuite.GetTestNamespace(nil)
		kvconfig.EnableFeatureGate(featuregate.MigrationStallDetection)
		config := getCurrentKvConfig(virtClient)
		if config.DeveloperConfiguration.LogVerbosity == nil {
			config.DeveloperConfiguration.LogVerbosity = &v1.LogVerbosity{}
		}
		config.DeveloperConfiguration.LogVerbosity.VirtLauncher = 4
		kvconfig.UpdateKubeVirtConfigValueAndWait(config)

		libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(nil), stallTestNamespace)
		envWebhook = libpodmutator.Setup(libpodmutator.Options{
			Name:             stallDetectionWebhookName,
			SecretName:       stallDetectionWebhookSecretName,
			Port:             stallDetectionWebhookPort,
			Namespace:        stallTestNamespace,
			EnvFromConfigMap: stallDetectionConfigMapName,
		})
	})

	AfterEach(func() {
		libpodmutator.Teardown(envWebhook, stallDetectionWebhookSecretName)
		libpodmutator.DeleteEnvConfigMap(virtClient, stallDetectionConfigMapName, stallTestNamespace)
	})

	Context("with post-copy enabled", func() {
		It("should detect stall and switch to post-copy", func() {
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvAllowPostCopy:           "true",
				migrations.EnvAllowWorkloadDisruption: "true",
				migrations.EnvMaxDowntimeMs:           "1",
			}), stallTestNamespace)

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

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
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying post-copy mode is entered")
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPostCopy, 5*time.Minute)

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})
	})

	Context("with workload disruption allowed and post-copy disabled", func() {
		It("should detect stall and force switchover via max downtime", func() {
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvAllowPostCopy:           "false",
				migrations.EnvAllowWorkloadDisruption: "true",
				migrations.EnvMaxDowntimeMs:           "1",
			}), stallTestNamespace)

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

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
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying forced switchover log message")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
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
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvAllowPostCopy:           "false",
				migrations.EnvAllowWorkloadDisruption: "false",
				migrations.EnvMaxDowntimeMs:           "400",
				migrations.EnvPrecopyPossibleFactor:   "1000",
			}), stallTestNamespace)

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

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
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
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
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(maxDowntimeSetString(400, 1000)))

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})
	})

	Context("with relaxation mechanism", func() {
		It("should relax best remaining bytes when workload ramps up after initial progress", func() {
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvAllowPostCopy:           "false",
				migrations.EnvAllowWorkloadDisruption: "true",
				migrations.EnvMaxDowntimeMs:           "1",
				migrations.EnvCompletionTimeoutPerGiB: "500",
				migrations.EnvPrecopyPossibleFactor:   "10000",
				migrations.EnvStallMargin:             "0.00",
				migrations.EnvCompletionTimeoutFactor: "100",
			}), stallTestNamespace)

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

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
			DeferCleanup(func() {
				// stopStressRamp must be called in defer to ensure the go-routine exists
				// even if this test fails
				Expect(stopStressRamp()).To(Succeed())
			})

			By("Waiting for stall detection to trigger")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying relaxation mechanism triggers")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(relaxationString))

			By("Stopping stress ramp so migration can converge")
			Expect(stopStressRamp()).To(Succeed())

			By("Verifying oldBest < newBest in relaxation log")
			logs := getVirtLauncherSourceLogs(virtClient, vmi)
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
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(forceSwitchoverString))

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})
	})

	Context("with post-copy and workload disruption both disabled", func() {
		It("should detect stall and abort migration when convergence is impossible", func() {
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvAllowPostCopy:           "false",
				migrations.EnvAllowWorkloadDisruption: "false",
				migrations.EnvMaxDowntimeMs:           "1",
				migrations.EnvPrecopyPossibleFactor:   "1",
				migrations.EnvCompletionTimeoutFactor: "100000",
			}), stallTestNamespace)

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

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
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying migration abort log message")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
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
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvAllowPostCopy:           "false",
				migrations.EnvAllowWorkloadDisruption: "true",
				migrations.EnvMaxDowntimeMs:           "1",
				migrations.EnvCompletionTimeoutPerGiB: "5",
				migrations.EnvStallProgressTimeout:    "600",
				migrations.EnvCompletionTimeoutFactor: "100",
			}), stallTestNamespace)

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Verifying completion timeout triggers forced switchover")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(completionTimeoutForceSwitchoverString))

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})

		It("should switch to post-copy when completion timeout is reached", func() {
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvAllowPostCopy:           "true",
				migrations.EnvAllowWorkloadDisruption: "true",
				migrations.EnvMaxDowntimeMs:           "1",
				migrations.EnvCompletionTimeoutPerGiB: "5",
				migrations.EnvStallProgressTimeout:    "600",
				migrations.EnvCompletionTimeoutFactor: "100",
			}), stallTestNamespace)

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Verifying completion timeout triggers post-copy")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 5*time.Second).Should(ContainSubstring(completionTimeoutPostCopyString))

			By("Verifying post-copy mode is entered")
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPostCopy, 5*time.Minute)

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})

		It("should abort migration when completion timeout is reached without convergence options", func() {
			libpodmutator.CreateOrUpdateEnvConfigMap(virtClient, stallDetectionConfigMapName, migrationTestEnv(map[string]string{
				migrations.EnvAllowPostCopy:           "false",
				migrations.EnvAllowWorkloadDisruption: "false",
				migrations.EnvMaxDowntimeMs:           "1",
				migrations.EnvCompletionTimeoutPerGiB: "5",
				migrations.EnvStallProgressTimeout:    "600",
				migrations.EnvCompletionTimeoutFactor: "100",
			}), stallTestNamespace)

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Verifying completion timeout abort log message")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
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
