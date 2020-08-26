package tests

import (
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

func StartTCPServer(vmi *v1.VirtualMachineInstance, port int) {
	createServerString := fmt.Sprintf("screen -d -m nc -klp %d -e echo -e \"Hello World!\"\n", port)
	startServer(vmi, createServerString)
}

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int) {
	httpServerMaker := fmt.Sprintf("screen -d -m nc -klp %d -e echo -e \"HTTP/1.1 200 OK\\nContent-Length: 12\\n\\nHello World!\"\n", port)
	startServer(vmi, httpServerMaker)
}

func startServer(vmi *v1.VirtualMachineInstance, createServer string) {
	expecter, err := LoggedInCirrosExpecter(vmi)
	Expect(err).NotTo(HaveOccurred())
	prompt := "\\$ "
	defer expecter.Close()

	resp, err := ExpectBatchWithValidatedSend(expecter, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: createServer},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: RetValue("0")},
	}, 60*time.Second)
	log.DefaultLogger().Infof("%v", resp)
	Expect(err).ToNot(HaveOccurred())
}
