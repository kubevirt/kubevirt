package guestlog

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const cirrosStartupTimeout = 60
const testString = "GuestConsoleTest3413254123535234523"

var _ = Describe("[sig-compute]Guest console log", decorators.SigCompute, func() {

	var (
		virtClient kubecli.KubevirtClient
		alpineVmi  *v1.VirtualMachineInstance
		cirrosVmi  *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		alpineVmi = libvmi.NewAlpine()
		alpineVmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(true)
		alpineVmi.Spec.Domain.Devices.LogSerialConsole = pointer.P(true)

		cirrosVmi = libvmi.NewCirros()
		cirrosVmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(true)
		cirrosVmi.Spec.Domain.Devices.LogSerialConsole = pointer.P(true)
	})

	Describe("[level:component] Guest console log container", func() {
		Context("set LogSerialConsole", func() {
			DescribeTable("should successfully start with LogSerialConsole", func(autoattachSerialConsole, logSerialConsole, expected bool) {
				By("Starting a VMI")
				alpineVmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(autoattachSerialConsole)
				alpineVmi.Spec.Domain.Devices.LogSerialConsole = pointer.P(logSerialConsole)
				vmi := tests.RunVMIAndExpectLaunch(alpineVmi, cirrosStartupTimeout)

				By("Finding virt-launcher pod")
				virtlauncherPod, err := libvmi.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())
				if expected {
					Expect(virtlauncherPod.Spec.Containers).To(HaveLen(3))
				} else {
					Expect(virtlauncherPod.Spec.Containers).To(HaveLen(2))
				}
				foundContainer := false
				for _, container := range virtlauncherPod.Spec.Containers {
					if container.Name == "guest-console-log" {
						foundContainer = true
					}
				}
				Expect(foundContainer).To(Equal(expected))

				if expected {
					for _, containerStatus := range virtlauncherPod.Status.ContainerStatuses {
						if containerStatus.Name == "guest-console-log" {
							Expect(containerStatus.State.Running).ToNot(BeNil())
						}
					}
				}
			},
				Entry("with AutoattachSerialConsole and LogSerialConsole", true, true, true),
				Entry("with AutoattachSerialConsole but not LogSerialConsole", true, false, false),
				Entry("without AutoattachSerialConsole but with LogSerialConsole", false, true, false),
				Entry("without AutoattachSerialConsole and without LogSerialConsole", false, false, false),
			)
		})

		Context("fetch logs", func() {
			var cirrosLogo = strings.Replace(` ____               ____  ____
 / __/ __ ____ ____ / __ \/ __/
/ /__ / // __// __// /_/ /\ \ 
\___//_//_/  /_/   \____/___/ 
   http://cirros-cloud.net
`, "\n", "\r\n", 5)

			It("it should fetch logs for a running VM with logs API", func() {
				vmi := tests.RunVMIAndExpectLaunch(cirrosVmi, cirrosStartupTimeout)

				By("Finding virt-launcher pod")
				virtlauncherPod, err := libvmi.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				By("Getting logs with logs API and ensure the logs are correctly ordered with no unexpected line breaks")

				Eventually(func(g Gomega) bool {
					logs, err := getConsoleLogs(virtClient, virtlauncherPod)
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
				expecter, errChan, eerr := console.NewExpecter(virtClient, vmi, 90*time.Second)
				Expect(eerr).ToNot(HaveOccurred())
				if eerr == nil {
					defer func() {
						derr := expecter.Close()
						Expect(derr).ToNot(HaveOccurred())
					}()
				}

				Consistently(errChan).ShouldNot(Receive())

				logs, err := getConsoleLogs(virtClient, virtlauncherPod)
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring that logs contain the login attempt")
				Expect(logs).To(ContainSubstring(vmi.Name + " login: cirros"))

				// TODO: console.LoginToCirros is not systematically waiting for `\u001b[8m` to prevent echoing the password, fix it first
				// By("Ensuring that logs don't contain the login password")
				// Expect(outputString).ToNot(ContainSubstring("Password: gocubsgo"))

				By("Ensuring that logs contain the test command and its output")
				Expect(logs).To(ContainSubstring("echo " + testString + "\r\n"))
				Expect(logs).To(ContainSubstring("\r\n" + testString + "\r\n"))
			})

			It("it should rotate the internal log files", func() {
				vmi := tests.RunVMIAndExpectLaunch(cirrosVmi, cirrosStartupTimeout)

				By("Finding virt-launcher pod")
				virtlauncherPod, err := libvmi.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				By("Generating 9MB of log data to force log rotation and discarding")
				generateHugeLogData(vmi, 9)

				By("Ensuring that log fetching is not failing")
				_, err = getConsoleLogs(virtClient, virtlauncherPod)
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring that we have 4 rotated log files (+term one)")
				outputString, err := exec.ExecuteCommandOnPod(virtClient, virtlauncherPod, "guest-console-log", []string{"/bin/ls", "-l", fmt.Sprintf("/var/run/kubevirt-private/%v", vmi.UID)})
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Count(outputString, "virt-serial0-log")).To(Equal(4 + 1))
			})

			It("it should not skip any log line even trying to flood the serial console for QOSGuaranteed VMs", func() {
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
				vmi := tests.RunVMIAndExpectLaunch(cirrosVmi, cirrosStartupTimeout)
				Expect(vmi.Status.QOSClass).ToNot(BeNil())
				Expect(*vmi.Status.QOSClass).To(Equal(k8sv1.PodQOSGuaranteed))

				By("Finding virt-launcher pod")
				virtlauncherPod, err := libvmi.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				By("Generating 1MB of log data")
				generateHugeLogData(vmi, 1)

				By("Ensuring that log fetching is not failing")
				logs, err := getConsoleLogs(virtClient, virtlauncherPod)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that log lines are sequential with no gaps")
				outputLines := strings.Split(logs, "\r\n")
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

func getConsoleLogs(virtClient kubecli.KubevirtClient, virtlauncherPod *k8sv1.Pod) (string, error) {
	logsRaw, err := virtClient.CoreV1().
		Pods(virtlauncherPod.Namespace).
		GetLogs(virtlauncherPod.Name, &k8sv1.PodLogOptions{
			Container: "guest-console-log",
		}).
		DoRaw(context.Background())
	return string(logsRaw), err
}
