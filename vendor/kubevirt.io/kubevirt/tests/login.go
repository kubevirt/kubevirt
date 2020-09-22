package tests

import (
	"fmt"
	"regexp"
	"time"

	v1 "kubevirt.io/client-go/api/v1"

	expect "github.com/google/goexpect"
	"google.golang.org/grpc/codes"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
)

// LoginToCirros call LoggedInFedoraExpecter but does not return the expecter
func LoginToCirros(vmi *v1.VirtualMachineInstance) error {
	expecter, err := LoggedInCirrosExpecter(vmi)
	defer expecter.Close()
	return err
}

// LoggedInCirrosExpecter return prepared and ready to use console expecter for
// Alpine test VM
func LoggedInCirrosExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
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

// LoggedInAlpineExpecter return prepared and ready to use console expecter for
// Alpine test VM
func LoggedInAlpineExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
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
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BCas{C: []expect.Caser{
			&expect.Case{
				// Using only "login: " would match things like "Last failed login: Tue Jun  9 22:25:30 UTC 2020 on ttyS0"
				// and in case the VM's did not get hostname form DHCP server try the default hostname
				R:  regexp.MustCompile(fmt.Sprintf(`(localhost|%s) login: `, vmi.Name)),
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
	PanicOnError(err)
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
