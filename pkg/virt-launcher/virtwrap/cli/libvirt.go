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
 *
 */

package cli

//go:generate mockgen -source $GOFILE -imports "libvirt=libvirt.org/go/libvirt" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
	"time"

	"libvirt.org/go/libvirt"

	"kubevirt.io/client-go/log"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	virterrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
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
	DomainEventJobCompletedRegister(callback libvirt.DomainEventJobCompletedCallback) error
	DomainEventLifecycleRegister(callback libvirt.DomainEventLifecycleCallback) error
	DomainEventRebootRegister(callback libvirt.DomainEventGenericCallback) error
	DomainEventDeviceAddedRegister(callback libvirt.DomainEventDeviceAddedCallback) error
	DomainEventDeviceRemovedRegister(callback libvirt.DomainEventDeviceRemovedCallback) error
	DomainEventMigrationIterationRegister(callback libvirt.DomainEventMigrationIterationCallback) (unsubscribe func())
	AgentEventLifecycleRegister(callback libvirt.DomainEventAgentLifecycleCallback) error
	VolatileDomainEventDeviceRemovedRegister(domain VirDomain, callback libvirt.DomainEventDeviceRemovedCallback) (unsubscribe func(), err error)
	VolatileDomainEventBlockJobRegister(callback libvirt.DomainEventBlockJobCallback) (unsubscribe func())
	DomainEventMemoryDeviceSizeChangeRegister(callback libvirt.DomainEventMemoryDeviceSizeChangeCallback) error
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
	GetDomainDirtyRate(calculationDuration time.Duration, flags libvirt.DomainDirtyRateCalcFlags) ([]*stats.DomainStatsDirtyRate, error)
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

type dispatcher[T any] struct {
	lock        sync.RWMutex
	subscribers []*T
}

func (d *dispatcher[T]) subscribe(callback T) (unsubscribe func()) {
	d.lock.Lock()
	defer d.lock.Unlock()

	subscriber := &callback
	d.subscribers = append(d.subscribers, subscriber)
	return func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		d.subscribers = slices.DeleteFunc(d.subscribers, func(s *T) bool {
			return s == subscriber
		})
	}
}

func (d *dispatcher[T]) callbacks() []T {
	d.lock.RLock()
	defer d.lock.RUnlock()

	callbacks := make([]T, 0, len(d.subscribers))
	for _, subscriber := range d.subscribers {
		callbacks = append(callbacks, *subscriber)
	}
	return callbacks
}

type LibvirtConnection struct {
	Connect       *libvirt.Connect
	dispatcherIDs []int
	user          string
	pass          string
	uri           string
	alive         bool
	stop          chan struct{}
	reconnect     chan bool
	reconnectLock *sync.Mutex

	lifecycle              dispatcher[libvirt.DomainEventLifecycleCallback]
	reboot                 dispatcher[libvirt.DomainEventGenericCallback]
	jobCompleted           dispatcher[libvirt.DomainEventJobCompletedCallback]
	deviceAdded            dispatcher[libvirt.DomainEventDeviceAddedCallback]
	deviceRemoved          dispatcher[libvirt.DomainEventDeviceRemovedCallback]
	agentLifecycle         dispatcher[libvirt.DomainEventAgentLifecycleCallback]
	memoryDeviceSizeChange dispatcher[libvirt.DomainEventMemoryDeviceSizeChangeCallback]
	migrationIteration     dispatcher[libvirt.DomainEventMigrationIterationCallback]
	blockJob               dispatcher[libvirt.DomainEventBlockJobCallback]
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

func (l *LibvirtConnection) SetReconnectChan(reconnect chan bool) {
	l.reconnect = reconnect
}

func (l *LibvirtConnection) Close() (int, error) {
	close(l.stop)
	if l.Connect != nil {
		return l.Connect.Close()
	} else {
		return 0, nil
	}
}

type eventCallback[E any] interface {
	~func(*libvirt.Connect, *libvirt.Domain, *E)
}

// registerDispatcher registers one libvirt callback that forwards each event to all of d's subscribers.
func registerDispatcher[T eventCallback[E], E any](
	d *dispatcher[T],
	register func(*libvirt.Domain, T) (int, error),
) (int, error) {
	return register(nil, func(c *libvirt.Connect, dom *libvirt.Domain, event *E) {
		for _, callback := range d.callbacks() {
			callback(c, dom, event)
		}
	})
}

func (l *LibvirtConnection) registerDispatchers(conn *libvirt.Connect) (ids []int, err error) {
	track := func(id int, registerErr error) {
		if registerErr != nil {
			err = errors.Join(err, registerErr)
			return
		}
		ids = append(ids, id)
	}

	track(registerDispatcher(&l.lifecycle, conn.DomainEventLifecycleRegister))
	track(registerDispatcher(&l.jobCompleted, conn.DomainEventJobCompletedRegister))
	track(registerDispatcher(&l.deviceAdded, conn.DomainEventDeviceAddedRegister))
	track(registerDispatcher(&l.deviceRemoved, conn.DomainEventDeviceRemovedRegister))
	track(registerDispatcher(&l.agentLifecycle, conn.DomainEventAgentLifecycleRegister))
	track(registerDispatcher(&l.memoryDeviceSizeChange, conn.DomainEventMemoryDeviceSizeChangeRegister))
	track(registerDispatcher(&l.migrationIteration, conn.DomainEventMigrationIterationRegister))
	track(registerDispatcher(&l.blockJob, conn.DomainEventBlockJob2Register))
	track(conn.DomainEventRebootRegister(nil, func(c *libvirt.Connect, dom *libvirt.Domain) {
		for _, callback := range l.reboot.callbacks() {
			callback(c, dom)
		}
	}))

	if err != nil {
		deregisterCallbacks(conn, ids)
		return nil, err
	}
	return ids, nil
}

func deregisterCallbacks(conn *libvirt.Connect, ids []int) {
	if conn == nil {
		return
	}
	for _, id := range ids {
		if err := conn.DomainEventDeregister(id); err != nil {
			log.Log.Reason(err).V(4).Infof("Failed to deregister callback %d from libvirt connection", id)
		}
	}
}

func (l *LibvirtConnection) DomainEventJobCompletedRegister(callback libvirt.DomainEventJobCompletedCallback) error {
	l.jobCompleted.subscribe(callback)
	return nil
}

func (l *LibvirtConnection) DomainEventLifecycleRegister(callback libvirt.DomainEventLifecycleCallback) error {
	l.lifecycle.subscribe(callback)
	return nil
}

func (l *LibvirtConnection) DomainEventRebootRegister(callback libvirt.DomainEventGenericCallback) error {
	l.reboot.subscribe(callback)
	return nil
}

func (l *LibvirtConnection) DomainEventDeviceAddedRegister(callback libvirt.DomainEventDeviceAddedCallback) error {
	l.deviceAdded.subscribe(callback)
	return nil
}

func (l *LibvirtConnection) DomainEventDeviceRemovedRegister(callback libvirt.DomainEventDeviceRemovedCallback) error {
	l.deviceRemoved.subscribe(callback)
	return nil
}

func (l *LibvirtConnection) DomainEventMigrationIterationRegister(callback libvirt.DomainEventMigrationIterationCallback) (unsubscribe func()) {
	return l.migrationIteration.subscribe(callback)
}

func (l *LibvirtConnection) AgentEventLifecycleRegister(callback libvirt.DomainEventAgentLifecycleCallback) error {
	l.agentLifecycle.subscribe(callback)
	return nil
}

func (l *LibvirtConnection) VolatileDomainEventDeviceRemovedRegister(domain VirDomain, callback libvirt.DomainEventDeviceRemovedCallback) (unsubscribe func(), err error) {
	if domain == nil {
		return l.deviceRemoved.subscribe(callback), nil
	}
	name, err := domain.GetName()
	if err != nil {
		return nil, err
	}
	return l.deviceRemoved.subscribe(func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventDeviceRemoved) {
		if eventDomain, err := d.GetName(); err == nil && eventDomain == name {
			callback(c, d, event)
		}
	}), nil
}

func (l *LibvirtConnection) VolatileDomainEventBlockJobRegister(callback libvirt.DomainEventBlockJobCallback) (unsubscribe func()) {
	return l.blockJob.subscribe(callback)
}

func (l *LibvirtConnection) DomainEventMemoryDeviceSizeChangeRegister(callback libvirt.DomainEventMemoryDeviceSizeChangeCallback) error {
	l.memoryDeviceSizeChange.subscribe(callback)
	return nil
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

		dirtyRateInfo := domStat.DirtyRate

		stat := &stats.DomainStats{}
		err = statsconv.Convert_libvirt_DomainStats_to_stats_DomainStats(statsconv.DomainIdentifier(domStat.Domain), &domStats[i], memStats, migrateJobInfo, dirtyRateInfo, stat)
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

func (l *LibvirtConnection) GetDomainDirtyRate(calculationDuration time.Duration, flags libvirt.DomainDirtyRateCalcFlags) ([]*stats.DomainStatsDirtyRate, error) {
	domStats, err := l.GetAllDomainStats(libvirt.DOMAIN_STATS_DIRTYRATE, libvirt.CONNECT_GET_ALL_DOMAINS_STATS_RUNNING|libvirt.CONNECT_GET_ALL_DOMAINS_STATS_PAUSED)
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

	for i := range domStats {
		err := domStats[i].Domain.StartDirtyRateCalc(int(calculationDuration.Seconds()), flags)
		if err != nil {
			log.Log.Reason(err).Warning("Can't start dirty rate calc")
		}
	}

	// Wait for the dirty rate calculation to finish
	time.Sleep(calculationDuration)

	var dirtyRateStats []*stats.DomainStatsDirtyRate
	getDomainStats := func() ([]*stats.DomainStatsDirtyRate, error) {
		var dirtyRateStats []*stats.DomainStatsDirtyRate
		for _, domStat := range domStats {
			dirtyRateInfo := domStat.DirtyRate
			if dirtyRateInfo == nil {
				log.Log.Warning("Dirty rate info is nil")
				continue
			}
			// warn if dirty rate calc is not set or not finished
			if !dirtyRateInfo.CalcStatusSet {
				return nil, fmt.Errorf("dirty rate stats calculation status is not set")
			} else if dirtyRateInfo.CalcStatus != int(libvirt.DOMAIN_DIRTYRATE_MEASURED) {
				var status string
				switch dirtyRateInfo.CalcStatus {
				case int(libvirt.DOMAIN_DIRTYRATE_UNSTARTED):
					status = "DOMAIN_DIRTYRATE_UNSTARTED"
				case int(libvirt.DOMAIN_DIRTYRATE_MEASURING):
					status = "DOMAIN_DIRTYRATE_MEASURING"
				}

				log.Log.Infof("trying to get dirty rate stats, but dirty rate calc status is not finished: %s", status)
				return nil, fmt.Errorf("dirty calc status is %s", status)
			} else {
				dirtyRateStat := statsconv.Convert_libvirt_DomainStatsDirtyRate_To_stats_DomainStatsDirtyRate(dirtyRateInfo)
				dirtyRateStats = append(dirtyRateStats, dirtyRateStat)
			}
		}
		return dirtyRateStats, nil
	}
	retryTicker := time.NewTicker(200 * time.Millisecond)
	retryTimeout := time.After(1 * time.Second)

retryLoop:
	for {
		select {
		case <-retryTicker.C:
			dirtyRateStats, err = getDomainStats()
			if err == nil {
				break retryLoop
			}
		case <-retryTimeout:
			break retryLoop
		}
	}

	if err != nil || dirtyRateStats == nil {
		return nil, fmt.Errorf("failed to get dirty rate stats: %w", err)
	}

	return dirtyRateStats, nil
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
		if iface.Target != nil {
			devAliasMap[iface.Target.Device] = iface.Alias.GetName()
		}
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

func (l *LibvirtConnection) reconnectIfNecessary() error {
	l.reconnectLock.Lock()
	defer l.reconnectLock.Unlock()

	return l.reconnectIfNecessaryLocked()
}

// reconnectIfNecessaryLocked expects reconnectLock to be held.
func (l *LibvirtConnection) reconnectIfNecessaryLocked() (err error) {
	// TODO add a reconnect backoff, and immediately return an error in these cases
	// We need this to avoid swamping libvirt with reconnect tries
	if l.alive {
		return nil
	}

	newConn, err := newConnection(l.uri, l.user, l.pass)
	if err != nil {
		return err
	}

	log.Log.Info("Established new Libvirt Connection")

	newIDs, err := l.registerDispatchers(newConn)
	if err != nil {
		if _, closeErr := newConn.Close(); closeErr != nil {
			log.Log.Reason(closeErr).V(4).Info("Failed to close partially set up libvirt connection")
		}
		return err
	}

	oldConn, oldIDs := l.Connect, l.dispatcherIDs
	l.Connect, l.dispatcherIDs, l.alive = newConn, newIDs, true
	deregisterCallbacks(oldConn, oldIDs)

	log.Log.Info("Registered event dispatchers on the new libvirt connection")

	if l.reconnect != nil {
		l.reconnect <- true
	}
	return nil
}

func (l *LibvirtConnection) checkConnectionLost(err error) {
	l.reconnectLock.Lock()
	defer l.reconnectLock.Unlock()

	l.checkConnectionLostLocked(err)
}

// checkConnectionLostLocked expects reconnectLock to be held.
func (l *LibvirtConnection) checkConnectionLostLocked(err error) {
	if virterrors.IsOk(err) {
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
	CreateWithFlags(flags libvirt.DomainCreateFlags) error
	Suspend() error
	Resume() error
	BlockResize(disk string, size uint64, flags libvirt.DomainBlockResizeFlags) error
	GetBlockInfo(disk string, flags uint32) (*libvirt.DomainBlockInfo, error)
	BlockCommit(disk string, base string, top string, bandwidth uint64, flags libvirt.DomainBlockCommitFlags) error
	BlockJobAbort(disk string, flags libvirt.DomainBlockJobAbortFlags) error
	GetBlockJobInfo(disk string, flags libvirt.DomainBlockJobInfoFlags) (*libvirt.DomainBlockJobInfo, error)
	AttachDeviceFlags(xml string, flags libvirt.DomainDeviceModifyFlags) error
	UpdateDeviceFlags(xml string, flags libvirt.DomainDeviceModifyFlags) error
	DetachDeviceFlags(xml string, flags libvirt.DomainDeviceModifyFlags) error
	DestroyFlags(flags libvirt.DomainDestroyFlags) error
	ShutdownFlags(flags libvirt.DomainShutdownFlags) error
	Reboot(flags libvirt.DomainRebootFlagValues) error
	Reset(flags uint32) error
	UndefineFlags(flags libvirt.DomainUndefineFlagsValues) error
	GetName() (string, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags libvirt.DomainXMLFlags) (string, error)
	MigrateToURI3(string, *libvirt.DomainMigrateParameters, libvirt.DomainMigrateFlags) error
	MigrateStartPostCopy(flags uint32) error
	MigrateSetMaxDowntime(downtime uint64, flags uint32) error
	MemoryStats(nrStats uint32, flags uint32) ([]libvirt.DomainMemoryStat, error)
	GetJobStats(flags libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error)
	GetJobInfo() (*libvirt.DomainJobInfo, error)
	GetGuestInfo(types libvirt.DomainGuestInfoTypes, flags uint32) (*libvirt.DomainGuestInfo, error)
	GetDiskErrors(flags uint32) ([]libvirt.DomainDiskError, error)
	SetTime(secs int64, nsecs uint, flags libvirt.DomainSetTimeFlags) error
	SetUserPassword(user string, password string, flags libvirt.DomainSetUserPasswordFlags) error
	AuthorizedSSHKeysSet(user string, keys []string, flags libvirt.DomainAuthorizedSSHKeysFlags) error
	AbortJob() error
	Free() error
	CoreDumpWithFormat(to string, format libvirt.DomainCoreDumpFormat, flags libvirt.DomainCoreDumpFlags) error
	PinVcpuFlags(vcpu uint, cpuMap []bool, flags libvirt.DomainModificationImpact) error
	PinEmulator(cpumap []bool, flags libvirt.DomainModificationImpact) error
	SetVcpusFlags(vcpu uint, flags libvirt.DomainVcpuFlags) error
	GetLaunchSecurityInfo(flags uint32) (*libvirt.DomainLaunchSecurityParameters, error)
	SetLaunchSecurityState(params *libvirt.DomainLaunchSecurityStateParameters, flags uint32) error
	FDAssociate(name string, files []os.File, flags libvirt.DomainFDAssociateFlags) error
	FSFreeze(mounts []string, flags uint32) error
	FSThaw(mounts []string, flags uint32) error
	Screenshot(stream *libvirt.Stream, screen, flags uint32) (string, error)
	BackupBegin(backupXML string, checkpointXML string, flags libvirt.DomainBackupBeginFlags) error
	CreateCheckpointXML(xmlConfig string, flags libvirt.DomainCheckpointCreateFlags) (*libvirt.DomainCheckpoint, error)
	CreateSnapshotXML(xmlConfig string, flags libvirt.DomainSnapshotCreateFlags) (*libvirt.DomainSnapshot, error)
}

func NewConnection(uri string, user string, pass string, checkInterval time.Duration) (Connection, error) {
	return NewConnectionWithTimeout(uri, user, pass, checkInterval, ConnectionInterval, ConnectionTimeout)
}

func NewConnectionWithTimeout(uri string, user string, pass string, checkInterval, connectionInterval, connectionTimeout time.Duration) (Connection, error) {
	logger := log.Log
	logger.V(1).Infof("Connecting to libvirt daemon: %s", uri)

	var err error
	var virConn *libvirt.Connect

	err = virtwait.PollImmediately(connectionInterval, connectionTimeout, func(_ context.Context) (done bool, err error) {
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
	if lvConn.dispatcherIDs, err = lvConn.registerDispatchers(virConn); err != nil {
		if _, closeErr := virConn.Close(); closeErr != nil {
			logger.Reason(closeErr).V(4).Info("Failed to close libvirt connection")
		}
		return nil, fmt.Errorf("failed to register libvirt event dispatchers: %w", err)
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
