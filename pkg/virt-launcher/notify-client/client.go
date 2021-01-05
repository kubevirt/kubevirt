package eventsclient

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	libvirt "libvirt.org/libvirt-go"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/reference"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	com "kubevirt.io/kubevirt/pkg/handler-launcher-com"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var (
	// add older version when supported
	// don't use the variable in pkg/handler-launcher-com/notify/v1/version.go in order to detect version mismatches early
	supportedNotifyVersions = []uint32{1}
)

type Notifier struct {
	v1client         notifyv1.NotifyClient
	conn             *grpc.ClientConn
	connLock         sync.Mutex
	pipeSocketPath   string
	legacySocketPath string
}

type libvirtEvent struct {
	Domain     string
	Event      *libvirt.DomainEventLifecycle
	AgentEvent *libvirt.DomainEventAgentLifecycle
}

func NewNotifier(virtShareDir string) *Notifier {
	return &Notifier{
		pipeSocketPath:   filepath.Join(virtShareDir, "domain-notify-pipe.sock"),
		legacySocketPath: filepath.Join(virtShareDir, "domain-notify.sock"),
	}
}

func negotiateVersion(infoClient info.NotifyInfoClient) (uint32, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	info, err := infoClient.Info(ctx, &info.NotifyInfoRequest{})
	if err != nil {
		return 0, fmt.Errorf("could not check cmd server version: %v", err)
	}
	version, err := com.GetHighestCompatibleVersion(info.SupportedNotifyVersions, supportedNotifyVersions)
	if err != nil {
		return 0, err
	}

	switch version {
	case 1:
		// fall-through for all supported versions
	default:
		return 0, fmt.Errorf("cmd v1client version %v not implemented yet", version)
	}

	return version, nil
}

func (n *Notifier) detectSocketPath() string {

	// use the legacy domain socket if it exists. This would
	// occur if the vmi was started with a hostPath shared mount
	// using our old method for virt-handler to virt-launcher communication
	exists, _ := diskutils.FileExists(n.legacySocketPath)
	if exists {
		return n.legacySocketPath
	}

	// default to using the new pipe socket
	return n.pipeSocketPath
}

func (n *Notifier) connect() error {
	connectionInterval := 2 * time.Second
	connectionTimeout := 20 * time.Second

	err := utilwait.PollImmediate(connectionInterval, connectionTimeout, func() (done bool, err error) {
		n.connLock.Lock()
		defer n.connLock.Unlock()
		if n.conn != nil {
			// already connected
			return true, nil
		}

		socketPath := n.detectSocketPath()

		// dial socket
		conn, err := grpcutil.DialSocketWithTimeout(socketPath, 5)
		if err != nil {
			log.Log.Reason(err).Infof("failed to dial notify socket: %s", socketPath)
			return false, nil
		}

		version, err := negotiateVersion(info.NewNotifyInfoClient(conn))
		if err != nil {
			log.Log.Reason(err).Infof("failed to negotiate version")
			conn.Close()
			return false, nil
		}

		// create cmd v1client
		switch version {
		case 1:
			client := notifyv1.NewNotifyClient(conn)
			n.v1client = client
			n.conn = conn
		default:
			conn.Close()
			return false, fmt.Errorf("cmd v1client version %v not implemented yet", version)
		}

		log.Log.Infof("Successfully connected to domain notify socket at %s", socketPath)
		return true, nil
	})

	return err
}

func newV1Notifier(client notifyv1.NotifyClient, conn *grpc.ClientConn) *Notifier {
	return &Notifier{
		v1client: client,
		conn:     conn,
	}
}

func (n *Notifier) SendDomainEvent(event watch.Event) error {

	var domainJSON []byte
	var statusJSON []byte
	var err error

	err = n.connect()
	if err != nil {
		return err
	}

	if event.Type == watch.Error {
		status := event.Object.(*metav1.Status)
		statusJSON, err = json.Marshal(status)
		if err != nil {
			log.Log.Reason(err).Infof("JSON marshal of notify ERROR event failed")
			return err
		}
	} else {
		domain := event.Object.(*api.Domain)
		domainJSON, err = json.Marshal(domain)
		if err != nil {
			log.Log.Reason(err).Infof("JSON marshal of notify event failed")
			return err
		}
	}
	request := notifyv1.DomainEventRequest{
		DomainJSON: domainJSON,
		StatusJSON: statusJSON,
		EventType:  string(event.Type),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := n.v1client.HandleDomainEvent(ctx, &request)

	if err != nil {
		log.Log.Reason(err).Infof("Failed to send domain notify event")
		return err
	} else if response.Success != true {
		msg := fmt.Sprintf("failed to notify domain event: %s", response.Message)
		return fmt.Errorf(msg)
	}

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
		now := metav1.Now()
		domain.ObjectMeta.DeletionTimestamp = &now
		watchEvent := watch.Event{Type: watch.Modified, Object: domain}
		client.SendDomainEvent(watchEvent)
		updateEvents(watchEvent, domain, events)
	default:
		if libvirtEvent.Event != nil {
			if libvirtEvent.Event.Event == libvirt.DOMAIN_EVENT_DEFINED && libvirt.DomainEventDefinedDetailType(libvirtEvent.Event.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
				event := watch.Event{Type: watch.Added, Object: domain}
				client.SendDomainEvent(event)
				updateEvents(event, domain, events)
			} else if libvirtEvent.Event.Event == libvirt.DOMAIN_EVENT_STARTED && libvirt.DomainEventStartedDetailType(libvirtEvent.Event.Detail) == libvirt.DOMAIN_EVENT_STARTED_MIGRATED {
				event := watch.Event{Type: watch.Added, Object: domain}
				client.SendDomainEvent(event)
				updateEvents(event, domain, events)
			}
		}
		if interfaceStatus != nil {
			domain.Status.Interfaces = interfaceStatus
		}
		if osInfo != nil {
			domain.Status.OSInfo = *osInfo
		}

		err := client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})
		if err != nil {
			log.Log.Reason(err).Error("Could not send domain notify event.")
		}
	}
}

var updateEvents = updateEventsClosure()

func updateEventsClosure() func(event watch.Event, domain *api.Domain, events chan watch.Event) {
	firstAddEvent := true
	firstDeleteEvent := true

	return func(event watch.Event, domain *api.Domain, events chan watch.Event) {
		if event.Type == watch.Added && firstAddEvent {
			firstAddEvent = false
			events <- event
		} else if event.Type == watch.Modified && domain.ObjectMeta.DeletionTimestamp != nil && firstDeleteEvent {
			firstDeleteEvent = false
			events <- event
		}
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
				if interfaceStatuses != nil {
					interfaceStatuses = agentpoller.MergeAgentStatusesWithDomainData(domainCache.Spec.Devices.Interfaces, interfaceStatuses)
				}

				eventCallback(domainConn, domainCache, libvirtEvent{}, n, deleteNotificationSent,
					interfaceStatuses, guestOsInfo)
			case <-reconnectChan:
				n.SendDomainEvent(newWatchEventError(fmt.Errorf("Libvirt reconnect, domain %s", domainName)))
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

func (n *Notifier) SendK8sEvent(vmi *v1.VirtualMachineInstance, severity string, reason string, message string) error {

	err := n.connect()
	if err != nil {
		return err
	}

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

	request := notifyv1.K8SEventRequest{
		EventJSON: json,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := n.v1client.HandleK8SEvent(ctx, &request)

	if err != nil {
		return err
	} else if response.Success != true {
		msg := fmt.Sprintf("failed to notify k8s event: %s", response.Message)
		return fmt.Errorf(msg)
	}

	return nil
}

func (n *Notifier) Close() {
	n.connLock.Lock()
	defer n.connLock.Unlock()

	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}
}
