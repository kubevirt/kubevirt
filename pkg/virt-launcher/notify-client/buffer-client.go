package eventsclient

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var _ notifierClient = &client{}

func NewClient() *client {
	return &client{
		domainNotification: make(chan *watch.Event),
		k8sNotification:    make(chan *k8sv1.Event),
	}
}

type client struct {
	domainNotification chan *watch.Event
	k8sNotification    chan *k8sv1.Event
}

func (client *client) Close() {
}

func (client *client) HandleDomainEvent(event *watch.Event) error {
	client.domainNotification <- event
	return nil
}
func (client *client) HandleK8SEvent(event *k8sv1.Event) error {
	client.k8sNotification <- event
	return nil
}

func (client *client) DomainEventChan() <-chan *watch.Event {
	return client.domainNotification
}

func (client *client) K8SEventChan() <-chan *k8sv1.Event {
	return client.k8sNotification
}
