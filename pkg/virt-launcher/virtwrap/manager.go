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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package virtwrap

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/xml"
	"fmt"

	"github.com/libvirt/libvirt-go"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sync"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"os"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	"kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/registry-disk"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type DomainManager interface {
	SyncVMI(vmi *v1.VirtualMachineInstance, useEmulation bool) error
	KillVMI(vmi *v1.VirtualMachineInstance) error
	DeleteVMI(vmi *v1.VirtualMachineInstance) error
	SignalShutdownVMI(vmi *v1.VirtualMachineInstance) error
	ListAllDomains() ([]*api.Domain, error)
}

type LazyLibvirtDomainManager struct {
	pointerLock  *sync.Mutex
	_manager     DomainManager
	StopChan     chan struct{}
	events       chan watch.Event
	virtShareDir string
	initOnce     *sync.Once
}

func NewLazyLibvirtDomainManager(virtShareDir string, stopChan chan struct{}) (manager DomainManager, events chan watch.Event) {
	events = make(chan watch.Event, 10)
	return &LazyLibvirtDomainManager{
		StopChan:     stopChan,
		pointerLock:  &sync.Mutex{},
		events:       events,
		virtShareDir: virtShareDir,
		initOnce:     &sync.Once{},
	}, events
}

func (l *LazyLibvirtDomainManager) SyncVMI(vmi *v1.VirtualMachineInstance, useEmulation bool) error {
	pending, err := l.initialSync(vmi, useEmulation)
	if err != nil {
		return err
	}
	if pending {
		return nil
	}
	return l.get().SyncVMI(vmi, useEmulation)
}

func (l *LazyLibvirtDomainManager) KillVMI(vmi *v1.VirtualMachineInstance) error {
	if l.get() == nil {
		return nil
	}
	return l.get().KillVMI(vmi)
}

func (l *LazyLibvirtDomainManager) DeleteVMI(vmi *v1.VirtualMachineInstance) error {
	if l.get() == nil {
		return nil
	}
	return l.get().DeleteVMI(vmi)
}

func (l *LazyLibvirtDomainManager) SignalShutdownVMI(vmi *v1.VirtualMachineInstance) error {
	if l.get() == nil {
		return nil
	}
	return l.get().SignalShutdownVMI(vmi)
}

func (l *LazyLibvirtDomainManager) get() DomainManager {
	l.pointerLock.Lock()
	defer l.pointerLock.Unlock()
	return l._manager
}

func (l *LazyLibvirtDomainManager) initialSync(vmi *v1.VirtualMachineInstance, useEmulation bool) (pending bool, err error) {
	l.pointerLock.Lock()
	defer l.pointerLock.Unlock()

	var domainCon cli.Connection

	if l._manager == nil {
		l.initOnce.Do(func() {
			util.StartLibvirt(l.StopChan)
			go func() {
				domainCon, err = cli.NewConnection("qemu:///system", "", "", 10*time.Second)
				if err != nil {
					log.Log.Object(vmi).Reason(err).Error("Could not establish initial connection with libvirt, exiting.")
					l.fatalError(vmi, err)
				}
				util.StartDomainEventMonitoring()
				err = eventsclient.StartNotifier(l.virtShareDir, domainCon, l.events)
				if err != nil {
					log.Log.Object(vmi).Reason(err).Error("Could not establish initial connection with libvirt, exiting.")
					l.fatalError(vmi, err)
				}
				manager := NewLibvirtDomainManager(domainCon)
				err := manager.SyncVMI(vmi, useEmulation)
				if err != nil {
					log.Log.Object(vmi).Reason(err).Error("Initial synchronization with libvirt failed.")
					l.syncError(vmi, api.ReasonDomainSyncError, err)
				}

				l.pointerLock.Lock()
				defer l.pointerLock.Unlock()
				l._manager = manager
			}()
		})
		return true, nil
	}

	return false, nil
}

func (l LazyLibvirtDomainManager) fatalError(vmi *v1.VirtualMachineInstance, syncError error) {
	l.syncError(vmi, api.ReasonLibvirtUnreachable, syncError)
	os.Exit(1)
}

func (l LazyLibvirtDomainManager) syncError(vmi *v1.VirtualMachineInstance, reason api.StateChangeReason, syncError error) {
	cli, err := eventsclient.NewDomainEventClient(l.virtShareDir)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Can't inform virt-handler about a sync error.")
		return
	}
	err = cli.SendErrorDomainEvent(vmi.Name, vmi.Namespace, vmi.UID, reason, syncError)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Can't inform virt-handler about a sync error.")
		return
	}
}

func (l LazyLibvirtDomainManager) ListAllDomains() ([]*api.Domain, error) {
	if l.get() == nil {
		return nil, nil
	}
	return l.get().ListAllDomains()
}

type LibvirtDomainManager struct {
	virConn cli.Connection
}

func NewLibvirtDomainManager(connection cli.Connection) DomainManager {
	manager := LibvirtDomainManager{
		virConn: connection,
	}

	return &manager
}

// All local environment setup that needs to occur before VirtualMachineInstance starts
// can be done in this function. This includes things like...
//
// - storage prep
// - network prep
// - cloud-init
//
// The Domain.Spec can be alterned in this function and any changes
// made to the domain will get set in libvirt after this function exits.
func (l *LibvirtDomainManager) preStartHook(vmi *v1.VirtualMachineInstance, domain *api.Domain) (*api.Domain, error) {

	// ensure registry disk files have correct ownership privileges
	err := registrydisk.SetFilePermissions(vmi)
	if err != nil {
		return domain, err
	}

	// generate cloud-init data
	cloudInitData := cloudinit.GetCloudInitNoCloudSource(vmi)
	if cloudInitData != nil {
		hostname := dns.SanitizeHostname(vmi)

		err := cloudinit.GenerateLocalData(vmi.Name, hostname, vmi.Namespace, cloudInitData)
		if err != nil {
			return domain, err
		}
	}

	// setup networking
	err = network.SetupPodNetwork(vmi, domain)
	if err != nil {
		return domain, err
	}

	// Create images for volumes that are marked ephemeral.
	err = ephemeraldisk.CreateEphemeralImages(vmi)
	if err != nil {
		return domain, err
	}
	// create empty disks if they exist
	if err := emptydisk.CreateTemporaryDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating empty disks failed: %v", err)
	}

	hooksManager := hooks.GetManager()
	domainSpec, err := hooksManager.OnDefineDomain(&domain.Spec, vmi)
	if err != nil {
		return domain, err
	}
	domain.Spec = *domainSpec

	return domain, err
}

func (l *LibvirtDomainManager) SyncVMI(vmi *v1.VirtualMachineInstance, useEmulation bool) error {
	logger := log.Log.Object(vmi)

	domain := &api.Domain{}

	// Map the VirtualMachineInstance to the Domain
	c := &api.ConverterContext{
		VirtualMachine: vmi,
		UseEmulation:   useEmulation,
	}
	if err := api.Convert_v1_VirtualMachine_To_api_Domain(vmi, domain, c); err != nil {
		logger.Error("Conversion failed.")
		return err
	}

	// Set defaults which are not coming from the cluster
	api.SetObjectDefaults_Domain(domain)

	dom, err := l.virConn.LookupDomainByName(domain.Spec.Name)
	newDomain := false
	if err != nil {
		// We need the domain but it does not exist, so create it
		if domainerrors.IsNotFound(err) {
			newDomain = true
			domain, err = l.preStartHook(vmi, domain)
			if err != nil {
				logger.Reason(err).Error("pre start setup for VirtualMachineInstance failed.")
				return err
			}
			dom, err = util.SetDomainSpec(l.virConn, vmi, domain.Spec)
			if err != nil {
				return err
			}
			logger.Info("Domain defined.")
		} else {
			logger.Reason(err).Error("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()
	domState, _, err := dom.GetState()
	if err != nil {
		logger.Reason(err).Error("Getting the domain state failed.")
		return err
	}

	// To make sure, that we set the right qemu wrapper arguments,
	// we update the domain XML whenever a VirtualMachineInstance was already defined but not running
	if !newDomain && cli.IsDown(domState) {
		dom, err = util.SetDomainSpec(l.virConn, vmi, domain.Spec)
		if err != nil {
			return err
		}
	}

	// TODO Suspend, Pause, ..., for now we only support reaching the running state
	// TODO for migration and error detection we also need the state change reason
	// TODO blocked state
	if cli.IsDown(domState) {
		err = dom.Create()
		if err != nil {
			logger.Reason(err).Error("Starting the VirtualMachineInstance failed.")
			return err
		}
		logger.Info("Domain started.")
	} else if cli.IsPaused(domState) {
		// TODO: if state change reason indicates a system error, we could try something smarter
		err := dom.Resume()
		if err != nil {
			logger.Reason(err).Error("Resuming the VirtualMachineInstance failed.")
			return err
		}
		logger.Info("Domain resumed.")
	} else {
		// Nothing to do
	}

	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return err
	}

	var newSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &newSpec)
	if err != nil {
		logger.Reason(err).Error("Parsing domain XML failed.")
		return err
	}

	// TODO: check if VirtualMachineInstance Spec and Domain Spec are equal or if we have to sync
	return nil
}

func (l *LibvirtDomainManager) getDomainSpec(dom cli.VirDomain) (*api.DomainSpec, error) {
	return util.GetDomainSpec(dom)
}

func (l *LibvirtDomainManager) SignalShutdownVMI(vmi *v1.VirtualMachineInstance) error {
	domName := util.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VirtualMachineInstance does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed during graceful shutdown.")
			return err
		}
	}
	defer dom.Free()

	domState, _, err := dom.GetState()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		domSpec, err := l.getDomainSpec(dom)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Unable to retrieve domain xml")
			return err
		}

		if domSpec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp == nil {
			err = dom.ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_ACPI_POWER_BTN)
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("Signalling graceful shutdown failed.")
				return err
			}
			log.Log.Object(vmi).Infof("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())

			now := k8sv1.Now()
			domSpec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp = &now
			_, err = util.SetDomainSpec(l.virConn, vmi, *domSpec)
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("Unable to update grace period start time on domain xml")
				return err
			}
		}
	}

	return nil
}

func (l *LibvirtDomainManager) KillVMI(vmi *v1.VirtualMachineInstance) error {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VirtualMachineInstance does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()
	// TODO: Graceful shutdown
	domState, _, err := dom.GetState()
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		log.Log.Object(vmi).Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED || domState == libvirt.DOMAIN_SHUTDOWN {
		err = dom.DestroyFlags(libvirt.DOMAIN_DESTROY_GRACEFUL)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				return nil
			}
			log.Log.Object(vmi).Reason(err).Error("Destroying the domain state failed.")
			return err
		}
		log.Log.Object(vmi).Info("Domain stopped.")
		return nil
	}

	log.Log.Object(vmi).Info("Domain not running or paused, nothing to do.")
	return nil
}

func (l *LibvirtDomainManager) DeleteVMI(vmi *v1.VirtualMachineInstance) error {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the domain does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()

	err = dom.Undefine()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Undefining the domain failed.")
		return err
	}
	log.Log.Object(vmi).Info("Domain undefined.")
	return nil
}

func (l *LibvirtDomainManager) ListAllDomains() ([]*api.Domain, error) {

	doms, err := l.virConn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, err
	}

	var list []*api.Domain
	for _, dom := range doms {
		domain, err := util.NewDomain(dom)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		spec, err := util.GetDomainSpec(dom)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		domain.Spec = *spec
		status, reason, err := dom.GetState()
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		domain.SetState(util.ConvState(status), util.ConvReason(status, reason))
		list = append(list, domain)
		dom.Free()
	}

	return list, nil
}
