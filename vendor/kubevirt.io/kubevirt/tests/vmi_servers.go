package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
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
	LoginToCirros(vmi)
	TCPServer.Start(vmi, port)
}

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int) {
	LoginToCirros(vmi)
	HTTPServer.Start(vmi, port)
}

func (s server) Start(vmi *v1.VirtualMachineInstance, port int) {
	Expect(VmiConsoleRunCommand(vmi, s.composeNetcatServerCommand(port), 60*time.Second)).To(Succeed())
}
