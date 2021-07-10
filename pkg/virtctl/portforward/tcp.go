package portforward

import (
	"io"
	"net"

	"github.com/golang/glog"
)

func (p *portForwarder) startForwardingTCP(address *net.IPAddr, port forwardedPort) error {
	listener, err := net.ListenTCP(
		port.protocol,
		&net.TCPAddr{
			IP:   address.IP,
			Zone: address.Zone,
			Port: port.local,
		})
	if err != nil {
		return err
	}

	go p.waitForConnection(listener, port)

	return nil
}

func (p *portForwarder) waitForConnection(listener net.Listener, port forwardedPort) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			glog.Errorln("error accepting connection:", err)
			return
		}
		glog.Infof("opening new tcp tunnel to %d", port.remote)
		stream, err := p.resource.PortForward(p.name, port.remote, port.protocol)
		if err != nil {
			glog.Errorf("can't access %s/%s.%s: %v", p.kind, p.name, p.namespace, err)
			return
		}
		go p.handleConnection(conn, stream.AsConn(), port)
	}
}

// handleConnection copies data between the local connection and the stream to
// the remote server.
func (p *portForwarder) handleConnection(local, remote net.Conn, port forwardedPort) {
	glog.Infof("handling tcp connection for %d", port.local)
	errs := make(chan error)
	go func() {
		_, err := io.Copy(remote, local)
		errs <- err
	}()
	go func() {
		_, err := io.Copy(local, remote)
		errs <- err
	}()

	handleConnectionError(<-errs, port)
	local.Close()
	remote.Close()
	handleConnectionError(<-errs, port)
}
