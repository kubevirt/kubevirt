package eventsclient

import (
	"context"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/reference"

	v1 "kubevirt.io/api/core/v1"
)

var (
	// add older version when supported
	// don't use the variable in pkg/handler-launcher-com/notify/v1/version.go in order to detect version mismatches early
	supportedNotifyVersions = []uint32{1}
)

type Notifier struct {
	client *notifyClient
}

func NewNotifier(virtShareDir string) *Notifier {
	return &Notifier{client: newGRPCNotifyClient(virtShareDir)}
}

var (
	schemeBuilder = runtime.NewSchemeBuilder(v1.AddKnownTypesGenerator(v1.GroupVersions))
	addToScheme   = schemeBuilder.AddToScheme
	scheme        = runtime.NewScheme()
)

func init() {
	addToScheme(scheme)
}

// used by unit tests
func (n *Notifier) SetCustomTimeouts(interval, send, total time.Duration) {
	n.client.intervalTimeout = interval
	n.client.sendTimeout = send
	n.client.totalTimeout = total

}

func (n *Notifier) SendDomainEvent(ctx context.Context, event watch.Event) error {
	return n.client.HandleDomainEvent(ctx, &event)
}

//nolint:dupl
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

	return n.client.HandleK8SEvent(&event)
}

func (n *Notifier) Close() {
	n.client.Close()
}

type notifierClient interface {
	HandleDomainEvent(context.Context, *watch.Event) error
	HandleK8SEvent(*k8sv1.Event) error
	Close()
}
