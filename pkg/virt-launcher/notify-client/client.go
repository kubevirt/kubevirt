package eventsclient

import (
	"encoding/json"
	"fmt"
	"net/rpc"
	"path/filepath"

	"github.com/libvirt/libvirt-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/log"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type DomainEventClient struct {
	client *rpc.Client
}

func NewDomainEventClient(virtShareDir string) (*DomainEventClient, error) {
	socketPath := filepath.Join(virtShareDir, "domain-notify.sock")
	conn, err := rpc.Dial("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("client failed to connect to domain notifier socket: %s", socketPath)
		return nil, err
	}

	return &DomainEventClient{client: conn}, nil
}

func (c *DomainEventClient) SendDomainEvent(event watch.Event) error {

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
	args := &notifyserver.Args{
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

func libvirtEventCallback(c cli.Connection, domain *api.Domain, event *libvirt.DomainEventLifecycle, client *DomainEventClient, events chan watch.Event) {

	// check for reconnects, and emit an error to force a resync
	if event == nil {
		client.SendDomainEvent(newWatchEventError(fmt.Errorf("Libvirt reconnect")))
		return
	}
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
		spec, err := util.GetDomainSpec(status, d)
		if err != nil {
			if !domainerrors.IsNotFound(err) {
				log.Log.Reason(err).Error("Could not fetch the Domain specification.")
				client.SendDomainEvent(newWatchEventError(err))
				return
			}
		} else {
			domain.Spec = *spec
			domain.ObjectMeta.UID = spec.Metadata.KubeVirt.UID
		}

		log.Log.Infof("kubevirt domain status: %v(%v):%v(%v)", domain.Status.Status, status, domain.Status.Reason, reason)
	}

	switch domain.Status.Reason {
	case api.ReasonNonExistent:
		event := watch.Event{Type: watch.Deleted, Object: domain}
		client.SendDomainEvent(event)
		events <- event
	default:
		if event.Event == libvirt.DOMAIN_EVENT_DEFINED && libvirt.DomainEventDefinedDetailType(event.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
			event := watch.Event{Type: watch.Added, Object: domain}
			client.SendDomainEvent(event)
			events <- event
		} else if event.Event == libvirt.DOMAIN_EVENT_STARTED && libvirt.DomainEventStartedDetailType(event.Detail) == libvirt.DOMAIN_EVENT_STARTED_MIGRATED {
			event := watch.Event{Type: watch.Added, Object: domain}
			client.SendDomainEvent(event)
			events <- event
		} else {
			client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})
		}
	}
}

func StartNotifier(virtShareDir string, domainConn cli.Connection, deleteNotificationSent chan watch.Event, vmiUID types.UID) error {
	type LibvirtEvent struct {
		Domain string
		Event  *libvirt.DomainEventLifecycle
	}

	eventChan := make(chan LibvirtEvent, 10)

	// Run the event process logic in a separate go-routine to not block libvirt
	go func() {
		for event := range eventChan {
			// TODO don't make a client every single time
			client, err := NewDomainEventClient(virtShareDir)
			if err != nil {
				log.Log.Reason(err).Error("Unable to create domain event notify client")
				continue
			}

			libvirtEventCallback(domainConn, util.NewDomainFromName(event.Domain, vmiUID), event.Event, client, deleteNotificationSent)
			log.Log.Info("processed event")
		}
	}()

	entrypointCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
		log.Log.Infof("Libvirt event %d with reason %d received", event.Event, event.Detail)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info("Could not determine name of libvirt domain in event callback.")
		}
		select {
		case eventChan <- LibvirtEvent{Event: event, Domain: name}:
		default:
			log.Log.Infof("Libvirt event channel is full, dropping event.")
		}
	}
	err := domainConn.DomainEventLifecycleRegister(entrypointCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event callback with libvirt")
		return err
	}
	log.Log.Infof("Registered libvirt event notify callback")
	return nil
}
