package libvmi

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"

	netutils "k8s.io/utils/net"

	expect "github.com/google/goexpect"
	"google.golang.org/grpc/codes"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
)

const (
	PromptExpression = `(\$ |\# )`
	CRLF             = "\r\n"
)

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewConsoleExpecter(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, timeout time.Duration, opts ...expect.Option) (expect.Expecter, <-chan error, error) {
	vmiReader, vmiWriter := io.Pipe()
	expecterReader, expecterWriter := io.Pipe()
	resCh := make(chan error)

	startTime := time.Now()
	con, err := virtCli.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: timeout})
	if err != nil {
		return nil, nil, err
	}
	timeout = timeout - time.Now().Sub(startTime)

	go func() {
		resCh <- con.Stream(kubecli.StreamOptions{
			In:  vmiReader,
			Out: expecterWriter,
		})
	}()

	opts = append(opts, expect.SendTimeout(timeout))
	opts = append(opts, expect.Verbose(true))
	opts = append(opts, expect.VerboseWriter(GinkgoWriter))
	return expect.SpawnGeneric(&expect.GenOptions{
		In:  vmiWriter,
		Out: expecterReader,
		Wait: func() error {
			return <-resCh
		},
		Close: func() error {
			expecterWriter.Close()
			vmiReader.Close()
			return nil
		},
		Check: func() bool { return true },
	}, timeout, opts...)
}

// ExpectBatchWithValidatedSend adds the expect.BSnd command to the exect.BExp expression.
// It is done to make sure the match was found in the result of the expect.BSnd
// command and not in a leftover that wasn't removed from the buffer.
// NOTE: the method doesn't support multiline commands in the sent value.
func ExpectBatchWithValidatedSend(expecter expect.Expecter, batch []expect.Batcher, timeout time.Duration) ([]expect.BatchRes, error) {
	sendFlag := false
	expectFlag := false
	previousSend := ""
	for i, batcher := range batch {
		switch batcher.Cmd() {
		case expect.BatchExpect:
			if expectFlag == true {
				return nil, fmt.Errorf("Two sequential expect.BExp are not allowed")
			}
			expectFlag = true
			sendFlag = false
			if _, ok := batch[i].(*expect.BExp); !ok {
				return nil, fmt.Errorf("ExpectBatchWithValidatedSend support only expect of type BExp")
			}
			bExp, _ := batch[i].(*expect.BExp)
			previousSend := regexp.QuoteMeta(previousSend)

			// Remove the \n since it is translated by the console to \r\n.
			previousSend = strings.TrimSuffix(previousSend, "\n")
			bExp.R = fmt.Sprintf("%s%s%s", previousSend, "((?s).*)", bExp.R)
			previousSend = ""
		case expect.BatchSend:
			if sendFlag == true {
				return nil, fmt.Errorf("Two sequential expect.BSend are not allowed")
			}
			sendFlag = true
			expectFlag = false
			previousSend = batcher.Arg()
		case expect.BatchSwitchCase:
			return nil, fmt.Errorf("ExpectBatchWithValidatedSend doesn't support BatchSwitchCase")
		default:
			return nil, fmt.Errorf("Unkown command: ExpectBatchWithValidatedSend supports only BatchExpect and BatchSend")
		}
	}

	res, err := expecter.ExpectBatch(batch, timeout)
	return res, err
}

func CheckForTextExpecter(vmi *v1.VirtualMachineInstance, expected []expect.Batcher, wait int) error {
	virtClient, err := kubecli.GetKubevirtClient()
	panicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 30*time.Second)
	if err != nil {
		return err
	}
	defer expecter.Close()

	resp, err := ExpectBatchWithValidatedSend(expecter, expected, time.Second*time.Duration(wait))
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("%v", resp)
	}
	return err
}

func configureConsole(expecter expect.Expecter, prompt string, shouldSudo bool) error {
	sudoString := ""
	if shouldSudo {
		sudoString = "sudo "
	}
	batch := append([]expect.Batcher{
		&expect.BSnd{S: "stty cols 500 rows 500\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: RetValue("0")},
		&expect.BSnd{S: fmt.Sprintf("%sdmesg -n 1\n", sudoString)},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: RetValue("0")}})
	resp, err := expecter.ExpectBatch(batch, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Infof("%v", resp)
	}
	return err
}

func configureIPv6OnVMI(vmi *v1.VirtualMachineInstance, expecter expect.Expecter, virtClient kubecli.KubevirtClient, prompt string) error {
	hasEth0Iface := func() bool {
		hasNetEth0Batch := append([]expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: prompt},
			&expect.BSnd{S: "ip a | grep -q eth0; echo $?\n"},
			&expect.BExp{R: RetValue("0")}})
		_, err := ExpectBatchWithValidatedSend(expecter, hasNetEth0Batch, 30*time.Second)
		return err == nil
	}

	hasGlobalIPv6 := func() bool {
		hasGlobalIPv6Batch := append([]expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: prompt},
			&expect.BSnd{S: "ip -6 address show dev eth0 scope global | grep -q inet6; echo $?\n"},
			&expect.BExp{R: RetValue("0")}})
		_, err := ExpectBatchWithValidatedSend(expecter, hasGlobalIPv6Batch, 30*time.Second)
		return err == nil
	}

	clusterSupportsIpv6 := func() bool {
		pod := GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		for _, ip := range pod.Status.PodIPs {
			if netutils.IsIPv6String(ip.IP) {
				return true
			}
		}
		return false
	}

	if !clusterSupportsIpv6() ||
		(vmi.Spec.Domain.Devices.Interfaces == nil || len(vmi.Spec.Domain.Devices.Interfaces) == 0 || vmi.Spec.Domain.Devices.Interfaces[0].InterfaceBindingMethod.Masquerade == nil) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface != nil && !*vmi.Spec.Domain.Devices.AutoattachPodInterface) ||
		(!hasEth0Iface() || hasGlobalIPv6()) {
		return nil
	}

	addIPv6Address := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "sudo ip -6 addr add fd10:0:2::2/120 dev eth0; echo $?\n"},
		&expect.BExp{R: RetValue("0")}})
	resp, err := ExpectBatchWithValidatedSend(expecter, addIPv6Address, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6Address failed: %v", resp)
		expecter.Close()
		return err
	}

	time.Sleep(5 * time.Second)
	addIPv6DefaultRoute := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "sudo ip -6 route add default via fd10:0:2::1 src fd10:0:2::2; echo $?\n"},
		&expect.BExp{R: RetValue("0")}})
	resp, err = ExpectBatchWithValidatedSend(expecter, addIPv6DefaultRoute, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6DefaultRoute failed: %v", resp)
		expecter.Close()
		return err
	}

	return nil
}

func LoggedInCirrosExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	panicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	hostName := dns.SanitizeHostname(vmi)

	// Do not login, if we already logged in
	err = expecter.Send("\n")
	if err != nil {
		expecter.Close()
		return nil, err
	}
	_, _, err = expecter.Expect(regexp.MustCompile(`\$`), 10*time.Second)
	if err == nil {
		return expecter, nil
	}

	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "login as 'cirros' user. default password: 'gocubsgo'. use 'sudo' for root."},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: hostName + " login:"},
		&expect.BSnd{S: "cirros\n"},
		&expect.BExp{R: "Password:"},
		&expect.BSnd{S: "gocubsgo\n"},
		&expect.BExp{R: "\\$"}})
	resp, err := expecter.ExpectBatch(b, 180*time.Second)

	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %v", resp)
		expecter.Close()
		return nil, err
	}

	err = configureConsole(expecter, "\\$ ", true)
	if err != nil {
		expecter.Close()
		return nil, err
	}

	return expecter, configureIPv6OnVMI(vmi, expecter, virtClient, "\\$ ")
}

func LoggedInAlpineExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	panicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}

	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "localhost login:"},
		&expect.BSnd{S: "root\n"},
		&expect.BExp{R: "localhost:~\\#"}})
	res, err := expecter.ExpectBatch(b, 180*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %v", res)
		expecter.Close()
		return nil, err
	}

	err = configureConsole(expecter, "localhost:~\\#", false)
	if err != nil {
		expecter.Close()
		return nil, err
	}
	return expecter, err
}

// LoggedInFedoraExpecter return prepared and ready to use console expecter for
// Fedora test VM
func LoggedInFedoraExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	panicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BCas{C: []expect.Caser{
			&expect.Case{
				// In case the VM's did not get hostname form DHCP server try the default hostname
				R:  regexp.MustCompile(`localhost login: `),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				// Using only "login: " would match things like "Last failed login: Tue Jun  9 22:25:30 UTC 2020 on ttyS0"
				R:  regexp.MustCompile(vmi.Name + ` login: `),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Password:`),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Login incorrect`),
				T:  expect.LogContinue("Failed to log in", expect.NewStatus(codes.PermissionDenied, "login failed")),
				Rt: 10,
			},
			&expect.Case{
				R: regexp.MustCompile(`\$ `),
				T: expect.OK(),
			},
		}},
		&expect.BSnd{S: "sudo su\n"},
		&expect.BExp{R: "\\#"},
	})
	res, err := expecter.ExpectBatch(b, 3*time.Minute)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %+v", res)
		expecter.Close()
		return expecter, err
	}

	err = configureConsole(expecter, "\\#", false)
	if err != nil {
		expecter.Close()
		return nil, err
	}

	return expecter, configureIPv6OnVMI(vmi, expecter, virtClient, "\\#")
}

// ReLoggedInFedoraExpecter return prepared and ready to use console expecter for
// Fedora test VM, when you are reconnecting (no login needed)
func ReLoggedInFedoraExpecter(vmi *v1.VirtualMachineInstance, timeout int) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	panicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "#"}})
	res, err := expecter.ExpectBatch(b, time.Duration(timeout)*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %+v", res)
		expecter.Close()
		return expecter, err
	}
	return expecter, err
}

func SecureBootExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	panicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	b := append([]expect.Batcher{
		&expect.BExp{R: "secureboot: Secure boot enabled"},
	})
	res, err := expecter.ExpectBatch(b, 180*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %+v", res)
		expecter.Close()
		return expecter, err
	}

	return expecter, err
}

type VMIExpecterFactory func(*v1.VirtualMachineInstance) (expect.Expecter, error)

func RetValue(retcode string) string {
	return "\n" + retcode + CRLF + ".*" + PromptExpression
}
