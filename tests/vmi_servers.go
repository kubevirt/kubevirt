package tests

import (
	"fmt"
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

func (s server) composeNetcatServerCommand(port int) string {
	return fmt.Sprintf("screen -d -m sudo nc -klp %d -e echo -e %s", port, string(s))
}

func StartTCPServer(vmi *v1.VirtualMachineInstance, port int) {
	libnet.WithIPv6(console.LoginToCirros)(vmi)
	TCPServer.Start(vmi, port)
}

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int) {
	libnet.WithIPv6(console.LoginToCirros)(vmi)
	HTTPServer.Start(vmi, port)
}

func StartPythonHttpServer(vmi *v1.VirtualMachineInstance, port int) {
	serverCommand := fmt.Sprintf("python3 -m http.server %d --bind ::0 &\n", port)
	Expect(console.RunCommand(vmi, serverCommand, 60*time.Second)).To(Succeed())
}

func (s server) Start(vmi *v1.VirtualMachineInstance, port int) {
	Expect(console.RunCommand(vmi, s.composeNetcatServerCommand(port), 60*time.Second)).To(Succeed())
}
