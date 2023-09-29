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

//go:generate mockgen -source $GOFILE -imports "libvirt=libvirt.org/go/libvirt" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"encoding/xml"
	"fmt"
	"io"
	"sync"
	"time"

	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"libvirt.org/go/libvirt"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/statsconv"
)

const ConnectionTimeout = 15 * time.Second
const ConnectionInterval = 500 * time.Millisecond

// TODO: Should we handle libvirt connection errors transparent or panic?
type Connection interface {
	LookupDomainByName(name string) (VirDomain, error)
	DomainDefineXML(xml string) (VirDomain, error)
	Close() (int, error)
	DomainEventLifecycleRegister(callback libvirt.DomainEventLifecycleCallback) error
	DomainEventDeviceAddedRegister(callback libvirt.DomainEventDeviceAddedCallback) error
	DomainEventDeviceRemovedRegister(callback libvirt.DomainEventDeviceRemovedCallback) error
	AgentEventLifecycleRegister(callback libvirt.DomainEventAgentLifecycleCallback) error
	VolatileDomainEventDeviceRemovedRegister(domain VirDomain, callback libvirt.DomainEventDeviceRemovedCallback) (int, error)
	DomainEventMemoryDeviceSizeChangeRegister(callback libvirt.DomainEventMemoryDeviceSizeChangeCallback) error
	DomainEventDeregister(registrationID int) error
	ListAllDomains(flags libvirt.ConnectListAllDomainsFlags) ([]VirDomain, error)
	NewStream(flags libvirt.StreamFlags) (Stream, error)
	SetReconnectChan(reconnect chan bool)
	QemuAgentCommand(command string, domainName string) (string, error)
	GetAllDomainStats(statsTypes libvirt.DomainStatsTypes, flags libvirt.ConnectGetAllDomainStatsFlags) ([]libvirt.DomainStats, error)
	// helper method, not found in libvirt
	// We add this helper to
	// 1. avoid to expose to the client code the libvirt-specific return type, see docs in stats/ subpackage
	// 2. transparently handling the addition of the memory stats, currently (libvirt 4.9) not handled by the bulk stats API
	GetDomainStats(statsTypes libvirt.DomainStatsTypes, l *stats.DomainJobInfo, flags libvirt.ConnectGetAllDomainStatsFlags) ([]*stats.DomainStats, error)
	GetQemuVersion() (string, error)
	GetSEVInfo() (*api.SEVNodeParameters, error)
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

	domainEventCallbacks                        []libvirt.DomainEventLifecycleCallback
	domainDeviceAddedEventCallbacks             []libvirt.DomainEventDeviceAddedCallback
	domainDeviceRemovedEventCallbacks           []libvirt.DomainEventDeviceRemovedCallback
	domainEventMigrationIterationCallbacks      []libvirt.DomainEventMigrationIterationCallback
	agentEventCallbacks                         []libvirt.DomainEventAgentLifecycleCallback
	domainDeviceMemoryDeviceSizeChangeCallbacks []libvirt.DomainEventMemoryDeviceSizeChangeCallback
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
func (s *VirStream) Close() error {
	errFinish := s.Finish()
	errFree := s.Free()
	if errFinish != nil {
		return errFinish
	}
	return errFree
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
	if l.Connect != nil {
		return l.Connect.Close()
	} else {
		return 0, nil
	}
}

func (l *LibvirtConnection) DomainEventLifecycleRegister(callback libvirt.DomainEventLifecycleCallback) (err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	l.domainEventCallbacks = append(l.domainEventCallbacks, callback)
	_, err = l.Connect.DomainEventLifecycleRegister(nil, callback)
	l.checkConnectionLost(err)
	return
}

func (l *LibvirtConnection) DomainEventDeviceAddedRegister(callback libvirt.DomainEventDeviceAddedCallback) (err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	l.domainDeviceAddedEventCallbacks = append(l.domainDeviceAddedEventCallbacks, callback)
	_, err = l.Connect.DomainEventDeviceAddedRegister(nil, callback)
	l.checkConnectionLost(err)
	return
}

func (l *LibvirtConnection) DomainEventDeviceRemovedRegister(callback libvirt.DomainEventDeviceRemovedCallback) (err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	l.domainDeviceRemovedEventCallbacks = append(l.domainDeviceRemovedEventCallbacks, callback)
	_, err = l.VolatileDomainEventDeviceRemovedRegister(nil, callback)
	l.checkConnectionLost(err)
	return
}

func (l *LibvirtConnection) AgentEventLifecycleRegister(callback libvirt.DomainEventAgentLifecycleCallback) (err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	l.agentEventCallbacks = append(l.agentEventCallbacks, callback)
	_, err = l.Connect.DomainEventAgentLifecycleRegister(nil, callback)
	l.checkConnectionLost(err)
	return
}

func (l *LibvirtConnection) VolatileDomainEventDeviceRemovedRegister(domain VirDomain, callback libvirt.DomainEventDeviceRemovedCallback) (int, error) {
	var dom *libvirt.Domain
	if domain != nil {
		dom = domain.(*libvirt.Domain)
	}
	return l.Connect.DomainEventDeviceRemovedRegister(dom, callback)
}

func (l *LibvirtConnection) DomainEventMemoryDeviceSizeChangeRegister(callback libvirt.DomainEventMemoryDeviceSizeChangeCallback) (err error) {
	if err = l.reconnectIfNecessary(); err != nil {
		return
	}

	l.domainDeviceMemoryDeviceSizeChangeCallbacks = append(l.domainDeviceMemoryDeviceSizeChangeCallbacks, callback)
	_, err = l.Connect.DomainEventMemoryDeviceSizeChangeRegister(nil, callback)
	l.checkConnectionLost(err)
	return
}

func (l *LibvirtConnection) DomainEventDeregister(registrationID int) error {
	return l.Connect.DomainEventDeregister(registrationID)
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
	for i := range virDoms {
		doms[i] = &virDoms[i]
	}
	return doms, nil
}

// Execute a command on the Qemu guest agent
// command - the qemu command, for example this gets the interfaces: {"execute":"guest-network-get-interfaces"}
// domainName -  the qemu domain name
func (l *LibvirtConnection) QemuAgentCommand(command string, domainName string) (string, error) {
	if err := l.reconnectIfNecessary(); err != nil {
		return "", err
	}
	domain, err := l.Connect.LookupDomainByName(domainName)
	if err != nil {
		return "", err
	}
	defer domain.Free()
	result, err := domain.QemuAgentCommand(command, libvirt.DOMAIN_QEMU_AGENT_COMMAND_DEFAULT, uint32(0))
	return result, err
}

func (l *LibvirtConnection) GetAllDomainStats(statsTypes libvirt.DomainStatsTypes, flags libvirt.ConnectGetAllDomainStatsFlags) ([]libvirt.DomainStats, error) {
	if err := l.reconnectIfNecessary(); err != nil {
		return nil, err
	}

	doms := []*libvirt.Domain{}
	domStats, err := l.Connect.GetAllDomainStats(doms, statsTypes, flags)
	if err != nil {
		l.checkConnectionLost(err)
		return nil, err
	}
	return domStats, nil
}

func (l *LibvirtConnection) GetQemuVersion() (string, error) {
	version, err := l.Connect.GetVersion()
	if err != nil {
		return "", err
	}
	// The following code works because version it's an uint32 var. Therefore, the divisions will result in
	// an integer number. For instance:
	// version = 7002002
	// major = 7002002 / 1000000 = 7 --> decimals are discard in this operation
	// version = 7002002 - (7*1000000) = 2000
	// minor = 2002 / 1000 = 2
	// version = version - (2*1000) = 2
	// release = 2
	major := version / 1000000
	version = version - (major * 1000000)
	minor := version / 1000
	version = version - (minor * 1000)
	release := version

	return fmt.Sprintf("QEMU %d.%d.%d", major, minor, release), err
}

func (l *LibvirtConnection) GetDomainStats(statsTypes libvirt.DomainStatsTypes, migrateJobInfo *stats.DomainJobInfo, flags libvirt.ConnectGetAllDomainStatsFlags) ([]*stats.DomainStats, error) {
	domStats, err := l.GetAllDomainStats(statsTypes, flags)
	if err != nil {
		return nil, err
	}
	// Free memory allocated for domains
	defer func() {
		for i := range domStats {
			err := domStats[i].Domain.Free()
			if err != nil {
				log.Log.Reason(err).Warning("Error freeing a domain.")
			}
		}
	}()

	var list []*stats.DomainStats
	for i, domStat := range domStats {
		var err error

		memStats, err := domStat.Domain.MemoryStats(uint32(libvirt.DOMAIN_MEMORY_STAT_NR), 0)
		if err != nil {
			return list, err
		}

		devAliasMap, err := l.GetDeviceAliasMap(domStat.Domain)
		if err != nil {
			return list, err
		}

		domInfo, err := domStat.Domain.GetInfo()
		if err != nil {
			return list, err
		}

		stat := &stats.DomainStats{}
		err = statsconv.Convert_libvirt_DomainStats_to_stats_DomainStats(statsconv.DomainIdentifier(domStat.Domain), &domStats[i], memStats, domInfo, devAliasMap, migrateJobInfo, stat)
		if err != nil {
			return list, err
		}

		cpuMap, err := domStat.Domain.GetVcpuPinInfo(libvirt.DOMAIN_AFFECT_CURRENT)
		if err != nil {
			return list, err
		}

		stat.CPUMap = cpuMap
		stat.CPUMapSet = true

		list = append(list, stat)
	}

	return list, nil
}

func (l *LibvirtConnection) GetSEVInfo() (*api.SEVNodeParameters, error) {
	const flags = uint32(0)
	params, err := l.Connect.GetSEVInfo(flags)
	if err != nil {
		return nil, err
	}

	sevNodeParameters := &api.SEVNodeParameters{}
	if params.PDHSet {
		sevNodeParameters.PDH = params.PDH
	}
	if params.CertChainSet {
		sevNodeParameters.CertChain = params.CertChain
	}

	return sevNodeParameters, nil
}

func (l *LibvirtConnection) GetDeviceAliasMap(domain *libvirt.Domain) (map[string]string, error) {
	devAliasMap := make(map[string]string)

	domSpec := &api.DomainSpec{}
	domxml, err := domain.GetXMLDesc(0)
	if err != nil {
		return devAliasMap, err
	}
	err = xml.Unmarshal([]byte(domxml), domSpec)
	if err != nil {
		return devAliasMap, err
	}

	for _, iface := range domSpec.Devices.Interfaces {
		devAliasMap[iface.Target.Device] = iface.Alias.GetName()
	}

	for _, disk := range domSpec.Devices.Disks {
		devAliasMap[disk.Target.Device] = disk.Alias.GetName()
	}

	return devAliasMap, nil
}

// Installs a watchdog which will check periodically if the libvirt connection is still alive.
func (l *LibvirtConnection) installWatchdog(checkInterval time.Duration) {
	go func() {
		for {
			select {

			case <-l.stop:
				return

			case <-time.After(checkInterval):
				var alive bool
				var err error
				err = l.reconnectIfNecessary()
				if l.Connect != nil {
					alive, err = l.Connect.IsAlive()
				}

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

		log.Log.Info("Established new Libvirt Connection")

		for _, callback := range l.domainEventCallbacks {
			log.Log.Info("Re-registered domain callback")
			_, err = l.Connect.DomainEventLifecycleRegister(nil, callback)
		}
		for _, callback := range l.domainEventMigrationIterationCallbacks {
			log.Log.Info("Re-registered iteration callback")
			_, err = l.Connect.DomainEventMigrationIterationRegister(nil, callback)
		}
		for _, callback := range l.agentEventCallbacks {
			log.Log.Info("Re-registered agent callback")
			_, err = l.Connect.DomainEventAgentLifecycleRegister(nil, callback)
		}
		for _, callback := range l.domainDeviceAddedEventCallbacks {
			log.Log.Info("Re-registered domain device added callback")
			_, err = l.Connect.DomainEventDeviceAddedRegister(nil, callback)
		}
		for _, callback := range l.domainDeviceRemovedEventCallbacks {
			log.Log.Info("Re-registered domain device removed callback")
			_, err = l.Connect.DomainEventDeviceRemovedRegister(nil, callback)
		}
		for _, callback := range l.domainDeviceMemoryDeviceSizeChangeCallbacks {
			log.Log.Info("Re-registered domain memory device size change callback")
			_, err = l.Connect.DomainEventMemoryDeviceSizeChangeRegister(nil, callback)
		}

		log.Log.Error("Re-registered domain and agent callbacks for new connection")

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
	CreateWithFlags(flags libvirt.DomainCreateFlags) error
	Suspend() error
	Resume() error
	BlockResize(disk string, size uint64, flags libvirt.DomainBlockResizeFlags) error
	GetBlockInfo(disk string, flags uint32) (*libvirt.DomainBlockInfo, error)
	AttachDevice(xml string) error
	AttachDeviceFlags(xml string, flags libvirt.DomainDeviceModifyFlags) error
	UpdateDeviceFlags(xml string, flags libvirt.DomainDeviceModifyFlags) error
	DetachDevice(xml string) error
	DetachDeviceFlags(xml string, flags libvirt.DomainDeviceModifyFlags) error
	DestroyFlags(flags libvirt.DomainDestroyFlags) error
	ShutdownFlags(flags libvirt.DomainShutdownFlags) error
	Reboot(flags libvirt.DomainRebootFlagValues) error
	UndefineFlags(flags libvirt.DomainUndefineFlagsValues) error
	GetName() (string, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags libvirt.DomainXMLFlags) (string, error)
	GetMetadata(tipus libvirt.DomainMetadataType, uri string, flags libvirt.DomainModificationImpact) (string, error)
	OpenConsole(devname string, stream *libvirt.Stream, flags libvirt.DomainConsoleFlags) error
	MigrateToURI3(string, *libvirt.DomainMigrateParameters, libvirt.DomainMigrateFlags) error
	MigrateStartPostCopy(flags uint32) error
	MemoryStats(nrStats uint32, flags uint32) ([]libvirt.DomainMemoryStat, error)
	GetJobStats(flags libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error)
	GetJobInfo() (*libvirt.DomainJobInfo, error)
	GetDiskErrors(flags uint32) ([]libvirt.DomainDiskError, error)
	SetTime(secs int64, nsecs uint, flags libvirt.DomainSetTimeFlags) error
	AuthorizedSSHKeysGet(user string, flags libvirt.DomainAuthorizedSSHKeysFlags) ([]string, error)
	AuthorizedSSHKeysSet(user string, keys []string, flags libvirt.DomainAuthorizedSSHKeysFlags) error
	AbortJob() error
	Free() error
	CoreDumpWithFormat(to string, format libvirt.DomainCoreDumpFormat, flags libvirt.DomainCoreDumpFlags) error
	PinVcpuFlags(vcpu uint, cpuMap []bool, flags libvirt.DomainModificationImpact) error
	PinEmulator(cpumap []bool, flags libvirt.DomainModificationImpact) error
	SetVcpusFlags(vcpu uint, flags libvirt.DomainVcpuFlags) error
	GetLaunchSecurityInfo(flags uint32) (*libvirt.DomainLaunchSecurityParameters, error)
	SetLaunchSecurityState(params *libvirt.DomainLaunchSecurityStateParameters, flags uint32) error
}

func NewConnection(uri string, user string, pass string, checkInterval time.Duration) (Connection, error) {
	return NewConnectionWithTimeout(uri, user, pass, checkInterval, ConnectionInterval, ConnectionTimeout)
}

func NewConnectionWithTimeout(uri string, user string, pass string, checkInterval, connectionInterval, connectionTimeout time.Duration) (Connection, error) {
	logger := log.Log
	logger.V(1).Infof("Connecting to libvirt daemon: %s", uri)

	var err error
	var virConn *libvirt.Connect

	err = utilwait.PollImmediate(connectionInterval, connectionTimeout, func() (done bool, err error) {
		virConn, err = newConnection(uri, user, pass)
		if err != nil {
			logger.V(1).Infof("Connecting to libvirt daemon failed: %v", err)
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
