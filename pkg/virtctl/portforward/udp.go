/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package portforward

import (
	"net"
	"sync"

	"kubevirt.io/client-go/log"
)

const bufSize = 1500

func (p *portForwarder) startForwardingUDP(address *net.IPAddr, port forwardedPort) error {
	listener, err := net.ListenUDP(
		"udp",
		&net.UDPAddr{
			IP:   address.IP,
			Zone: address.Zone,
			Port: port.local,
		},
	)
	if err != nil {
		return err
	}

	proxy := udpProxy{
		listener: listener,
		remoteDialer: func() (net.Conn, error) {
			log.Log.Infof("opening new udp tunnel to %d", port.remote)
			stream, err := p.resource.PortForward(p.name, port.remote, port.protocol)
			if err != nil {
				log.Log.Errorf("can't access %s/%s.%s: %v", p.kind, p.name, p.namespace, err)
				return nil, err
			}
			return stream.AsConn(), nil
		},
		clients: make(map[string]*udpProxyConn),
	}

	go proxy.Run()
	return nil
}

type udpProxy struct {
	listener *net.UDPConn

	remoteDialer func() (net.Conn, error)

	sync.Mutex
	clients map[string]*udpProxyConn
}

func (p *udpProxy) Run() {
	buf := make([]byte, bufSize)
	for {
		if err := p.handleRead(buf); err != nil {
			log.Log.Errorf("%v", err)
		}
	}
}

func (p *udpProxy) handleRead(buf []byte) error {
	n, clientAddr, err := p.listener.ReadFromUDP(buf[0:])
	if err != nil {
		return err
	}
	clientID := clientAddr.String()

	p.Lock()
	defer p.Unlock()

	client, isKnownClient := p.clients[clientID]

	if !isKnownClient {
		remoteConn, err := p.remoteDialer()
		if err != nil {
			return err
		}
		client = &udpProxyConn{
			localConn:  p.listener,
			clientAddr: clientAddr,
			remoteConn: remoteConn,
			close:      make(chan struct{}),
		}
		p.clients[clientID] = client
		go client.handleRemoteReads()
		go p.cleanupClient(clientID, client)
	}

	_, err = client.remoteConn.Write(buf[0:n])
	return err
}

func (p *udpProxy) cleanupClient(clientID string, client *udpProxyConn) {
	<-client.close
	p.Lock()
	defer p.Unlock()
	delete(p.clients, clientID)
}

type udpProxyConn struct {
	localConn  *net.UDPConn
	clientAddr *net.UDPAddr
	remoteConn net.Conn

	close chan struct{}
}

func (c *udpProxyConn) handleRemoteReads() {
	defer close(c.close)
	buf := make([]byte, bufSize)
	for {
		if err := c.handleRemoteRead(buf); err != nil {
			log.Log.Errorf("closing client: %v\n", err)
			return
		}
	}
}

func (c *udpProxyConn) handleRemoteRead(buf []byte) error {
	n, err := c.remoteConn.Read(buf[0:])
	if err != nil {
		return err
	}
	_, err = c.localConn.WriteToUDP(buf[0:n], c.clientAddr)
	if err != nil {
		return err
	}
	return nil
}
