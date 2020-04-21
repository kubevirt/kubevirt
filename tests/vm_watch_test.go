package tests_test

import (
	"io"
	"os/exec"
	"strings"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	// Define relevant k8s versions
	relevantk8sVer = "1.16.2"

	// Define a timeout for read opeartions in order to prevent test hanging
	readTimeout     = 1 * time.Minute
	processWaitTime = 2 * time.Minute

	bufferSize = 1024
)

var _ = Describe("[rfe_id:3423][crit:high][vendor:cnv-qe@redhat.com][level:component]VmWatch", func() {
	tests.FlagParse()

	virtCli, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	type vmStatus struct {
		name,
		age,
		running,
		volume string
	}

	type vmiStatus struct {
		name,
		age,
		phase,
		ip,
		node string
	}

	newVMStatus := func(fields []string) *vmStatus {
		flen := len(fields)
		stat := &vmStatus{}

		switch {
		case flen > 3:
			stat.volume = fields[3]
			fallthrough
		case flen > 2:
			stat.running = fields[2]
			fallthrough
		case flen > 1:
			stat.age = fields[1]
			fallthrough
		case flen > 0:
			stat.name = fields[0]
		}

		return stat
	}

	newVMIStatus := func(fields []string) *vmiStatus {
		flen := len(fields)
		stat := &vmiStatus{}

		switch {
		case flen > 4:
			stat.node = fields[4]
			fallthrough
		case flen > 3:
			stat.ip = fields[3]
			fallthrough
		case flen > 2:
			stat.phase = fields[2]
			fallthrough
		case flen > 1:
			stat.age = fields[1]
			fallthrough
		case flen > 0:
			stat.name = fields[0]
		}

		return stat
	}

	// Fail the test if stderr has something to read
	failOnError := func(rc io.ReadCloser) {
		defer GinkgoRecover()

		buf := make([]byte, bufferSize)

		n, err := rc.Read(buf)

		if err != nil && n > 0 {
			rc.Close()
			Fail(string(buf[:n]))
		}
	}

	// Reads from stdin until a newline character is found
	readLine := func(rc io.ReadCloser, timeout time.Duration) string {
		lineChan := make(chan string)

		go func() {
			defer GinkgoRecover()

			var line strings.Builder
			buf := make([]byte, 1)
			defer close(lineChan)

			for {
				n, err := rc.Read(buf)

				if err != nil && err != io.EOF && !strings.Contains(err.Error(), "file already closed") {
					Fail(err.Error())
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
			return line
		case <-time.After(timeout):
			err := rc.Close()
			Expect(err).ToNot(HaveOccurred(), "stdout should have been closed properly")

			Fail("Timeout reached on read operation")

			return ""
		}
	}

	// Reads VM status from the given pipe (stdin in this case) and
	// returns a new status.
	// if old_status is non-nil, the function will read status lines until
	// new_status.running != old_status.running in order to skip duplicated status lines
	readVMStatus := func(rc io.ReadCloser, oldStatus *vmStatus, timeout time.Duration) *vmStatus {
		newStat := newVMStatus(strings.Fields(readLine(rc, timeout)))

		for oldStatus != nil && newStat.running == oldStatus.running {
			newStat = newVMStatus(strings.Fields(readLine(rc, timeout)))
		}

		return newStat
	}

	// Reads VMI status from the given pipe (stdin in this case) and
	// returns a new status.
	// if old_status is non-nil, the function will read status lines until
	// new_status.phase != old_status.phase in order to skip duplicated lines
	readVMIStatus := func(rc io.ReadCloser, oldStatus *vmiStatus, timeout time.Duration) *vmiStatus {
		newStat := newVMIStatus(strings.Fields(readLine(rc, timeout)))

		for oldStatus != nil && newStat.phase == oldStatus.phase {
			newStat = newVMIStatus(strings.Fields(readLine(rc, timeout)))
		}

		return newStat
	}

	// Create a command with output/error redirection.
	// Returns (cmd, stdout, stderr)
	createCommandWithNSAndRedirect := func(namespace, cmdName string, args ...string) (*exec.Cmd, io.ReadCloser, io.ReadCloser) {
		cmdName, cmd, err := tests.CreateCommandWithNS(namespace, cmdName, args...)

		Expect(cmdName).ToNot(Equal(""))
		Expect(cmd).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred(), "Command should have been created with proper kubectl/oc arguments")

		// Output redirection
		stdOut, err := cmd.StdoutPipe()
		Expect(err).ToNot(HaveOccurred(), "stdout should have been redirected")
		Expect(stdOut).ToNot(BeNil())

		stdErr, err := cmd.StderrPipe()
		Expect(err).ToNot(HaveOccurred(), "stderr should have been redirected")
		Expect(stdErr).ToNot(BeNil())

		return cmd, stdOut, stdErr
	}

	BeforeEach(func() {
		tests.SkipIfVersionBelow("Printing format for `kubectl get -w` on custom resources is only relevant for 1.16.2+", relevantk8sVer)
		tests.BeforeTestCleanup()
	})

	PIt("[test_id:3468] [flaky] Should update vm status with the proper columns using 'kubectl get vm -w'", func() {
		By("Creating a new VM spec")
		vm := tests.NewRandomVMWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
		Expect(vm).ToNot(BeNil())

		By("Setting up the kubectl command")
		cmd, stdout, stderr :=
			createCommandWithNSAndRedirect(vm.ObjectMeta.Namespace, tests.GetK8sCmdClient(), "get", "vm", "-w")
		Expect(cmd).ToNot(BeNil())

		err = cmd.Start()
		Expect(err).ToNot(HaveOccurred(), "Command should have started successfully")

		defer cmd.Process.Kill()

		time.Sleep(processWaitTime)

		go failOnError(stderr)

		By("Applying the VM to the cluster")
		vm, err := virtCli.VirtualMachine(vm.ObjectMeta.Namespace).Create(vm)
		Expect(err).ToNot(HaveOccurred(), "VM should have been added to the cluster")

		// Read column titles
		vmStatus := readVMStatus(stdout, nil, readTimeout)
		Expect(vmStatus.name).To(Equal("NAME"), "Output should have the NAME column")
		Expect(vmStatus.age).To(Equal("AGE"), "Output should have the AGE column")
		Expect(vmStatus.running).To(Equal("RUNNING"), "Output should have the RUNNING column")
		Expect(vmStatus.volume).To(Equal("VOLUME"), "Output should have the VOLUME column")

		// Read first status of the vm
		vmStatus = readVMStatus(stdout, vmStatus, readTimeout)
		Expect(vmStatus.name).To(Equal(vm.Name))
		By("Expecting vm.running == false")
		Expect(vmStatus.running).To(Equal("false"))

		By("Starting the VM")
		vm = tests.StartVirtualMachine(vm)

		vmStatus = readVMStatus(stdout, vmStatus, readTimeout)
		By("Expecting vm.running == true")
		Expect(vmStatus.running).To(Equal("true"))

		By("Restarting the VM")
		err = virtCli.VirtualMachine(vm.ObjectMeta.Namespace).Restart(vm.ObjectMeta.Name)
		Expect(err).ToNot(HaveOccurred(), "VM should have been restarted")

		vmStatus = readVMStatus(stdout, nil, readTimeout)
		By("Expecting vm.running == true")
		Expect(vmStatus.running).To(Equal("true"))

		By("Stopping the VM")
		vm = tests.StopVirtualMachine(vm)

		vmStatus = readVMStatus(stdout, vmStatus, readTimeout)
		By("Expecting vm.running == false")
		Expect(vmStatus.running).To(Equal("false"))

		By("Deleting the VM")
		err = virtCli.VirtualMachine(vm.ObjectMeta.Namespace).Delete(vm.ObjectMeta.Name, &v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "VM should have been deleted from the cluster")
	})

	It("[test_id:3466]Should update vmi status with the proper columns using 'kubectl get vmi -w'", func() {
		By("Creating a random VMI spec")
		vm := tests.NewRandomVMWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))

		Expect(vm).ToNot(BeNil())

		By("Setting up the kubectl command")
		cmd, stdout, stderr :=
			createCommandWithNSAndRedirect(vm.ObjectMeta.Namespace, tests.GetK8sCmdClient(), "get", "vmi", "-w")
		Expect(cmd).ToNot(BeNil())

		err = cmd.Start()
		Expect(err).ToNot(HaveOccurred(), "Command should have stared successfully")

		defer cmd.Process.Kill()

		time.Sleep(processWaitTime)

		go failOnError(stderr)

		By("Applying vmi to the cluster")
		vm, err = virtCli.VirtualMachine(vm.ObjectMeta.Namespace).Create(vm)
		Expect(err).ToNot(HaveOccurred(), "VMI should have been added to the cluster")

		// Start a VMI
		vm = tests.StartVirtualMachine(vm)

		// Read the column titles
		vmiStatus := readVMIStatus(stdout, nil, readTimeout)
		Expect(vmiStatus.name).To(Equal("NAME"), "Output should have the NAME column")
		Expect(vmiStatus.age).To(Equal("AGE"), "Output should have the AGE column")
		Expect(vmiStatus.phase).To(Equal("PHASE"), "Output should have the PHASE column")
		Expect(vmiStatus.ip).To(Equal("IP"), "Output should have the IP column")
		Expect(vmiStatus.node).To(Equal("NODENAME"), "Output should have the NODENAME column")

		By("Expecting vmi.phase == ''")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal(""))

		By("Expecting vmi.phase == Pending")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Pending"))

		By("Expecting vmi.phase == Scheduling")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Scheduling"))

		By("Expecting vmi.phase == Scheduled")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Scheduled"))

		By("Expecting vmi.phase == Running")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Running"))

		// Restart the VMI
		err = virtCli.VirtualMachine(vm.ObjectMeta.Namespace).Restart(vm.ObjectMeta.Name)
		Expect(err).ToNot(HaveOccurred(), "VMI should have been restarted")

		By("Expecting vmi.phase == Failed")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Failed"))

		By("Expecting vmi.phase == ''")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal(""))

		By("Expecting vmi.phase == Pending")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Pending"))

		By("Expecting vmi.phase == Scheduling")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Scheduling"))

		By("Expecting vmi.phase == Scheduled")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Scheduled"))

		By("Expecting vmi.phase == Running")
		vmiStatus = readVMIStatus(stdout, vmiStatus, readTimeout)
		Expect(vmiStatus.phase).To(Equal("Running"))

		By("Deleting the VM")
		err = virtCli.VirtualMachine(vm.ObjectMeta.Namespace).Delete(vm.ObjectMeta.Name, &v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "VM should have been deleted from the cluster")
	})
})
