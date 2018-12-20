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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package cli

//go:generate mockgen -source $GOFILE -imports "libvirt=github.com/libvirt/libvirt-go" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"
	"io"
	"sync"
	"time"

	libvirt "github.com/libvirt/libvirt-go"
	utilwait "k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
)

const ConnectionTimeout = 15 * time.Second
const ConnectionInterval = 10 * time.Second

// TODO: Should we handle libvirt connection errors transparent or panic?
type Connection interface {
	LookupDomainByName(name string) (VirDomain, error)
	DomainDefineXML(xml string) (VirDomain, error)
	Close() (int, error)
	DomainEventLifecycleRegister(callback libvirt.DomainEventLifecycleCallback) error
	AgentEventLifecycleRegister(callback libvirt.DomainEventAgentLifecycleCallback) error
	ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]VirDomain, error)
	NewStream(flags libvirt.StreamFlags) (Stream, error)
	SetReconnectChan(reconnect chan bool)
	QemuAgentCommand(command string, domainName string) (string, error)
}

type Stream interface {
	io.ReadWriteCloser
	UnderlyingStream() *libvirt.Stream
}

type VirStream struct {
	*libvirt.Stream
}

type LibvirtConnection struct {
	Connect       *libvirt.Connect
	user          string
	pass          string
	uri           string
	alive         bool
	stop          chan struct{}
	reconnect     chan bool
	reconnectLock *sync.Mutex
}

func (s *VirStream) Write(p []byte) (n int, err error) {
	return s.Stream.Send(p)
}

func (s *VirStream) Read(p []byte) (n int, err error) {
	return s.Stream.Recv(p)
}

/*
Close the stream and free its resources. Since closing a stream involves multiple calls with errors,
the first error occurred will be returned. The stream will always be freed.
*/
func (s *VirStream) Close() (e error) {
	e = s.Finish()
	if e != nil {
		return s.Free()
	}
	s.Free()
	return e
}

func (s *VirStream) UnderlyingStream() *libvirt.Stream {
	return s.Stream
}

func (l *LibvirtConnection) SetReconnectChan(reconnect chan bool) {
	l.reconnect = reconnect
}

func (l *LibvirtConnection) NewStream(flags libvirt.StreamFlags) (Stream, error) {
	if err := l.reconnectIfNecessary(); err != nil {
		return nil, err
	}

	s, err := l.Connect.NewStream(flags)
	if err != nil {
		l.checkConnectionLost(err)
		return nil, err
	}
	return &VirStream{Stream: s}, nil
}

func (l *LibvirtConnection) Close() (int, error) {
	close(l.stop)
	return l.Connect.Close()
}

func (l *LibvirtConnection) DomainEventLifecycleRegister(callback libvirt.DomainEventLifecycleCallback) (err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	_, err = l.Connect.DomainEventLifecycleRegister(nil, callback)
	l.checkConnectionLost(err)
	return
}

func (l *LibvirtConnection) AgentEventLifecycleRegister(callback libvirt.DomainEventAgentLifecycleCallback) (err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	_, err = l.Connect.DomainEventAgentLifecycleRegister(nil, callback)
	l.checkConnectionLost(err)
	return
}

func (l *LibvirtConnection) LookupDomainByName(name string) (dom VirDomain, err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	domain, err := l.Connect.LookupDomainByName(name)
	l.checkConnectionLost(err)
	return domain, err
}

func (l *LibvirtConnection) DomainDefineXML(xml string) (dom VirDomain, err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	dom, err = l.Connect.DomainDefineXML(xml)
	l.checkConnectionLost(err)
	return
}

func (l *LibvirtConnection) ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]VirDomain, error) {
	if err := l.reconnectIfNecessary(); err != nil {
		return nil, err
	}

	virDoms, err := l.Connect.ListAllDomains(flags)
	if err != nil {
		l.checkConnectionLost(err)
		return nil, err
	}
	doms := make([]VirDomain, len(virDoms))
	for i, d := range virDoms {
		doms[i] = &d
	}
	return doms, nil
}

// Execute a command on the Qemu guest agent
// command - the qemu command, for example this gets the interfaces: {"execute":"guest-network-get-interfaces"}
// domainName -  the qemu domain name
func (l *LibvirtConnection) QemuAgentCommand(command string, domainName string) (string, error) {
	domain, err := l.Connect.LookupDomainByName(domainName)
	result, err := domain.QemuAgentCommand(command, libvirt.DOMAIN_QEMU_AGENT_COMMAND_DEFAULT, uint32(0))
	return result, err
}

// Installs a watchdog which will check periodically if the libvirt connection is still alive.
func (l *LibvirtConnection) installWatchdog(checkInterval time.Duration) {
	go func() {
		for {
			select {

			case <-l.stop:
				return

			case <-time.After(checkInterval):
				l.reconnectIfNecessary()

				alive, err := l.Connect.IsAlive()

				// If the connection is ok, continue
				if alive {
					continue
				}

				if err == nil {
					// Connection is not alive but we have no error
					log.Log.Error("Connection to libvirt lost")
					l.reconnectLock.Lock()
					l.alive = false
					l.reconnectLock.Unlock()
				} else {
					// Do the usual error check to determine if the connection is lost
					l.checkConnectionLost(err)
				}
			}
		}
	}()
}

func (l *LibvirtConnection) reconnectIfNecessary() (err error) {
	l.reconnectLock.Lock()
	defer l.reconnectLock.Unlock()
	// TODO add a reconnect backoff, and immediately return an error in these cases
	// We need this to avoid swamping libvirt with reconnect tries
	if !l.alive {
		l.Connect, err = newConnection(l.uri, l.user, l.pass)
		if err != nil {
			return
		}
		l.alive = true

		if l.reconnect != nil {
			// Notify the callback about the reconnect through channel.
			// This way we give the callback a chance to emit an error to the watcher
			// ListWatcher will re-register automatically afterwards
			l.reconnect <- true
		}
	}
	return nil
}

func (l *LibvirtConnection) checkConnectionLost(err error) {
	l.reconnectLock.Lock()
	defer l.reconnectLock.Unlock()

	if errors.IsOk(err) {
		return
	}

	libvirtError, ok := err.(libvirt.Error)
	if !ok {
		return
	}

	switch libvirtError.Code {
	case
		libvirt.ERR_INTERNAL_ERROR,
		libvirt.ERR_INVALID_CONN,
		libvirt.ERR_AUTH_CANCELLED,
		libvirt.ERR_NO_MEMORY,
		libvirt.ERR_AUTH_FAILED,
		libvirt.ERR_SYSTEM_ERROR,
		libvirt.ERR_RPC:
		l.alive = false
		log.Log.With("code", libvirtError.Code).Reason(libvirtError).Error("Connection to libvirt lost.")
	}
}

type VirDomain interface {
	GetState() (libvirt.DomainState, int, error)
	Create() error
	Resume() error
	DestroyFlags(flags libvirt.DomainDestroyFlags) error
	ShutdownFlags(flags libvirt.DomainShutdownFlags) error
	Undefine() error
	GetName() (string, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags libvirt.DomainXMLFlags) (string, error)
	GetMetadata(tipus libvirt.DomainMetadataType, uri string, flags libvirt.DomainModificationImpact) (string, error)
	OpenConsole(devname string, stream *libvirt.Stream, flags libvirt.DomainConsoleFlags) error
	Migrate(*libvirt.Connect, libvirt.DomainMigrateFlags, string, string, uint64) (*libvirt.Domain, error)
	Free() error
}

func NewConnection(uri string, user string, pass string, checkInterval time.Duration) (Connection, error) {
	logger := log.Log
	logger.V(1).Infof("Connecting to libvirt daemon: %s", uri)

	var err error
	var virConn *libvirt.Connect

	err = utilwait.PollImmediate(ConnectionInterval, ConnectionTimeout, func() (done bool, err error) {
		virConn, err = newConnection(uri, user, pass)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to libvirt daemon: %v", err)
	}
	logger.V(1).Info("Connected to libvirt daemon")

	lvConn := &LibvirtConnection{
		Connect: virConn, user: user, pass: pass, uri: uri, alive: true,
		reconnectLock: &sync.Mutex{},
		stop:          make(chan struct{}),
	}
	lvConn.installWatchdog(checkInterval)

	return lvConn, nil
}

// TODO: needs a functional test.
func newConnection(uri string, user string, pass string) (*libvirt.Connect, error) {
	callback := func(creds []*libvirt.ConnectCredential) {
		for _, cred := range creds {
			if cred.Type == libvirt.CRED_AUTHNAME {
				cred.Result = user
				cred.ResultLen = len(cred.Result)
			} else if cred.Type == libvirt.CRED_PASSPHRASE {
				cred.Result = pass
				cred.ResultLen = len(cred.Result)
			}
		}
	}
	auth := &libvirt.ConnectAuth{
		CredType: []libvirt.ConnectCredentialType{
			libvirt.CRED_AUTHNAME, libvirt.CRED_PASSPHRASE,
		},
		Callback: callback,
	}
	virConn, err := libvirt.NewConnectWithAuth(uri, auth, 0)

	return virConn, err
}

func IsDown(domState libvirt.DomainState) bool {
	switch domState {
	case libvirt.DOMAIN_NOSTATE, libvirt.DOMAIN_SHUTDOWN, libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_CRASHED:
		return true

	}
	return false
}

func IsPaused(domState libvirt.DomainState) bool {
	switch domState {
	case libvirt.DOMAIN_PAUSED:
		return true

	}
	return false
}
