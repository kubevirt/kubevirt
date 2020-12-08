package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
)

type server string

const (
	TCPServer  = server("\"Hello World!\"")
	HTTPServer = server("\"HTTP/1.1 200 OK\\nContent-Length: 12\\n\\nHello World!\"")
)

func (s server) composeNetcatServerCommand(port int) string {
		return fmt.Sprintf("sudo nc -klp %d -e echo -e %s &\n", port, string(s))
}

func StartTCPServer(vmi *v1.VirtualMachineInstance, port int) {
	libnet.WithIPv6(console.LoginToCirros)(vmi)
	TCPServer.Start(vmi, port)
}

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int) {
	libnet.WithIPv6(console.LoginToCirros)(vmi)
	HTTPServer.Start(vmi, port)
}

func (s server) Start(vmi *v1.VirtualMachineInstance, port int) {
	Expect(console.RunCommand(vmi, s.composeNetcatServerCommand(port), 60*time.Second)).To(Succeed())
}
