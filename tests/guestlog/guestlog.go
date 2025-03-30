package guestlog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	cirrosStartupTimeout = 60
	testString           = "GuestConsoleTest3413254123535234523"
)

var _ = Describe("[sig-compute]Guest console log", decorators.SigCompute, func() {
	var cirrosVmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		cirrosVmi = libvmifact.NewCirros(libvmi.WithLogSerialConsole(true))
		cirrosVmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(true)
	})

	Describe("[level:component] Guest console log container", func() {
		Context("set LogSerialConsole", func() {
			It("it should exit cleanly when the shutdown is initiated by the guest", func() {
				By("Starting a VMI")
				vmi := libvmops.RunVMIAndExpectLaunch(cirrosVmi, cirrosStartupTimeout)

				By("Finding virt-launcher pod")
				virtlauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())
				foundContainer := false
				for _, container := range virtlauncherPod.Spec.Containers {
					if container.Name == "guest-console-log" {
						foundContainer = true
					}
				}
				Expect(foundContainer).To(BeTrue())

				By("Triggering a shutdown from the guest OS")
				Expect(console.LoginToCirros(vmi)).To(Succeed())
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo poweroff\n"},
					&expect.BExp{R: "The system is going down NOW!"},
				}, 240)).To(Succeed())

				By("Ensuring virt-launcher pod is not reporting errors")
				Eventually(func(g Gomega) {
					virtlauncherPod, err := matcher.ThisPod(virtlauncherPod)()
					g.Expect(err).ToNot(HaveOccurred())
					Expect(virtlauncherPod).ToNot(matcher.BeInPhase(k8sv1.PodFailed))
					g.Expect(virtlauncherPod).To(matcher.BeInPhase(k8sv1.PodSucceeded))
				}, 60*time.Second, 1*time.Second).Should(Succeed(), "virt-launcher should reach the PodSucceeded phase never hitting the PodFailed one")
			})
		})

		Context("fetch logs", func() {
			var vmi *v1.VirtualMachineInstance

			cirrosLogo := `
  ____               ____  ____
 / __/ __ ____ ____ / __ \/ __/
/ /__ / // __// __// /_/ /\ \ 
\___//_//_/  /_/   \____/___/ 
   http://cirros-cloud.net
`

			It("[QUARANTINE] it should fetch logs for a running VM with logs API", decorators.Quarantine, func() {
				vmi = libvmops.RunVMIAndExpectLaunch(cirrosVmi, cirrosStartupTimeout)

				By("Finding virt-launcher pod")
				virtlauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				By("Getting logs with logs API and ensure the logs are correctly ordered with no unexpected line breaks")

				Eventually(func(g Gomega) bool {
					logs, err := getConsoleLogs(virtlauncherPod)
					g.Expect(err).ToNot(HaveOccurred())
					return strings.Contains(logs, cirrosLogo)
				}, cirrosStartupTimeout*time.Second, 2*time.Second).Should(BeTrue())

				By("Obtaining the serial console, logging in and executing a command there")
				Expect(console.LoginToCirros(vmi)).To(Succeed())
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "echo " + testString + "\n"},
					&expect.BExp{R: testString},
				}, 240)).To(Succeed())

				By("Ensuring that log fetching is not breaking an open console")
				expecter, errChan, eerr := console.NewExpecter(kubevirt.Client(), vmi, 90*time.Second)
				Expect(eerr).ToNot(HaveOccurred())
				if eerr == nil {
					defer func() {
						derr := expecter.Close()
						Expect(derr).ToNot(HaveOccurred())
					}()
				}

				Consistently(errChan).ShouldNot(Receive())

				logs, err := getConsoleLogs(virtlauncherPod)
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring that logs contain the login attempt")
				Expect(logs).To(ContainSubstring(vmi.Name + " login: cirros"))

				// TODO: console.LoginToCirros is not systematically waiting for `\u001b[8m` to prevent echoing the password, fix it first
				// By("Ensuring that logs don't contain the login password")
				// Expect(outputString).ToNot(ContainSubstring("Password: gocubsgo"))

				By("Ensuring that logs contain the test command and its output")
				Expect(logs).To(ContainSubstring("echo " + testString + "\n"))
				Expect(logs).To(ContainSubstring("\n" + testString + "\n"))
			})

			It("it should rotate the internal log files", decorators.Periodic, func() {
				vmi = libvmops.RunVMIAndExpectLaunch(cirrosVmi, cirrosStartupTimeout)

				By("Finding virt-launcher pod")
				virtlauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				By("Generating 9MB of log data to force log rotation and discarding")
				generateHugeLogData(vmi, 9)

				By("Ensuring that log fetching is not failing")
				_, err = getConsoleLogs(virtlauncherPod)
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring that we have 4 rotated log files (+term one)")
				outputString, err := exec.ExecuteCommandOnPod(virtlauncherPod, "guest-console-log", []string{"/bin/ls", "-l", fmt.Sprintf("/var/run/kubevirt-private/%v", vmi.UID)})
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Count(outputString, "virt-serial0-log")).To(Equal(4 + 1))
			})

			It("[QUARANTINE] it should not skip any log line even trying to flood the serial console for QOSGuaranteed VMs", decorators.Quarantine, decorators.Periodic, func() {
				cirrosVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceCPU:    resource.MustParse("1000m"),
						k8sv1.ResourceMemory: resource.MustParse("256M"),
					},
					Limits: k8sv1.ResourceList{
						k8sv1.ResourceCPU:    resource.MustParse("1000m"),
						k8sv1.ResourceMemory: resource.MustParse("256M"),
					},
				}
				vmi = libvmops.RunVMIAndExpectLaunch(cirrosVmi, cirrosStartupTimeout)
				Expect(vmi.Status.QOSClass).ToNot(BeNil())
				Expect(*vmi.Status.QOSClass).To(Equal(k8sv1.PodQOSGuaranteed))

				By("Finding virt-launcher pod")
				virtlauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				By("Generating 1MB of log data")
				generateHugeLogData(vmi, 1)

				By("Ensuring that log fetching is not failing")
				logs, err := getConsoleLogs(virtlauncherPod)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that log lines are sequential with no gaps")
				outputLines := strings.Split(logs, "\n")
				Expect(len(outputLines)).To(BeNumerically(">", 1000))
				matchingLines := 0
				prevSeqn := -1
				prevLine := ""
				r, err := regexp.Compile(`^logline (?P<seqn>\d{7})\s*`)
				Expect(err).ToNot(HaveOccurred())
				seqnIndex := r.SubexpIndex("seqn")
				for _, line := range outputLines {
					if matches := r.FindStringSubmatch(line); len(matches) > seqnIndex {
						seqnString := matches[seqnIndex]
						i, err := strconv.Atoi(seqnString)
						Expect(err).ToNot(HaveOccurred())
						if prevSeqn > 0 {
							Expect(i).To(Equal(prevSeqn+1), fmt.Sprintf("log line seq number should match previous+1: previous %d, current: %d.\nprevLine: %s\nline: %s", prevSeqn, i, line, prevLine))
						}
						prevSeqn = i
						prevLine = line
						matchingLines = matchingLines + 1
					}
				}
				Expect(matchingLines).To(BeNumerically(">", 1000))
			})

			AfterEach(func() {
				if CurrentSpecReport().Failed() {
					if vmi != nil {
						virtlauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
						if err == nil {
							artifactsDir, _ := os.LookupEnv("ARTIFACTS")
							outputString, err := exec.ExecuteCommandOnPod(virtlauncherPod, "guest-console-log", []string{"/bin/bash", "-c", "/bin/tail -v -n +1 " + fmt.Sprintf("/var/run/kubevirt-private/%v/virt-serial*-log*", vmi.UID)})
							if err == nil {
								lpath := filepath.Join(artifactsDir, fmt.Sprintf("serial_logs_content_%v.txt", vmi.UID))
								_, _ = fmt.Fprintf(GinkgoWriter, "Serial console log failed, serial console logs dump from virt-launcher pod collected at file at %s\n", lpath)
								_ = os.WriteFile(lpath, []byte(outputString), 0o644)
							}
						}
					}
				}
			})
		})
	})
})

func generateHugeLogData(vmi *v1.VirtualMachineInstance, mb int) {
	By("Obtaining the serial console, logging in")
	Expect(console.LoginToCirros(vmi)).To(Succeed())
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "echo " + testString + "\n"},
		&expect.BExp{R: testString},
	}, 240)).To(Succeed())

	By(fmt.Sprintf("Generating about %dMB of data", mb))
	// (128 bytes/line) * (8 * 1024 * N) = N MB
	// serial is expected to be at 115200 bps -> 1MB takes 73 seconds
	startn := fmt.Sprintf("%07d", 1)
	endn := fmt.Sprintf("%07d", 8*1024*mb)
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "for num in $(seq -w " + startn + " " + endn + "); do echo \"logline ${num} ${num} ${num} ${num} ${num} ${num} ${num} ${num} ${num} ${num} ${num} ${num} ${num} ${num} ${num}\"; done" + "\n"},
		&expect.BExp{R: "logline " + endn},
	}, 240)).To(Succeed())
}

func getConsoleLogs(virtlauncherPod *k8sv1.Pod) (string, error) {
	logsRaw, err := kubevirt.Client().CoreV1().
		Pods(virtlauncherPod.Namespace).
		GetLogs(virtlauncherPod.Name, &k8sv1.PodLogOptions{
			Container: "guest-console-log",
		}).
		DoRaw(context.Background())
	return strings.ReplaceAll(string(logsRaw), "\r\n", "\n"), err
}
