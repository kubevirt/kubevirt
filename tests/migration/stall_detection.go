package migration

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/migrations/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/testsuite"

	kvpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

const (
	stressSizeHeavy          = "50%"
	stressSizeLight          = "25%"
	stallDetectionTimeout    = 5 * time.Minute
	migrationCompleteTimeout = 300

	stallDetectedString = "stall detected: bestRemainingBytes=([0-9.]+)Mib outsideWindowMin=([0-9.]+)Mib candidates=([0-9]+)"
	relaxationString    = "relaxed best remaining bytes: oldBest=([0-9.]+)Mib newBest=([0-9.]+)Mib iterElapsedMs=([0-9]+)ms nextPatienceMs=([0-9]+)ms nextDeadlineMs=([0-9]+)ms"
	abortTimeoutString  = "aborting migration due to completion timeout: elapsedSec=([0-9]+)s acceptableCompletionSec=([0-9]+)s"
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

// startPeriodicDirtyRateStress spawns additional stress-ng instances on an interval until stop is closed.
// It is used to ensure the dirty rate is consistent increasing to reliably trigger "relaxation".
func startPeriodicDirtyRateStress(vmi *v1.VirtualMachineInstance, stressSize string, interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				runParticularStressTest(vmi, stressSize, "write64", 0)
			}
		}
	}()
	return stop
}

func getVirtLauncherSourceLogs(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	uid := vmi.GetObjectMeta().GetUID()

	labelSelector := fmt.Sprintf("%s=%s", v1.CreatedByLabel, string(uid))
	pods, err := virtClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Should list pods")

	podName := ""
	for i := range pods.Items {
		if pods.Items[i].ObjectMeta.DeletionTimestamp != nil {
			continue
		}
		if _, isMigrationTarget := pods.Items[i].Labels[v1.MigrationJobLabel]; isMigrationTarget {
			continue
		}
		podName = pods.Items[i].ObjectMeta.Name
		break
	}
	ExpectWithOffset(1, podName).ToNot(BeEmpty(), "Should find pod not scheduled for deletion")

	logsRaw, err := virtClient.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			Container: "compute",
		}).
		DoRaw(context.Background())
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Should get virt-launcher pod logs")

	return string(logsRaw)
}

var _ = Describe(SIG("Migration Stall Detection", Serial, decorators.RequiresTwoSchedulableNodes, func() {
	var virtClient kubecli.KubevirtClient
	var migrationPolicy *v1alpha1.MigrationPolicy

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		kvconfig.EnableFeatureGate(featuregate.MigrationStallDetection)
		kvconfig.EnableFeatureGate(featuregate.AdvancedMigrationOptions)
		config := getCurrentKvConfig(virtClient)
		if config.DeveloperConfiguration.LogVerbosity == nil {
			config.DeveloperConfiguration.LogVerbosity = &v1.LogVerbosity{}
		}
		config.DeveloperConfiguration.LogVerbosity.VirtLauncher = 4
		kvconfig.UpdateKubeVirtConfigValueAndWait(config)
		policyName := fmt.Sprintf("testpolicy-%s", rand.String(5))
		migrationPolicy = kubecli.NewMinimalMigrationPolicy(policyName)
		migrationPolicy.Spec.AdvancedMigrationOptions = &v1.AdvancedMigrationOptions{
			// As of QEMU 10.1.0, bandwidth per migration is not correctly respect when using multifd
			//  see https://gitlab.com/qemu-project/qemu/-/work_items/3364
			// By setting ParallelMigrationThreads to 0, we disable multifd and thus bandwidth
			//  per migration is respected.
			ParallelMigrationThreads: kvpointer.P(uint(0)),
			StallDetector: &v1.StallDetectorOptions{
				StallProgressTimeout: kvpointer.P(uint64(5)),
				SwitchoverTimeout:    kvpointer.P(uint64(45)),
				// improves convergence odds before timeout
				StallMargin: kvpointer.P(float64(0.07)),
			},
		}
		migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(int64(250))
		// this migration speed is a healthy balance between the tests finishing in a reasonable amount of time
		// yet also ensure migration takes long enough that we can react to changes in time
		migrationPolicy.Spec.BandwidthPerMigration = kvpointer.P(resource.MustParse("15Mi"))

		DeferCleanup(func() {
			err := virtClient.MigrationPolicy().Delete(context.Background(), migrationPolicy.Name, metav1.DeleteOptions{})
			if err != nil {
				fmt.Fprintf(GinkgoWriter, "cleanup: failed to delete migration policy %q: %v\n", migrationPolicy.Name, err)
			}
		})
	})

	Context("with post-copy enabled", func() {
		It("should detect stall and switch to post-copy", func() {
			migrationPolicy.Spec.AllowPostCopy = kvpointer.P(true)
			migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(true)
			migrationPolicy.Spec.MaxDowntime = kvpointer.P(uint64(1))

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi.Namespace = testsuite.NamespacePrivileged

			By("Creating the MigrationPolicy")
			AlignPolicyAndVmi(vmi, migrationPolicy)
			CreateMigrationPolicy(virtClient, migrationPolicy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Logging in and running stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			runParticularStressTest(vmi, stressSizeHeavy, "write64", 1*time.Second)

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
			migrationPolicy.Spec.AllowPostCopy = kvpointer.P(false)
			migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(true)
			migrationPolicy.Spec.MaxDowntime = kvpointer.P(uint64(1))

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Creating the MigrationPolicy")
			AlignPolicyAndVmi(vmi, migrationPolicy)
			CreateMigrationPolicy(virtClient, migrationPolicy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Logging in and running stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			runParticularStressTest(vmi, stressSizeHeavy, "write64", 1*time.Second)

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
			migrationPolicy.Spec.AllowPostCopy = kvpointer.P(false)
			migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(false)
			migrationPolicy.Spec.MaxDowntime = kvpointer.P(uint64(400))
			// ensures migration won't be aborted for being too far off
			migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.PrecopyPossibleFactor = kvpointer.P(float64(1000))

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Creating the MigrationPolicy")
			AlignPolicyAndVmi(vmi, migrationPolicy)
			CreateMigrationPolicy(virtClient, migrationPolicy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Logging in and running aggressive stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			runParticularStressTest(vmi, stressSizeHeavy, "write64", 1*time.Second)

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
			}, stallDetectionTimeout, 5*time.Second).Should(MatchRegexp(maxDowntimeSetString(*migrationPolicy.Spec.MaxDowntime, *migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.PrecopyPossibleFactor)))

			By("Waiting for migration to complete successfully")
			libmigration.ExpectMigrationToSucceed(virtClient, migration, migrationCompleteTimeout)
		})
	})

	Context("with relaxation mechanism", func() {
		It("should relax best remaining bytes when workload ramps up after initial progress", func() {
			migrationPolicy.Spec.AllowPostCopy = kvpointer.P(false)
			migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(true)
			migrationPolicy.Spec.MaxDowntime = kvpointer.P(uint64(10))
			migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.PrecopyPossibleFactor = kvpointer.P(float64(10000))
			migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.StallMargin = kvpointer.P(float64(0.00))
			migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(int64(500)) // this test could take longer than other tests

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Creating the MigrationPolicy")
			AlignPolicyAndVmi(vmi, migrationPolicy)
			CreateMigrationPolicy(virtClient, migrationPolicy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Logging in with light initial stress")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			runParticularStressTest(vmi, stressSizeLight, "write64", 1*time.Second)

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration), 45*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))
			libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPreCopy, 5*time.Second)

			By("Ramping up stress while waiting for stall detection")
			stopStressRamp := startPeriodicDirtyRateStress(vmi, "3%", 4*time.Second)

			By("Waiting for stall detection to trigger")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(stallDetectedString))

			By("Verifying relaxation mechanism triggers")
			Eventually(func() string {
				return getVirtLauncherSourceLogs(virtClient, vmi)
			}, stallDetectionTimeout, 3*time.Second).Should(MatchRegexp(relaxationString))

			close(stopStressRamp)

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
			migrationPolicy.Spec.AllowPostCopy = kvpointer.P(false)
			migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(false)
			migrationPolicy.Spec.MaxDowntime = kvpointer.P(uint64(1))
			migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.PrecopyPossibleFactor = kvpointer.P(float64(1))
			// without this, migration won't abort instantly since it would think
			migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.CompletionTimeoutFactor = kvpointer.P(float64(100000))

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Creating the MigrationPolicy")
			AlignPolicyAndVmi(vmi, migrationPolicy)
			CreateMigrationPolicy(virtClient, migrationPolicy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Logging in and running stress-ng")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			runParticularStressTest(vmi, stressSizeHeavy, "write64", 1*time.Second)

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
			}, stallDetectionTimeout, 3*time.Second).Should(
				MatchRegexp(abortImpossibleString(
					*migrationPolicy.Spec.MaxDowntime,
					*migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.PrecopyPossibleFactor,
				)),
			)

			By("Verifying migration enters Failed phase")
			Eventually(matcher.ThisMigration(migration), 60*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed))

			By("Verifying the VMI remains running")
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
		})
	})

	Context("with completion timeout", func() {
		setCompletionTimeoutTestPolicy := func() {
			migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(int64(5))
			migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.StallProgressTimeout = kvpointer.P(uint64(600))
			// prevents test from aborting because we would exceed the completion timeout several times over
			migrationPolicy.Spec.AdvancedMigrationOptions.StallDetector.CompletionTimeoutFactor = kvpointer.P(100.0)
		}

		It("should force switchover when completion timeout is reached", func() {
			migrationPolicy.Spec.AllowPostCopy = kvpointer.P(false)
			migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(true)
			migrationPolicy.Spec.MaxDowntime = kvpointer.P(uint64(1))
			setCompletionTimeoutTestPolicy()

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Creating the MigrationPolicy")
			AlignPolicyAndVmi(vmi, migrationPolicy)
			CreateMigrationPolicy(virtClient, migrationPolicy)

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
			migrationPolicy.Spec.AllowPostCopy = kvpointer.P(true)
			migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(true)
			migrationPolicy.Spec.MaxDowntime = kvpointer.P(uint64(1))
			setCompletionTimeoutTestPolicy()

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi.Namespace = testsuite.NamespacePrivileged

			By("Creating the MigrationPolicy")
			AlignPolicyAndVmi(vmi, migrationPolicy)
			CreateMigrationPolicy(virtClient, migrationPolicy)

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
			migrationPolicy.Spec.AllowPostCopy = kvpointer.P(false)
			migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(false)
			migrationPolicy.Spec.MaxDowntime = kvpointer.P(uint64(1))
			setCompletionTimeoutTestPolicy()

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			By("Creating the MigrationPolicy")
			AlignPolicyAndVmi(vmi, migrationPolicy)
			CreateMigrationPolicy(virtClient, migrationPolicy)

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
