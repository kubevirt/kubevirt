package common

import (
	"encoding/json"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type DomainEventRequestInterface interface {
	GetDomainJSON() []byte
	GetStatusJSON() []byte
	GetEventType() string
}

type KubernetesEventRequestInterface interface {
	GetEventJSON() []byte
}

type KubernetesEventRecorderInterface interface {
	Record(request KubernetesEventRequestInterface) error
}

func EnqueueHandlerDomainEvent(eventChan chan watch.Event, request DomainEventRequestInterface) error {
	domain := &api.Domain{}
	status := &metav1.Status{}

	if len(request.GetDomainJSON()) > 0 {
		err := json.Unmarshal(request.GetDomainJSON(), domain)
		if err != nil {
			return fmt.Errorf("Failed to unmarshal domain json object : %v", err)
		}
	}
	if len(request.GetStatusJSON()) > 0 {
		err := json.Unmarshal(request.GetStatusJSON(), status)
		if err != nil {
			return fmt.Errorf("Failed to unmarshal status json object : %v", err)
		}
	}

	log.Log.Object(domain).Infof("Received Domain Event of type %s", request.GetEventType())
	switch request.GetEventType() {
	case string(watch.Added):
		eventChan <- watch.Event{Type: watch.Added, Object: domain}
	case string(watch.Modified):
		eventChan <- watch.Event{Type: watch.Modified, Object: domain}
	case string(watch.Deleted):
		eventChan <- watch.Event{Type: watch.Deleted, Object: domain}
	case string(watch.Error):
		eventChan <- watch.Event{Type: watch.Error, Object: status}
	}
	return nil
}

type EventRecorder struct {
	recorder       record.EventRecorder
	vmiSourceStore cache.Store
	vmiTargetStore cache.Store
}

func (r *EventRecorder) Record(request KubernetesEventRequestInterface) error {

	// unmarshal k8s event
	var event k8sv1.Event
	err := json.Unmarshal(request.GetEventJSON(), &event)
	if err != nil {
		return fmt.Errorf("Error unmarshalling k8s event: %v", err)
	}

	// get vmi and record event
	vmi, err := r.getVMI(event.InvolvedObject)
	if err != nil {
		return err
	}
	r.recorder.Event(vmi, event.Type, event.Reason, event.Message)
	return nil
}

func (r *EventRecorder) getVMI(involvedObj k8sv1.ObjectReference) (*v1.VirtualMachineInstance, error) {
	key := involvedObj.Namespace + "/" + involvedObj.Name
	if obj, exists, err := r.vmiSourceStore.GetByKey(key); err != nil {
		return nil, fmt.Errorf("Error getting VMI: %v", err)
	} else if exists && obj.(*v1.VirtualMachineInstance).UID != involvedObj.UID {
		return nil, fmt.Errorf("VMI %s not found", involvedObj.Name)
	} else if exists {
		return obj.(*v1.VirtualMachineInstance), nil
	}
	if obj, exists, err := r.vmiTargetStore.GetByKey(key); err != nil {
		return nil, fmt.Errorf("Error getting VMI: %v", err)
	} else if exists && obj.(*v1.VirtualMachineInstance).UID != involvedObj.UID {
		return nil, fmt.Errorf("VMI %s not found", involvedObj.Name)
	} else if exists {
		return obj.(*v1.VirtualMachineInstance), nil
	}
	return nil, fmt.Errorf("VMI %s not found", involvedObj.Name)
}

func NewEventRecorder(recorder record.EventRecorder, vmiSourceStore cache.Store, vmiTargetStore cache.Store) *EventRecorder {
	return &EventRecorder{
		recorder:       recorder,
		vmiSourceStore: vmiSourceStore,
		vmiTargetStore: vmiTargetStore,
	}
}
