package portforward

import (
	"errors"
	"net"
	"strings"

	"github.com/golang/glog"

	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
)

type portForwarder struct {
	kind, namespace, name string
	resource              portforwardableResource
}

type portforwardableResource interface {
	PortForward(name string, port int, protocol string) (kvcorev1.StreamInterface, error)
}

func (p *portForwarder) startForwarding(address *net.IPAddr, port forwardedPort) error {
	glog.Infof("forwarding %s %s:%d to %d", port.protocol, address, port.local, port.remote)
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
		glog.Errorf("error handling connection for %d: %v", port.local, err)
	}
}
