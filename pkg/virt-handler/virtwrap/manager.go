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

package virtwrap

//go:generate mockgen -source $GOFILE -imports "libvirt=github.com/libvirt/libvirt-go" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/xml"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jeevatkm/go-model"
	"github.com/libvirt/libvirt-go"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

type DomainManager interface {
	SyncVM(*v1.VM) (*api.DomainSpec, error)
	KillVM(*v1.VM) error
}

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
	alive         bool
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
the first error occured will be returned. The stream will always be freed.
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
	defer l.checkConnectionLost()

	s, err := l.Connect.NewStream(flags)
	if err != nil {
		return nil, err
	}
	return &VirStream{Stream: s}, nil
}

func (l *LibvirtConnection) Close() (int, error) {
	close(l.stop)
	return l.Close()
}

func (l *LibvirtConnection) DomainEventLifecycleRegister(callback libvirt.DomainEventLifecycleCallback) (err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}
	defer l.checkConnectionLost()

	l.callbacks = append(l.callbacks, callback)
	_, err = l.Connect.DomainEventLifecycleRegister(nil, callback)
	return
}

func (l *LibvirtConnection) LookupDomainByName(name string) (dom VirDomain, err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}
	defer l.checkConnectionLost()

	return l.Connect.LookupDomainByName(name)
}

func (l *LibvirtConnection) DomainDefineXML(xml string) (dom VirDomain, err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}
	defer l.checkConnectionLost()

	dom, err = l.Connect.DomainDefineXML(xml)
	return
}

func (l *LibvirtConnection) ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]VirDomain, error) {
	if err := l.reconnectIfNecessary(); err != nil {
		return nil, err
	}
	defer l.checkConnectionLost()

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
				l.reconnectIfNecessary()

				alive, err := l.Connect.IsAlive()

				// If the connection is ok, continue
				if alive {
					continue
				}

				if err == nil {
					// Connection is not alive but we have no error
					logging.DefaultLogger().Error().Msg("Connection to libvirt lost")
					l.reconnectLock.Lock()
					l.alive = false
					l.reconnectLock.Unlock()
				} else {
					// Do the usual error check to determine if the connection is lost
					l.checkConnectionLost()
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

func (l *LibvirtConnection) checkConnectionLost() {
	l.reconnectLock.Lock()
	defer l.reconnectLock.Unlock()

	err := libvirt.GetLastError()
	if err.Code == libvirt.ERR_OK {
		return
	}

	switch err.Code {
	case
		libvirt.ERR_INTERNAL_ERROR,
		libvirt.ERR_INVALID_CONN,
		libvirt.ERR_AUTH_CANCELLED,
		libvirt.ERR_NO_MEMORY,
		libvirt.ERR_AUTH_FAILED,
		libvirt.ERR_SYSTEM_ERROR,
		libvirt.ERR_RPC:
		l.alive = false
		logging.DefaultLogger().Error().Reason(err).With("code", err.Code).Msg("Connection to libvirt lost.")
	}
}

type VirDomain interface {
	GetState() (libvirt.DomainState, int, error)
	Create() error
	Resume() error
	Destroy() error
	GetName() (string, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags libvirt.DomainXMLFlags) (string, error)
	Undefine() error
	OpenConsole(devname string, stream *libvirt.Stream, flags libvirt.DomainConsoleFlags) error
	Free() error
}

type LibvirtDomainManager struct {
	virConn  Connection
	recorder record.EventRecorder
}

func waitForLibvirt(uri string, user string, pass string, timeout time.Duration) error {
	interval := 10 * time.Second
	return utilwait.PollImmediate(interval, timeout, func() (done bool, err error) {
		if virConn, err := newConnection(uri, user, pass); err == nil {
			defer virConn.Close()
			return true, nil
		}
		return false, nil
	})
}

func NewConnection(uri string, user string, pass string, checkInterval time.Duration) (Connection, error) {
	timeout := 15 * time.Second
	logger := logging.DefaultLogger()
	logger.Info().V(1).Msgf("Connecting to libvirt daemon: %s", uri)
	if err := waitForLibvirt(uri, user, pass, timeout); err != nil {
		return nil, fmt.Errorf("cannot connect to libvirt daemon: %v", err)
	}
	logger.Info().V(1).Msg("Connected to libvirt daemon")
	virConn, err := newConnection(uri, user, pass)
	if err != nil {
		return nil, err
	}
	lvConn := &LibvirtConnection{
		Connect: virConn, user: user, pass: pass, uri: uri, alive: true,
		callbacks:     make([]libvirt.DomainEventLifecycleCallback, 0),
		reconnectLock: &sync.Mutex{},
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

func NewLibvirtDomainManager(connection Connection, recorder record.EventRecorder) (DomainManager, error) {
	manager := LibvirtDomainManager{virConn: connection, recorder: recorder}
	return &manager, nil
}

func VMNamespaceKeyFunc(vm *v1.VM) string {
	// Construct the domain name with a namespace prefix. E.g. namespace>name
	domName := fmt.Sprintf("%s_%s", vm.GetObjectMeta().GetNamespace(), vm.GetObjectMeta().GetName())
	return domName
}

func (l *LibvirtDomainManager) SyncVM(vm *v1.VM) (*api.DomainSpec, error) {
	var wantedSpec api.DomainSpec
	mappingErrs := model.Copy(&wantedSpec, vm.Spec.Domain)

	if len(mappingErrs) > 0 {
		return nil, errors.NewAggregate(mappingErrs)
	}

	domName := VMNamespaceKeyFunc(vm)
	wantedSpec.Name = domName
	wantedSpec.UUID = string(vm.GetObjectMeta().GetUID())
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// We need the domain but it does not exist, so create it
		if err.(libvirt.Error).Code == libvirt.ERR_NO_DOMAIN {
			xmlStr, err := xml.Marshal(&wantedSpec)
			if err != nil {
				logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Generating the domain XML failed.")
				return nil, err
			}
			logging.DefaultLogger().Object(vm).Info().V(3).Msgf("Domain XML generated %s.", xmlStr)
			dom, err = l.virConn.DomainDefineXML(string(xmlStr))
			if err != nil {
				logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Defining the VM failed.")
				return nil, err
			}
			logging.DefaultLogger().Object(vm).Info().Msg("Domain defined.")
			l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Created.String(), "VM defined")
		} else {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain failed.")
			return nil, err
		}
	}
	defer dom.Free()
	domState, _, err := dom.GetState()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain state failed.")
		return nil, err
	}
	// TODO Suspend, Pause, ..., for now we only support reaching the running state
	// TODO for migration and error detection we also need the state change reason
	//state := LifeCycleTranslationMap[domState[0]]
	switch domState {
	case libvirt.DOMAIN_NOSTATE, libvirt.DOMAIN_SHUTDOWN, libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_CRASHED:
		err := dom.Create()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Starting the VM failed.")
			return nil, err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain started.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Started.String(), "VM started.")
	case libvirt.DOMAIN_PAUSED:
		// TODO: if state change reason indicates a system error, we could try something smarter
		err := dom.Resume()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Resuming the VM failed.")
			return nil, err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain resumed.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Resumed.String(), "VM resumed")
	default:
		// Nothing to do
		// TODO: blocked state
	}

	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	var newSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &newSpec)
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Parsing domain XML failed.")
		return nil, err
	}

	// TODO: check if VM Spec and Domain Spec are equal or if we have to sync
	return &newSpec, nil
}

func (l *LibvirtDomainManager) KillVM(vm *v1.VM) error {
	domName := VMNamespaceKeyFunc(vm)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VM does not exist, we are done
		if err.(libvirt.Error).Code == libvirt.ERR_NO_DOMAIN {
			return nil
		} else {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()
	// TODO: Graceful shutdown
	domState, _, err := dom.GetState()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		err = dom.Destroy()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Destroying the domain state failed.")
			return err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain stopped.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Stopped.String(), "VM stopped")
	}

	err = dom.Undefine()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Undefining the domain state failed.")
		return err
	}
	logging.DefaultLogger().Object(vm).Info().Msg("Domain undefined.")
	l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Deleted.String(), "VM undefined")
	return nil
}
