package eventsclient

import (
	"encoding/json"
	"fmt"
	"net/rpc"
	"path/filepath"

	"github.com/libvirt/libvirt-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/api/core/v1"

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

func (c *DomainEventClient) SendErrorDomainEvent(name string, namespace string, uid types.UID, reason api.StateChangeReason, err error) error {
	domain := api.NewDomainReferenceFromName(namespace, name)
	domain.GetObjectMeta().SetUID(uid)
	domain.Spec.Metadata.KubeVirt.UID = uid
	domain.Status.Status = api.Error
	domain.Status.Reason = reason
	domain.Status.Conditions = []api.DomainCondition{{
		Type:    api.DomainConditionSynchronized,
		Status:  v1.ConditionFalse,
		Reason:  string(reason),
		Message: err.Error(),
	}}
	return c.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})
}

func newWatchEventError(err error) watch.Event {
	return watch.Event{Type: watch.Error, Object: &metav1.Status{Status: metav1.StatusFailure, Message: err.Error()}}
}

func libvirtEventCallback(d cli.VirDomain, event *libvirt.DomainEventLifecycle, client *DomainEventClient, deleteNotificationSent chan watch.Event) {

	// check for reconnects, and emit an error to force a resync
	if event == nil {
		client.SendDomainEvent(newWatchEventError(fmt.Errorf("Libvirt reconnect")))
		return
	}

	domain, err := util.NewDomain(d)
	if err != nil {
		log.Log.Reason(err).Error("Could not create the Domain.")
		client.SendDomainEvent(newWatchEventError(err))
		return
	}

	// No matter which event, try to fetch the domain xml
	// and the state. If we get a IsNotFound error, that
	// means that the VirtualMachineInstance was removed.
	spec, err := util.GetDomainSpec(d)
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

	log.Log.Infof("domain status: %v:%v", status, reason)
	switch domain.Status.Reason {
	case api.ReasonNonExistent:
		event := watch.Event{Type: watch.Deleted, Object: domain}
		client.SendDomainEvent(event)
		deleteNotificationSent <- event
	default:
		if event.Event == libvirt.DOMAIN_EVENT_DEFINED && libvirt.DomainEventDefinedDetailType(event.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
			client.SendDomainEvent(watch.Event{Type: watch.Added, Object: domain})
		} else {
			client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})
		}
	}
}

func StartNotifier(virtShareDir string, domainConn cli.Connection, deleteNotificationSent chan watch.Event) error {
	entrypointCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
		log.Log.Infof("Libvirt event %d with reason %d received", event.Event, event.Detail)
		// TODO don't make a client every single time
		client, err := NewDomainEventClient(virtShareDir)
		if err != nil {
			log.Log.Reason(err).Error("Unable to create domain event notify client")
			return
		}

		libvirtEventCallback(d, event, client, deleteNotificationSent)
		log.Log.Info("processed event")
	}
	err := domainConn.DomainEventLifecycleRegister(entrypointCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event callback with libvirt")
		return err
	}
	log.Log.Infof("Registered libvirt event notify callback")
	return nil
}
