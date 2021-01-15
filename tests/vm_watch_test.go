package tests_test

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"time"

	v12 "kubevirt.io/client-go/api/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	// Define relevant k8s versions
	relevantk8sVer = "1.16.2"

	// Define a timeout for read opeartions in order to prevent test hanging
	readTimeout       = 30 * time.Second
	vmCreationTimeout = 1 * time.Minute

	vmAgeRegex = "^[0-9]+[sm]$"

	// Define a buffer size to read errors into
	bufferSize = 1024
)

// Reads up to buffSize characters from rc
func read(rc io.ReadCloser, buffSize int) (string, error) {
	buf := make([]byte, buffSize)

	n, err := rc.Read(buf)
	if err != nil {
		return "", err
	}

	if n > 0 {
		return string(buf[:n]), nil
	}

	return "", nil
}

// Reads from rc until a newline character is found
func readLineWithTimeout(rc io.ReadCloser, timeout time.Duration) (string, error) {
	lineChan := make(chan string)
	errChan := make(chan error)
	defer close(errChan)

	go func() {
		var line strings.Builder
		buf := make([]byte, 1)
		defer close(lineChan)

		for {
			n, err := rc.Read(buf)

			if err != nil && err != io.EOF && !strings.Contains(err.Error(), "file already closed") {
				errChan <- err
				return
			}

			if n > 0 {
				if buf[0] != '\n' {
					line.WriteByte(buf[0])
				} else {
					break
				}
			}
		}

		lineChan <- line.String()
	}()

	select {
	case line := <-lineChan:
		return line, nil
	case err := <-errChan:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout reached on read operation")
	}
}

// Reads VM/VMI status from rc and returns a new status.
// If oldStatus is non-nil, the function will read status lines until
// newStatus.running != oldStatus.running or newStatus.phase != oldStatus.phase
// in order to skip duplicated status lines
func readNewStatus(rc io.ReadCloser, oldStatus []string, timeout time.Duration) ([]string, error) {
	statusLine, err := readLineWithTimeout(rc, timeout)

	if err != nil {
		return nil, err
	}

	newStatus := strings.Fields(statusLine)

	// Skip status line with similar running state for VM or phase state for VMI
	// newStatus[2] and oldStatus[2] point to the VM running state or VMI phase
	if oldStatus != nil && len(oldStatus) == len(newStatus) {
		if len(oldStatus) == 2 {
			return readNewStatus(rc, newStatus, timeout)
		} else if len(oldStatus) >= 3 && newStatus[2] == oldStatus[2] {
			return readNewStatus(rc, newStatus, timeout)
		}
	}

	return newStatus, nil
}

// Create a command with output/error redirection.
// Returns (cmd, stdout, stderr)
func createCommandWithNSAndRedirect(namespace, cmdName string, args ...string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	_, cmd, err := tests.CreateCommandWithNS(namespace, cmdName, args...)

	if err != nil {
		return nil, nil, nil, err
	}

	// Output redirection
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	return cmd, stdOut, stdErr, nil
}

var _ = Describe("[rfe_id:3423][crit:high][vendor:cnv-qe@redhat.com][level:component]VmWatch", func() {
	var err error
	var virtCli kubecli.KubevirtClient

	var vm *v12.VirtualMachine

	// Reads an error from stderr and fails the test
	readFromStderr := func(stderr io.ReadCloser) {
		defer GinkgoRecover()
		msg, err := read(stderr, bufferSize)

		if err != nil {
			if err.Error() != "EOF" {
				Fail(fmt.Sprintf("Could not read from `kubectl` stderr: %v", err))
			}
		} else {
			Fail(fmt.Sprintf("Error from stderr: %s", msg))
		}
	}

	BeforeEach(func() {
		virtCli, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.SkipIfVersionBelow("Printing format for `kubectl get -w` on custom resources is only relevant for 1.16.2+", relevantk8sVer)
		tests.BeforeTestCleanup()

		vm = tests.NewRandomVMWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
		vm, err = virtCli.VirtualMachine(vm.ObjectMeta.Namespace).Create(vm)
		tests.PanicOnError(err)

		By("Making sure kubectl cache is updated to contain vm/vmi resources")
		Eventually(func() bool {
			_, getVM, err := tests.CreateCommandWithNS(tests.NamespaceTestDefault, tests.GetK8sCmdClient(), "get", "vm")
			tests.PanicOnError(err)
			_, getVMI, err := tests.CreateCommandWithNS(tests.NamespaceTestDefault, tests.GetK8sCmdClient(), "get", "vmi")
			tests.PanicOnError(err)

			return getVM.Run() == nil && getVMI.Run() == nil
		}, vmCreationTimeout, 1*time.Millisecond).Should(BeTrue())
	})

	AfterEach(func() {
		err := virtCli.VirtualMachine(tests.NamespaceTestDefault).Delete(vm.Name, &v1.DeleteOptions{})
		tests.PanicOnError(err)
	})

	It("[test_id:3468]Should update vm status with the proper columns using 'kubectl get vm -w'", func() {
		By("Waiting for a VM to be created")
		Eventually(func() bool {
			_, err := virtCli.VirtualMachine(tests.NamespaceTestDefault).Get(vm.Name, &v1.GetOptions{})
			return err == nil
		}, vmCreationTimeout, 1*time.Millisecond).Should(BeTrue())

		By("Setting up the kubectl command")
		cmd, stdout, stderr, err :=
			createCommandWithNSAndRedirect(vm.ObjectMeta.Namespace, tests.GetK8sCmdClient(), "get", "vm", "-w")
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).ToNot(BeNil())

		err = cmd.Start()
		Expect(err).ToNot(HaveOccurred(), "Command should have started successfully")

		defer cmd.Process.Kill()
		defer cmd.Process.Release()
		go readFromStderr(stderr)

		// Read column titles
		titles, err := readNewStatus(stdout, nil, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(Equal([]string{"NAME", "AGE", "VOLUME"}),
			"Output should have the proper columns")
	})

	It("[test_id:3466]Should update vmi status with the proper columns using 'kubectl get vmi -w'", func() {
		By("Waiting for a VM to be created")
		Eventually(func() bool {
			_, err := virtCli.VirtualMachine(tests.NamespaceTestDefault).Get(vm.Name, &v1.GetOptions{})
			return err == nil
		}, vmCreationTimeout, 1*time.Second).Should(BeTrue())

		By("Creating a running VMI to avoid empty output")
		guardVmi := tests.NewRandomVMI()
		guardVmi = tests.RunVMIAndExpectLaunch(guardVmi, 60)

		By("Setting up the kubectl command")
		cmd, stdout, stderr, err :=
			createCommandWithNSAndRedirect(vm.ObjectMeta.Namespace, tests.GetK8sCmdClient(), "get", "vmi", "-w")
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).ToNot(BeNil())

		err = cmd.Start()
		Expect(err).ToNot(HaveOccurred(), "Command should have stared successfully")

		defer cmd.Process.Kill()
		defer cmd.Process.Release()
		go readFromStderr(stderr)

		// Read the column titles
		titles, err := readNewStatus(stdout, nil, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(Equal([]string{"NAME", "AGE", "PHASE", "IP", "NODENAME"}),
			"Output should have the proper columns")

		// Read out the guard VMI
		vmiStatus, err := readNewStatus(stdout, titles, readTimeout)
		Expect(err).ToNot(HaveOccurred())

		// Start a VMI
		vm = tests.StartVirtualMachine(vm)
		var vmi *v12.VirtualMachineInstance

		By("Waiting for the VMI to be created")
		Eventually(func() bool {
			list, err := virtCli.VirtualMachineInstance(tests.NamespaceTestDefault).List(&v1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			if len(list.Items) >= 2 {
				for i := 0; i < len(list.Items); i++ {
					if list.Items[i].Name != guardVmi.Name {
						vmi = &list.Items[i]
						return true
					}
				}
			}

			return false
		}, vmCreationTimeout, 1*time.Second).Should(BeTrue())

		// There might be a second (or more?) guardVmi "running" line in the pipeline... Squashing it (/them) first
		for vmiStatus[0] == guardVmi.Name {
			vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex)),
			"VMI should not have a specified phase yet")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex), string(v12.Pending)),
			"VMI should be in the Pending phase")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex), string(v12.Scheduling)),
			"VMI should be in the Scheduling phase")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		// "scheduled" lines may or may not contain an IP, parsing only the first 3 fields
		Expect(len(vmiStatus)).To(BeNumerically(">=", 3), fmt.Sprintf("vmiStatus is missing expected properties %v", vmiStatus))
		Expect(vmiStatus[0]).To(Equal(vmi.Name))
		Expect(vmiStatus[1]).To(MatchRegexp(vmAgeRegex))
		Expect(vmiStatus[2]).To(Equal(string(v12.Scheduled)), "VMI should be in the Scheduled phase")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(vmiStatus)).To(Equal(5), fmt.Sprintf("vmiStatus is missing expected properties %v", vmiStatus))
		Expect(net.ParseIP(vmiStatus[3])).ToNot(BeNil())
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex), string(v12.Running), vmiStatus[3], vmi.Status.NodeName),
			"VMI should be in the Running phase")

		// Restart the VMI
		err = virtCli.VirtualMachine(vm.ObjectMeta.Namespace).Restart(vm.ObjectMeta.Name)
		Expect(err).ToNot(HaveOccurred(), "VMI should have been restarted")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(vmiStatus)).To(Equal(5), fmt.Sprintf("vmiStatus is missing expected properties %v", vmiStatus))
		Expect(net.ParseIP(vmiStatus[3])).ToNot(BeNil())
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex), string(v12.Failed), vmiStatus[3], vmi.Status.NodeName),
			"VMI should be in the Failed phase")

		By("Waiting for the second VMI to be created")
		Eventually(func() bool {
			list, err := virtCli.VirtualMachineInstance(tests.NamespaceTestDefault).List(&v1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			if len(list.Items) >= 2 {
				for i := 0; i < len(list.Items); i++ {
					if list.Items[i].Name != guardVmi.Name &&
						list.Items[i].UID != vmi.UID &&
						list.Items[i].Status.NodeName != "" {
						vmi = &list.Items[i]
						return true
					}
				}
			}

			return false
		}, vmCreationTimeout, 1*time.Second).Should(BeTrue())

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex)),
			"VMI should not have a specified phase yet")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex), string(v12.Pending)),
			"VMI should be in the Pending phase")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex), string(v12.Scheduling)),
			"VMI should be in the Scheduling phase")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(vmiStatus)).To(BeNumerically(">=", 3), fmt.Sprintf("vmiStatus is missing expected properties %v", vmiStatus))
		Expect(vmiStatus[0]).To(Equal(vmi.Name))
		Expect(vmiStatus[1]).To(MatchRegexp(vmAgeRegex))
		Expect(vmiStatus[2]).To(Equal(string(v12.Scheduled)), "VMI should be in the Scheduled phase")

		vmiStatus, err = readNewStatus(stdout, vmiStatus, readTimeout)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(vmiStatus)).To(Equal(5), fmt.Sprintf("vmiStatus is missing expected propertiesL %v", vmiStatus))
		Expect(net.ParseIP(vmiStatus[3])).ToNot(BeNil())
		Expect(vmiStatus).To(ConsistOf(vmi.Name, MatchRegexp(vmAgeRegex), string(v12.Running), vmiStatus[3], vmi.Status.NodeName),
			"VMI should be in the Running phase")
	})
})
