package eventsclient

import (
	"fmt"
	"sync"
	"time"

	"github.com/libvirt/libvirt-go"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/reference"

	v12 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type Notifier struct {
	DomainEventStore *Store
	K8sEventStore    *Store
}

type Store struct {
	value   interface{}
	lock    sync.Mutex
	channel chan struct{}
}

func (s *Store) Get() interface{} {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.value
}

func (s *Store) Set(value interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.value = value
	select {
	case s.channel <- struct{}{}:
	default:
	}
}

func (s *Store) UpdateChan() chan struct{} {
	return s.channel
}

type libvirtEvent struct {
	Domain     string
	Event      *libvirt.DomainEventLifecycle
	AgentEvent *libvirt.DomainEventAgentLifecycle
}

func NewNotifier() *Notifier {
	return &Notifier{
		DomainEventStore: &Store{channel: make(chan struct{}, 1)},
		K8sEventStore:    &Store{channel: make(chan struct{}, 1)},
	}
}

func (n *Notifier) EnqueueDomainEvent(event watch.Event) error {

	var domainJSON []byte
	var statusJSON []byte
	var err error

	if event.Type == watch.Error {
		status := event.Object.(*metav1.Status)
		statusJSON, err = json.Marshal(status)
		if err != nil {
			return err
		}
	} else {
		domain := event.Object.(*api.Domain)
		domainJSON, err = json.Marshal(domain)
		if err != nil {
			return err
		}
	}
	request := v12.DomainEventRequest{
		DomainJSON: domainJSON,
		StatusJSON: statusJSON,
		EventType:  string(event.Type),
	}
	n.DomainEventStore.Set(&request)
	return nil
}

func newWatchEventError(err error) watch.Event {
	return watch.Event{Type: watch.Error, Object: &metav1.Status{Status: metav1.StatusFailure, Message: err.Error()}}
}

func eventCallback(c cli.Connection, domain *api.Domain, libvirtEvent libvirtEvent, client *Notifier, events chan watch.Event,
	interfaceStatus []api.InterfaceStatus, osInfo *api.GuestOSInfo) {
	d, err := c.LookupDomainByName(util.DomainFromNamespaceName(domain.ObjectMeta.Namespace, domain.ObjectMeta.Name))
	if err != nil {
		if !domainerrors.IsNotFound(err) {
			log.Log.Reason(err).Error("Could not fetch the Domain.")
			client.EnqueueDomainEvent(newWatchEventError(err))
			return
		}
		domain.SetState(api.NoState, api.ReasonNonExistent)
	} else {
		defer d.Free()

		// No matter which event, try to fetch the domain xml
		// and the state. If we get a IsNotFound error, that
		// means that the VirtualMachineInstance was removed.
		status, reason, err := d.GetState()
		if err != nil {
			if !domainerrors.IsNotFound(err) {
				log.Log.Reason(err).Error("Could not fetch the Domain state.")
				client.EnqueueDomainEvent(newWatchEventError(err))
				return
			}
			domain.SetState(api.NoState, api.ReasonNonExistent)
		} else {
			domain.SetState(util.ConvState(status), util.ConvReason(status, reason))
		}

		spec, err := util.GetDomainSpecWithRuntimeInfo(status, d)
		if err != nil {
			// NOTE: Getting domain metadata for a live-migrating VM isn't allowed
			if !domainerrors.IsNotFound(err) && !domainerrors.IsInvalidOperation(err) {
				log.Log.Reason(err).Error("Could not fetch the Domain specification.")
				client.EnqueueDomainEvent(newWatchEventError(err))
				return
			}
		} else {
			domain.ObjectMeta.UID = spec.Metadata.KubeVirt.UID
		}
		if spec != nil {
			domain.Spec = *spec
		}

		log.Log.Infof("kubevirt domain status: %v(%v):%v(%v)", domain.Status.Status, status, domain.Status.Reason, reason)
	}

	switch domain.Status.Reason {
	case api.ReasonNonExistent:
		watchEvent := watch.Event{Type: watch.Modified, Object: domain}
		now := metav1.Now()
		domain.ObjectMeta.DeletionTimestamp = &now
		client.EnqueueDomainEvent(watchEvent)
		events <- watchEvent
	default:
		event := watch.Event{
			Type:   watch.Modified,
			Object: domain,
		}
		if libvirtEvent.Event != nil {
			if libvirtEvent.Event.Event == libvirt.DOMAIN_EVENT_DEFINED && libvirt.DomainEventDefinedDetailType(libvirtEvent.Event.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
				event.Type = watch.Added
			} else if libvirtEvent.Event.Event == libvirt.DOMAIN_EVENT_STARTED && libvirt.DomainEventStartedDetailType(libvirtEvent.Event.Detail) == libvirt.DOMAIN_EVENT_STARTED_MIGRATED {
				event.Type = watch.Added
			}
		}
		if interfaceStatus != nil {
			domain.Status.Interfaces = interfaceStatus
		}
		if osInfo != nil {
			log.Log.V(4).Infof("7) OSINFO IN EVENT CALLBACK: %v", osInfo)
			domain.Status.OSInfo = *osInfo
		}
		client.EnqueueDomainEvent(event)
		events <- event
	}
}

func (n *Notifier) StartDomainNotifier(
	domainConn cli.Connection,
	deleteNotificationSent chan watch.Event,
	vmiUID types.UID,
	domainName string,
	agentStore *agentpoller.AsyncAgentStore,
	qemuAgentSysInterval time.Duration,
	qemuAgentFileInterval time.Duration,
	qemuAgentUserInterval time.Duration,
	qemuAgentVersionInterval time.Duration,
) error {
	eventChan := make(chan libvirtEvent, 10)

	reconnectChan := make(chan bool, 10)

	var domainCache *api.Domain

	domainConn.SetReconnectChan(reconnectChan)

	agentPoller := agentpoller.CreatePoller(
		domainConn,
		vmiUID,
		domainName,
		agentStore,
		qemuAgentSysInterval,
		qemuAgentFileInterval,
		qemuAgentUserInterval,
		qemuAgentVersionInterval,
	)

	// Run the event process logic in a separate go-routine to not block libvirt
	go func() {
		var interfaceStatuses []api.InterfaceStatus
		var guestOsInfo *api.GuestOSInfo
		for {
			select {
			case event := <-eventChan:
				domainCache = util.NewDomainFromName(event.Domain, vmiUID)
				eventCallback(domainConn, domainCache, event, n, deleteNotificationSent, interfaceStatuses, guestOsInfo)
				log.Log.Infof("Domain name event: %v", domainCache.Spec.Name)
				if event.AgentEvent != nil {
					if event.AgentEvent.State == libvirt.CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_CONNECTED {
						agentPoller.Start()
					} else if event.AgentEvent.State == libvirt.CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_DISCONNECTED {
						agentPoller.Stop()
					}
				}
			case agentUpdate := <-agentStore.AgentUpdated:
				interfaceStatuses = agentUpdate.DomainInfo.Interfaces
				guestOsInfo = agentUpdate.DomainInfo.OSInfo
				if domainCache != nil && interfaceStatuses != nil {
					interfaceStatuses = agentpoller.MergeAgentStatusesWithDomainData(domainCache.Spec.Devices.Interfaces, interfaceStatuses)
				}

				eventCallback(domainConn, domainCache, libvirtEvent{}, n, deleteNotificationSent,
					interfaceStatuses, guestOsInfo)
			case <-reconnectChan:
				n.EnqueueDomainEvent(newWatchEventError(fmt.Errorf("Libvirt reconnect")))
				return
			}
		}
	}()

	domainEventLifecycleCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
		log.Log.Infof("DomainLifecycle event %d with reason %d received", event.Event, event.Detail)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info("Could not determine name of libvirt domain in event callback.")
		}
		select {
		case eventChan <- libvirtEvent{Event: event, Domain: name}:
		default:
			log.Log.Infof("Libvirt event channel is full, dropping event.")
		}
	}
	err := domainConn.DomainEventLifecycleRegister(domainEventLifecycleCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event callback with libvirt")
		return err
	}

	agentEventLifecycleCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventAgentLifecycle) {
		log.Log.Infof("GuestAgentLifecycle event state %d with reason %d received", event.State, event.Reason)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info("Could not determine name of libvirt domain in event callback.")
		}
		select {
		case eventChan <- libvirtEvent{AgentEvent: event, Domain: name}:
		default:
			log.Log.Infof("Libvirt event channel is full, dropping event.")
		}
	}
	err = domainConn.AgentEventLifecycleRegister(agentEventLifecycleCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event callback with libvirt")
		return err
	}

	log.Log.Infof("Registered libvirt event notify callback")
	return nil
}

func (n *Notifier) EnqueueK8sEvent(vmi *v1.VirtualMachineInstance, severity string, reason string, message string) error {

	vmiRef, err := reference.GetReference(v1.Scheme, vmi)
	if err != nil {
		return err
	}

	event := k8sv1.Event{
		InvolvedObject: *vmiRef,
		Type:           severity,
		Reason:         reason,
		Message:        message,
	}

	json, err := json.Marshal(event)
	if err != nil {
		return err
	}

	request := v12.K8SEventRequest{
		EventJSON: json,
	}
	n.K8sEventStore.Set(&request)
	return nil
}
