package eventsclient

import (
	"encoding/json"
	"fmt"
	"net/rpc"
	"path/filepath"
	"time"

	"k8s.io/client-go/tools/reference"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"

	k8sv1 "k8s.io/api/core/v1"

	libvirt "github.com/libvirt/libvirt-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/log"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type NotifyClient struct {
	client *rpc.Client
}

type libvirtEvent struct {
	Domain     string
	Event      *libvirt.DomainEventLifecycle
	AgentEvent *libvirt.DomainEventAgentLifecycle
}

func NewNotifyClient(virtShareDir string) (*NotifyClient, error) {
	socketPath := filepath.Join(virtShareDir, "domain-notify.sock")
	conn, err := rpc.Dial("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("client failed to connect to domain notifier socket: %s", socketPath)
		return nil, err
	}

	return &NotifyClient{client: conn}, nil
}

func (c *NotifyClient) SendDomainEvent(event watch.Event) error {

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
	args := &notifyserver.DomainEventArgs{
		DomainJSON: string(domainJSON),
		StatusJSON: string(statusJSON),
		EventType:  string(event.Type),
	}
	reply := &notifyserver.Reply{}

	err = c.client.Call("Notify.DomainEvent", args, reply)
	if err != nil {
		return err
	} else if reply.Success != true {
		msg := fmt.Sprintf("failed to notify domain event: %s", reply.Message)
		return fmt.Errorf(msg)
	}

	return nil
}

func newWatchEventError(err error) watch.Event {
	return watch.Event{Type: watch.Error, Object: &metav1.Status{Status: metav1.StatusFailure, Message: err.Error()}}
}

func eventCallback(c cli.Connection, domain *api.Domain, libvirtEvent libvirtEvent, client *NotifyClient, events chan watch.Event, interfaceStatus *[]api.InterfaceStatus) {
	d, err := c.LookupDomainByName(util.DomainFromNamespaceName(domain.ObjectMeta.Namespace, domain.ObjectMeta.Name))
	if err != nil {
		if !domainerrors.IsNotFound(err) {
			log.Log.Reason(err).Error("Could not fetch the Domain.")
			client.SendDomainEvent(newWatchEventError(err))
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
				client.SendDomainEvent(newWatchEventError(err))
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
				client.SendDomainEvent(newWatchEventError(err))
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
		watchEvent := watch.Event{Type: watch.Deleted, Object: domain}
		client.SendDomainEvent(watchEvent)
		events <- watchEvent
	default:
		if libvirtEvent.Event != nil {
			if libvirtEvent.Event.Event == libvirt.DOMAIN_EVENT_DEFINED && libvirt.DomainEventDefinedDetailType(libvirtEvent.Event.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
				event := watch.Event{Type: watch.Added, Object: domain}
				client.SendDomainEvent(event)
				events <- event
			} else if libvirtEvent.Event.Event == libvirt.DOMAIN_EVENT_STARTED && libvirt.DomainEventStartedDetailType(libvirtEvent.Event.Detail) == libvirt.DOMAIN_EVENT_STARTED_MIGRATED {
				event := watch.Event{Type: watch.Added, Object: domain}
				client.SendDomainEvent(event)
				events <- event
			}
		}
		if interfaceStatus != nil {
			domain.Status.Interfaces = *interfaceStatus
			event := watch.Event{Type: watch.Modified, Object: domain}
			client.SendDomainEvent(event)
			events <- event
		}
		client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})
	}
}

func (c *NotifyClient) StartDomainNotifier(domainConn cli.Connection, deleteNotificationSent chan watch.Event, vmiUID types.UID, qemuAgentPollerInterval *time.Duration) error {
	eventChan := make(chan libvirtEvent, 10)
	agentUpdateChan := make(chan agentpoller.AgentUpdateEvent, 10)

	reconnectChan := make(chan bool, 10)

	domainConn.SetReconnectChan(reconnectChan)

	agentPoller := agentpoller.CreatePoller(domainConn, vmiUID, agentUpdateChan, qemuAgentPollerInterval)

	// Run the event process logic in a separate go-routine to not block libvirt
	go func() {
		var interfaceStatuses *[]api.InterfaceStatus
		for {
			select {
			case event := <-eventChan:
				domain := util.NewDomainFromName(event.Domain, vmiUID)
				eventCallback(domainConn, domain, event, c, deleteNotificationSent, interfaceStatuses)
				agentPoller.UpdateDomain(domain)
				if event.AgentEvent != nil {
					if event.AgentEvent.State == libvirt.CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_CONNECTED {
						agentPoller.Start()
					} else if event.AgentEvent.State == libvirt.CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_DISCONNECTED {
						agentPoller.Stop()
					}
				}
				log.Log.Info("processed event")
			case agentUpdate := <-agentUpdateChan:
				interfaceStatuses = agentUpdate.InterfaceStatuses
				domainName := agentUpdate.DomainName
				eventCallback(domainConn, util.NewDomainFromName(domainName, vmiUID), libvirtEvent{}, c, deleteNotificationSent, interfaceStatuses)
			case <-reconnectChan:
				c.SendDomainEvent(newWatchEventError(fmt.Errorf("Libvirt reconnect")))
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

func (c *NotifyClient) SendK8sEvent(vmi *v1.VirtualMachineInstance, severity string, reason string, message string) error {

	reply := &notifyserver.Reply{}

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

	err = c.client.Call("Notify.K8sEvent", event, reply)
	if err != nil {
		return err
	} else if reply.Success != true {
		msg := fmt.Sprintf("failed to notify k8s event: %s", reply.Message)
		return fmt.Errorf(msg)
	}

	return nil
}
