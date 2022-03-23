package tests

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/tests/console"
)

type server string

const (
	TCPServer  = server("\"Hello World!\"&\n")
	HTTPServer = server("\"HTTP/1.1 200 OK\\nContent-Length: 12\\n\\nHello World!\"&\n")
)

func (s server) composeNetcatServerCommand(port int, extraArgs ...string) string {
	return fmt.Sprintf("nc %s -klp %d -e echo -e %s", strings.Join(extraArgs, " "), port, string(s))
}

func StartTCPServer(vmi *v1.VirtualMachineInstance, port int, loginTo console.LoginToFunction) {
	loginTo(vmi)
	TCPServer.Start(vmi, port)
}

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int, loginTo console.LoginToFunction) {
	loginTo(vmi)
	HTTPServer.Start(vmi, port)
}

func StartHTTPServerWithSourceIp(vmi *v1.VirtualMachineInstance, port int, sourceIP string, loginTo console.LoginToFunction) {
	loginTo(vmi)
	HTTPServer.Start(vmi, port, fmt.Sprintf("-s %s", sourceIP))
}

func StartPythonHttpServer(vmi *v1.VirtualMachineInstance, port int) {
	serverCommand := fmt.Sprintf("python3 -m http.server %d --bind ::0 &\n", port)
	Expect(console.RunCommand(vmi, serverCommand, 60*time.Second)).To(Succeed())
}

func (s server) Start(vmi *v1.VirtualMachineInstance, port int, extraArgs ...string) {
	Expect(console.RunCommand(vmi, s.composeNetcatServerCommand(port, extraArgs...), 60*time.Second)).To(Succeed())
}
