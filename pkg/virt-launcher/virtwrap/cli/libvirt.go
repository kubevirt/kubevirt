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

//go:generate mockgen -source $GOFILE -imports "libvirt=github.com/berrange/libvirt-go" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/berrange/libvirt-go"
	utilwait "k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/pkg/log"
)

const ConnectionTimeout = 15 * time.Second
const ConnectionInterval = 10 * time.Second

// TODO: Should we handle libvirt connection errors transparent or panic?
type Connection interface {
	LookupDomainByName(name string) (VirDomain, error)
	DomainDefineXML(xml string) (VirDomain, error)
	Close() (int, error)
	DomainEventLifecycleRegister(callback libvirt.DomainEventLifecycleCallback) error
	ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]VirDomain, error)
	NewStream(flags libvirt.StreamFlags) (Stream, error)
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
	stop          chan struct{}
	reconnectLock *sync.Mutex
	callbacks     []libvirt.DomainEventLifecycleCallback
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

func (l *LibvirtConnection) NewStream(flags libvirt.StreamFlags) (Stream, error) {
	if err := l.reconnectIfNecessary(); err != nil {
		return nil, err
	}

	s, err := l.Connect.NewStream(flags)
	if err != nil {
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

	l.callbacks = append(l.callbacks, callback)
	_, err = l.Connect.DomainEventLifecycleRegister(nil, callback)
	return
}

func (l *LibvirtConnection) LookupDomainByName(name string) (dom VirDomain, err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	return l.Connect.LookupDomainByName(name)
}

func (l *LibvirtConnection) DomainDefineXML(xml string) (dom VirDomain, err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	dom, err = l.Connect.DomainDefineXML(xml)
	return
}

func (l *LibvirtConnection) ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]VirDomain, error) {
	if err := l.reconnectIfNecessary(); err != nil {
		return nil, err
	}

	virDoms, err := l.Connect.ListAllDomains(flags)
	if err != nil {
		return nil, err
	}
	doms := make([]VirDomain, len(virDoms))
	for i, d := range virDoms {
		doms[i] = &d
	}
	return doms, nil
}

// Installs a watchdog which will check periodically if the libvirt connection is still alive.
func (l *LibvirtConnection) installWatchdog(checkInterval time.Duration) {
	go func() {
		for {
			select {

			case <-l.stop:
				return

			case <-time.After(checkInterval):
				err := l.reconnectIfNecessary()

				if err != nil {
					// Connection is not alive but we have no error
					log.Log.Error("Connection to libvirt lost")
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
	alive, err := l.Connect.IsAlive()
	if err != nil {
		return err
	}
	if !alive {
		l.Connect, err = newConnection(l.uri, l.user, l.pass)
		if err != nil {
			return
		}
		cbs := l.callbacks
		l.callbacks = make([]libvirt.DomainEventLifecycleCallback, 0)
		for _, cb := range cbs {
			// Notify the callback about the reconnect by sending a nil event.
			// This way we give the callback a chance to emit an error to the watcher
			// ListWatcher will re-register automatically afterwards
			cb(l.Connect, nil, nil)
		}
	}
	return nil
}

type VirDomain interface {
	GetState() (libvirt.DomainState, int, error)
	Create() error
	Resume() error
	Destroy() error
	Shutdown() error
	GetName() (string, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags libvirt.DomainXMLFlags) (string, error)
	Undefine() error
	OpenConsole(devname string, stream *libvirt.Stream, flags libvirt.DomainConsoleFlags) error
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
		Connect: virConn, user: user, pass: pass, uri: uri,
		callbacks:     make([]libvirt.DomainEventLifecycleCallback, 0),
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
