package tests

import (
	"fmt"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

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
	ExpectWithOffset(1, loginTo(vmi)).To(Succeed())
	TCPServer.Start(vmi, port)
}

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int, loginTo console.LoginToFunction) {
	ExpectWithOffset(1, loginTo(vmi)).To(Succeed())
	HTTPServer.Start(vmi, port)
}

func StartHTTPServerWithSourceIp(vmi *v1.VirtualMachineInstance, port int, sourceIP string, loginTo console.LoginToFunction) {
	ExpectWithOffset(1, loginTo(vmi)).To(Succeed())
	HTTPServer.Start(vmi, port, fmt.Sprintf("-s %s", sourceIP))
}

func StartPythonHttpServer(vmi *v1.VirtualMachineInstance, port int) {
	Expect(console.RunCommand(vmi, "echo 'Hello World!' > index.html", 60*time.Second)).To(Succeed())

	serverCommand := fmt.Sprintf("python3 -m http.server %d --bind ::0 &\n", port)
	Expect(console.RunCommand(vmi, serverCommand, 60*time.Second)).To(Succeed())
}

func StartPythonUDPServer(vmi *v1.VirtualMachineInstance, port int, ipFamily k8sv1.IPFamily) {
	var inetSuffix string
	if ipFamily == k8sv1.IPv6Protocol {
		inetSuffix = "6"
	}

	serverCommand := fmt.Sprintf(`cat >udp_server.py <<EOL
import socket
datagramSocket = socket.socket(socket.AF_INET%s, socket.SOCK_DGRAM);
datagramSocket.bind(("", %d));
while(True):
    msg, srcAddress = datagramSocket.recvfrom(128);
    datagramSocket.sendto("Hello Client".encode(), srcAddress);
EOL`, inetSuffix, port)
	Expect(console.ExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: fmt.Sprintf("%s\n", serverCommand)},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: EchoLastReturnValue},
		&expect.BExp{R: console.ShellSuccess},
	}, 60*time.Second)).To(Succeed())

	serverCommand = "python3 udp_server.py &"
	Expect(console.RunCommand(vmi, serverCommand, 60*time.Second)).To(Succeed())
}

func (s server) Start(vmi *v1.VirtualMachineInstance, port int, extraArgs ...string) {
	Expect(console.RunCommand(vmi, s.composeNetcatServerCommand(port, extraArgs...), 60*time.Second)).To(Succeed())
}
