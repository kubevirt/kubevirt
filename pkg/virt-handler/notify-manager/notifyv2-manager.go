package notifymanager

import (
	"encoding/json"
	"errors"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-handler/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var notImplemented error = errors.New("Not implemented")

func NewNotifyV2Manager(directChan chan<- watch.Event,
	recorder record.EventRecorder, podIsolationDetector isolation.PodIsolationDetector) *notifyV2Manager {
	return &notifyV2Manager{
		directChan:           directChan,
		recorder:             recorder,
		podIsolationDetector: podIsolationDetector,
	}
}

type notifyV2Manager struct {
	podIsolationDetector isolation.PodIsolationDetector
	directChan           chan<- watch.Event
	recorder             record.EventRecorder
}

func (m *notifyV2Manager) StartDomainNotify(domainPipeStopChan <-chan struct{}, vmi *v1.VirtualMachineInstance) error {
	res, err := m.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}

	root, err := res.MountRoot()
	if err != nil {
		return err
	}
	socket, err := safepath.JoinNoFollow(root, cmdclient.SocketOnGuest())
	if err != nil {
		return err
	}

	var client notifyclient.NotifyClient
	err = socket.ExecuteNoFollow(func(safePath string) error {
		client, err = notifyclient.NewClient(safePath)
		return err
	})
	if err != nil {
		return err
	}

	// 	// TODO need to use one ctx
	ctx := contextFromChan(domainPipeStopChan)
	stream, err := client.DomainEvents(ctx)
	if err != nil {
		return err
	}

	processed := make(chan error)
	go func() {
		err := processDomainEvent(stream, m.directChan)
		processed <- err
		close(processed)
	}()

	select {
	case err := <-processed:
		if cmdclient.IsUnimplemented(err) {
			return notImplemented
		}
	case <-time.After(3 * time.Second):
		// todo, better timeout? cancel stream?
	}

	go func(stream notifyclient.DomainEventsStream, directChan chan<- watch.Event) {
		for {
			processDomainEvents(stream, directChan)
			select {
			case <-ctx.Done():
				return
			default:
				newStream, err := client.DomainEvents(ctx)
				if err != nil {
					continue
				}
				stream = newStream
			}
		}
	}(stream, m.directChan)

	go func() {
		for {
			stream, err := client.KubernetesEvents(ctx)
			if err != nil {
				continue
			}
			processKubernetesEvents(stream, m.recorder, vmi)
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return nil
}

func processDomainEvents(stream notifyclient.DomainEventsStream, directChan chan<- watch.Event) {
	for {
		err := processDomainEvent(stream, directChan)
		if err != nil {
			return
		}
	}
}

func processDomainEvent(stream notifyclient.DomainEventsStream, directChan chan<- watch.Event) error {
	response, err := stream.Recv()
	if err != nil {
		return err
	}

	domain := &api.Domain{}
	err = json.Unmarshal(response.Domain, domain)
	if err != nil {
		log.Log.Errorf("Failed to unmarshal domain json object")
		// TODO beh
	}
	log.Log.Object(domain).V(3).Infof("Received Domain Event of type %s", response.EventType)
	switch response.EventType {
	case string(watch.Added):
		directChan <- watch.Event{Type: watch.Added, Object: domain}
	case string(watch.Modified):
		directChan <- watch.Event{Type: watch.Modified, Object: domain}
	case string(watch.Deleted):
		directChan <- watch.Event{Type: watch.Deleted, Object: domain}
	case string(watch.Error):
		// log.Log.Object(domain).Errorf("Domain error event with message: %s", status.Message)
	}
	return nil
}

func processKubernetesEvents(stream notifyclient.KubernetesEventsStream, recorder record.EventRecorder, vmi *v1.VirtualMachineInstance) {
	for {
		response, err := stream.Recv()
		if err != nil {
			// log
			return
		}

		event := k8sv1.Event{
			Message: response.Message,
			Reason:  response.Reason,
			Type:    response.Type,
		}
		recorder.Event(vmi, event.Type, event.Reason, event.Message)
	}
}
