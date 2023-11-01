package eventsclient

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	"libvirt.org/go/libvirt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/reference"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	com "kubevirt.io/kubevirt/pkg/handler-launcher-com"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const (
	cantDetermineLibvirtDomainName = "Could not determine name of libvirt domain in event callback."
	libvirtEventChannelFull        = "Libvirt event channel is full, dropping event."
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

	intervalTimeout time.Duration
	sendTimeout     time.Duration
	totalTimeout    time.Duration
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
		intervalTimeout:  defaultIntervalTimeout,
		sendTimeout:      defaultSendTimeout,
		totalTimeout:     defaultTotalTimeout,
	}
}

var (
	defaultIntervalTimeout = 1 * time.Second
	defaultSendTimeout     = 5 * time.Second
	defaultTotalTimeout    = 20 * time.Second
)

var (
	schemeBuilder = runtime.NewSchemeBuilder(v1.AddKnownTypesGenerator(v1.GroupVersions))
	addToScheme   = schemeBuilder.AddToScheme
	scheme        = runtime.NewScheme()
)

func init() {
	addToScheme(scheme)
}

func negotiateVersion(infoClient info.NotifyInfoClient) (uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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

// used by unit tests
func (n *Notifier) SetCustomTimeouts(interval, send, total time.Duration) {
	n.intervalTimeout = interval
	n.sendTimeout = send
	n.totalTimeout = total

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
	if n.conn != nil {
		// already connected
		return nil
	}

	socketPath := n.detectSocketPath()

	// dial socket
	conn, err := grpcutil.DialSocketWithTimeout(socketPath, 5)
	if err != nil {
		log.Log.Reason(err).Infof("failed to dial notify socket: %s", socketPath)
		return err
	}

	version, err := negotiateVersion(info.NewNotifyInfoClient(conn))
	if err != nil {
		log.Log.Reason(err).Infof("failed to negotiate version")
		conn.Close()
		return err
	}

	// create cmd v1client
	switch version {
	case 1:
		client := notifyv1.NewNotifyClient(conn)
		n.v1client = client
		n.conn = conn
	default:
		conn.Close()
		return fmt.Errorf("cmd v1client version %v not implemented yet", version)
	}

	log.Log.Infof("Successfully connected to domain notify socket at %s", socketPath)
	return nil
}

func (n *Notifier) SendDomainEvent(event watch.Event) error {

	var domainJSON []byte
	var statusJSON []byte
	var err error

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

	var response *notifyv1.Response
	err = utilwait.PollImmediate(n.intervalTimeout, n.totalTimeout, func() (done bool, err error) {
		n.connLock.Lock()
		defer n.connLock.Unlock()

		err = n.connect()
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to connect to notify server")
			return false, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), n.sendTimeout)
		defer cancel()
		response, err = n.v1client.HandleDomainEvent(ctx, &request)

		if err != nil {
			log.Log.Reason(err).Errorf("Failed to send domain notify event. closing connection.")
			n._close()
			return false, nil
		}

		return true, nil

	})

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
	interfaceStatus []api.InterfaceStatus, osInfo *api.GuestOSInfo, vmi *v1.VirtualMachineInstance, fsFreezeStatus *api.FSFreeze,
	metadataCache *metadata.Cache) {

	d, err := c.LookupDomainByName(util.DomainFromNamespaceName(domain.ObjectMeta.Namespace, domain.ObjectMeta.Name))
	if err != nil {
		if !domainerrors.IsNotFound(err) {
			log.Log.Reason(err).Error("Could not fetch the Domain.")
			return
		}
		domain.SetState(api.NoState, api.ReasonNonExistent)
	} else {
		defer d.Free()

		// Remember current status before it will be changed.
		var (
			prevStatus = domain.Status.Status
			prevReason = domain.Status.Reason
		)

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

		kubevirtMetadata := metadata.LoadKubevirtMetadata(metadataCache)
		spec, err := util.GetDomainSpecWithRuntimeInfo(d)
		if err != nil {
			// NOTE: Getting domain metadata for a live-migrating VM isn't allowed
			if !domainerrors.IsNotFound(err) && !domainerrors.IsInvalidOperation(err) {
				log.Log.Reason(err).Error("Could not fetch the Domain specification.")
				return
			}
		} else {
			domain.ObjectMeta.UID = kubevirtMetadata.UID
		}

		if spec != nil {
			spec.Metadata.KubeVirt = kubevirtMetadata
			domain.Spec = *spec
		}

		if domain.Status.Status == prevStatus && domain.Status.Reason == prevReason {
			// Status hasn't changed so log only in higher verbosity.
			log.Log.V(3).Infof("kubevirt domain status: %v(%v):%v(%v)", domain.Status.Status, status, domain.Status.Reason, reason)
		} else {
			log.Log.Infof("kubevirt domain status: %v(%v):%v(%v)", domain.Status.Status, status, domain.Status.Reason, reason)
		}
	}

	switch domain.Status.Reason {
	case api.ReasonNonExistent:
		now := metav1.Now()
		domain.ObjectMeta.DeletionTimestamp = &now
		watchEvent := watch.Event{Type: watch.Modified, Object: domain}
		client.SendDomainEvent(watchEvent)
		updateEvents(watchEvent, domain, events)
	case api.ReasonPausedIOError:
		domainDisksWithErrors, err := d.GetDiskErrors(0)
		if err != nil {
			log.Log.Reason(err).Error("Could not get disks with errors")
		}
		for _, disk := range domainDisksWithErrors {
			volumeName := converter.GetVolumeNameByTarget(domain, disk.Disk)
			var reasonError string
			switch disk.Error {
			case libvirt.DOMAIN_DISK_ERROR_NONE:
				continue
			case libvirt.DOMAIN_DISK_ERROR_UNSPEC:
				reasonError = fmt.Sprintf("VM Paused due to IO error at the volume: %s", volumeName)
			case libvirt.DOMAIN_DISK_ERROR_NO_SPACE:
				reasonError = fmt.Sprintf("VM Paused due to not enough space on volume: %s", volumeName)
			}
			err = client.SendK8sEvent(vmi, "Warning", "IOerror", reasonError)
			if err != nil {
				log.Log.Reason(err).Error(fmt.Sprintf("Could not send k8s event"))
			}
			event := watch.Event{Type: watch.Modified, Object: domain}
			client.SendDomainEvent(event)
			updateEvents(event, domain, events)
		}
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

		if fsFreezeStatus != nil {
			domain.Status.FSFreezeStatus = *fsFreezeStatus
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
	vmi *v1.VirtualMachineInstance,
	domainName string,
	agentStore *agentpoller.AsyncAgentStore,
	qemuAgentSysInterval time.Duration,
	qemuAgentFileInterval time.Duration,
	qemuAgentUserInterval time.Duration,
	qemuAgentVersionInterval time.Duration,
	qemuAgentFSFreezeStatusInterval time.Duration,
	metadataCache *metadata.Cache,
) error {

	eventChan := make(chan libvirtEvent, 10)

	reconnectChan := make(chan bool, 10)

	var domainCache *api.Domain

	domainConn.SetReconnectChan(reconnectChan)

	agentPoller := agentpoller.CreatePoller(
		domainConn,
		vmi.UID,
		domainName,
		agentStore,
		qemuAgentSysInterval,
		qemuAgentFileInterval,
		qemuAgentUserInterval,
		qemuAgentVersionInterval,
		qemuAgentFSFreezeStatusInterval,
	)

	// Run the event process logic in a separate go-routine to not block libvirt
	go func() {
		var interfaceStatuses []api.InterfaceStatus
		var guestOsInfo *api.GuestOSInfo
		var fsFreezeStatus *api.FSFreeze
		for {
			select {
			case event := <-eventChan:
				metadataCache.ResetNotification()
				domainCache = util.NewDomainFromName(event.Domain, vmi.UID)
				eventCallback(domainConn, domainCache, event, n, deleteNotificationSent, interfaceStatuses, guestOsInfo, vmi, fsFreezeStatus, metadataCache)
				log.Log.Infof("Domain name event: %v", domainCache.Spec.Name)
				if event.AgentEvent != nil {
					if event.AgentEvent.State == libvirt.CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_CONNECTED {
						agentPoller.Start()
					} else if event.AgentEvent.State == libvirt.CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_DISCONNECTED {
						agentPoller.Stop()
					}
				}
			case agentUpdate := <-agentStore.AgentUpdated:
				metadataCache.ResetNotification()
				interfaceStatuses = agentUpdate.DomainInfo.Interfaces
				guestOsInfo = agentUpdate.DomainInfo.OSInfo
				fsFreezeStatus = agentUpdate.DomainInfo.FSFreezeStatus

				eventCallback(domainConn, domainCache, libvirtEvent{}, n, deleteNotificationSent,
					interfaceStatuses, guestOsInfo, vmi, fsFreezeStatus, metadataCache)
			case <-reconnectChan:
				n.SendDomainEvent(newWatchEventError(fmt.Errorf("Libvirt reconnect, domain %s", domainName)))

			case <-metadataCache.Listen():
				// Metadata cache updates should be processed only *after* at least one
				// libvirt event arrived (which creates the first domainCache).
				if domainCache != nil {
					domainCache = util.NewDomainFromName(
						util.DomainFromNamespaceName(domainCache.ObjectMeta.Namespace, domainCache.ObjectMeta.Name),
						vmi.UID,
					)
					eventCallback(
						domainConn,
						domainCache,
						libvirtEvent{},
						n,
						deleteNotificationSent,
						interfaceStatuses,
						guestOsInfo,
						vmi,
						fsFreezeStatus,
						metadataCache,
					)
				}
			}
		}
	}()

	domainEventLifecycleCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {

		log.Log.Infof("DomainLifecycle event %s with event id %d reason %d received", event.String(), event.Event, event.Detail)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}
		select {
		case eventChan <- libvirtEvent{Event: event, Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}

	domainEventDeviceAddedCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventDeviceAdded) {
		log.Log.Infof("Domain Device Added event received")
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}
		select {
		case eventChan <- libvirtEvent{Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}

	domainEventDeviceRemovedCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventDeviceRemoved) {
		log.Log.Infof("Domain Device Removed event received")
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}

		select {
		case eventChan <- libvirtEvent{Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}

	domainEventMemoryDeviceSizeChange := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventMemoryDeviceSizeChange) {
		log.Log.Infof("Domain Memory Device size-change event received")
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}

		select {
		case eventChan <- libvirtEvent{Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}

	err := domainConn.DomainEventLifecycleRegister(domainEventLifecycleCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event callback with libvirt")
		return err
	}

	err = domainConn.DomainEventDeviceAddedRegister(domainEventDeviceAddedCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register device added event callback with libvirt")
		return err
	}
	err = domainConn.DomainEventDeviceRemovedRegister(domainEventDeviceRemovedCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register device removed event callback with libvirt")
		return err
	}
	err = domainConn.DomainEventMemoryDeviceSizeChangeRegister(domainEventMemoryDeviceSizeChange)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register memory device size change event callback with libvirt")
		return err
	}

	agentEventLifecycleCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventAgentLifecycle) {
		log.Log.Infof("GuestAgentLifecycle event state %d with reason %d received", event.State, event.Reason)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}
		select {
		case eventChan <- libvirtEvent{AgentEvent: event, Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
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
	vmiRef, err := reference.GetReference(scheme, vmi)
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

	var response *notifyv1.Response
	err = utilwait.PollImmediate(n.intervalTimeout, n.totalTimeout, func() (done bool, err error) {
		n.connLock.Lock()
		defer n.connLock.Unlock()

		err = n.connect()
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to connect to notify server")
			return false, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), n.sendTimeout)
		defer cancel()
		response, err = n.v1client.HandleK8SEvent(ctx, &request)

		if err != nil {
			log.Log.Reason(err).Errorf("Failed to send k8s notify event. closing connection.")
			n._close()
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return err
	} else if response.Success != true {
		msg := fmt.Sprintf("failed to notify k8s event: %s", response.Message)
		return fmt.Errorf(msg)
	}

	return nil
}

func (n *Notifier) _close() {
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}
}

func (n *Notifier) Close() {
	n.connLock.Lock()
	defer n.connLock.Unlock()
	n._close()

}
