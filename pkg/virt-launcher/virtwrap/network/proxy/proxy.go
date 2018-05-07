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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"

	"k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/log"
)

type ProxyService struct {
	DestAddress string
	Ports       []v1.ContainerPort
}

func NewService(destAddress string, ports []v1.ContainerPort) *ProxyService {
	return &ProxyService{DestAddress: destAddress, Ports: ports}
}

func (l *ProxyService) Start() error {
	for _, port := range l.Ports {
		channel := getChannel(port.Protocol)
		go channel.Serve(l.DestAddress, port)
	}
	return nil
}

type Channel interface {
	Serve(address string, ports v1.ContainerPort)
}

func getChannel(protocol v1.Protocol) Channel {
	switch protocol {
	case v1.ProtocolTCP:
		return new(TCPChannel)
	case v1.ProtocolUDP:
		return new(UDPChannel)
	}
	return nil
}

type TCPChannel struct {
	ports v1.ContainerPort
	addr  string
}

type UDPChannel struct {
	TCPChannel
	connections map[string]*net.UDPAddr
	targetAddr  *net.UDPAddr
	targetConn  *net.UDPConn
	proxyConn   *net.UDPConn
	lock        sync.Mutex
}

func (l *TCPChannel) Serve(address string, ports v1.ContainerPort) {
	l.addr = address
	l.ports = ports

	for {
		incoming, err := net.Listen("tcp", fmt.Sprintf(":%d", l.ports.ContainerPort))
		if err != nil {
			log.Log.Reason(err).Errorf("failed to serve on %d", l.ports.ContainerPort)
			panic(err)
		}

		log.Log.Infof("serving on port: %d", l.ports.ContainerPort)

		client, err := incoming.Accept()
		if err != nil {
			log.Log.Reason(err).Errorf("failed to accept client connection on port %d", l.ports.ContainerPort)
		}
		defer client.Close()
		log.Log.Infof("connected to client %s", client.RemoteAddr().String())

		target, err := net.Dial("tcp", fmt.Sprintf("%s:%d", l.addr, l.ports.HostPort))
		if err != nil {
			log.Log.Reason(err).Errorf("failed to connect to target %s:%d", l.addr, l.ports.HostPort)
		}
		defer target.Close()
		log.Log.Infof("connected to target at ", target.RemoteAddr().String())

		// start copy threads
		go func() { io.Copy(target, client) }()
		go func() { io.Copy(client, target) }()
	}
}
func (l *UDPChannel) forwardTargetToClient(clientAddr *net.UDPAddr) {

	var buffer [1500]byte
	for {
		// Read from target
		n, err := l.targetConn.Read(buffer[0:])
		if err != nil {
			continue
		}
		// write to the client
		_, err = l.proxyConn.WriteToUDP(buffer[0:n], clientAddr)
		if err != nil {
			continue
		}
	}
}

func (l *UDPChannel) handleForwarding() {
	var buffer [1500]byte
	for {
		n, cAddr, err := l.proxyConn.ReadFromUDP(buffer[0:])
		if err != nil {
			continue
		}

		l.lock.Lock()
		if clientAddr, present := l.connections[cAddr.String()]; !present {
			// add a new clients
			//TODO: limit the number of cliens?
			if clientAddr != nil {
				l.connections[clientAddr.String()] = clientAddr

				go l.forwardTargetToClient(clientAddr)
			}
		}
		l.lock.Unlock()

		// write to the target
		_, err = l.targetConn.Write(buffer[0:n])
		if err != nil {
			continue
		}
	}
}
func (l *UDPChannel) Serve(address string, ports v1.ContainerPort) {
	l.addr = address
	l.ports = ports

	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", l.ports.ContainerPort))
	if err != nil {
		panic(err)
	}

	// store destination address
	l.targetAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", l.addr, l.ports.HostPort))
	if err != nil {
		panic(err)
	}

	l.proxyConn, err = net.ListenUDP("udp", localAddr)
	if err != nil {
		panic(err)
	}
	// Start the forwarding threads and account for connected clients
	go l.handleForwarding()
}
