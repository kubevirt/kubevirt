package tests

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
)

type server string

const (
	TCPServer  = server("\"Hello World!\"\n")
	HTTPServer = server("\"HTTP/1.1 200 OK\\nContent-Length: 12\\n\\nHello World!\"\n")
)

func (s server) composeNetcatServerCommand(port int, extraArgs ...string) string {
	return fmt.Sprintf("screen -d -m sudo nc %s -klp %d -e echo -e %s", strings.Join(extraArgs, " "), port, string(s))
}

func StartTCPServer(vmi *v1.VirtualMachineInstance, port int) {
	libnet.WithIPv6(console.LoginToCirros)(vmi)
	TCPServer.Start(vmi, port)
}

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int) {
	libnet.WithIPv6(console.LoginToCirros)(vmi)
	HTTPServer.Start(vmi, port)
}

func StartHTTPServerWithSourceIp(vmi *v1.VirtualMachineInstance, port int, sourceIP string) {
	libnet.WithIPv6(console.LoginToCirros)(vmi)
	HTTPServer.Start(vmi, port, fmt.Sprintf("-s %s", sourceIP))
}

func StartPythonHttpServer(vmi *v1.VirtualMachineInstance, port int) {
	serverCommand := fmt.Sprintf("python3 -m http.server %d --bind ::0 &\n", port)
	Expect(console.RunCommand(vmi, serverCommand, 60*time.Second)).To(Succeed())
}

func (s server) Start(vmi *v1.VirtualMachineInstance, port int, extraArgs ...string) {
	Expect(console.RunCommand(vmi, s.composeNetcatServerCommand(port, extraArgs...), 60*time.Second)).To(Succeed())
}
