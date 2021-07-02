package portforward

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"kubevirt.io/client-go/kubecli"
)

type portForwarder struct {
	namespace, name string
	resource        portforwardableResource
}

type portforwardableResource interface {
	PortForward(name string, port int, protocol string) (kubecli.StreamInterface, error)
}

func (p *portForwarder) startForwarding(address *net.IPAddr, port forwardedPort) error {
	fmt.Printf("forwarding %s %s:%d -> %d\n", port.protocol, address, port.local, port.remote)
	if port.protocol == protocolUDP {
		return p.startForwardingUDP(address, port)
	}

	if port.protocol == protocolTCP {
		return p.startForwardingTCP(address, port)
	}

	return errors.New("unknown protocol: " + port.protocol)
}

func handleConnectionError(err error, port forwardedPort) {
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		fmt.Printf("error handling connection for %d: %v\n", port.local, err)
	}
}
