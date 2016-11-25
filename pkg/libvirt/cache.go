package libvirt

import (
	"encoding/xml"
	"github.com/rgbkrk/libvirt-go"
	"k8s.io/client-go/1.5/pkg/api"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/types"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/tools/cache"
)

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, namespace and field selector.
func NewListWatchFromClient(c libvirt.VirConnection, events ...int) *cache.ListWatch {
	if len(events) == 0 {
		events = []int{libvirt.VIR_DOMAIN_EVENT_ID_LIFECYCLE}
	}
	listFunc := func(options api.ListOptions) (runtime.Object, error) {
		doms, err := c.ListAllDomains(libvirt.VIR_CONNECT_LIST_DOMAINS_ACTIVE)
		if err != nil {
			return nil, err
		}
		list := DomainList{
			Items: []Domain{},
		}
		for _, dom := range doms {
			domain, err := NewDomain(&dom)
			if err != nil {
				return nil, err
			}
			spec, err := NewDomainSpec(&dom)
			if err != nil {
				return nil, err
			}
			domain.Spec = *spec
			status, err := dom.GetState()
			if err != nil {
				return nil, err
			}
			domain.Status.Status = LifeCycleTranslationMap[status[0]]
			list.Items = append(list.Items, *domain)
		}

		return &list, nil
	}
	watchFunc := func(options api.ListOptions) (watch.Interface, error) {
		return NewDomainWatcher(c, events...)
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

type DomainWatcher struct {
	C chan watch.Event
}

func (d *DomainWatcher) Stop() {

}

func (d *DomainWatcher) ResultChan() <-chan watch.Event {
	return d.C
}

func NewDomainWatcher(c libvirt.VirConnection, events ...int) (watch.Interface, error) {
	watcher := &DomainWatcher{C: make(chan watch.Event)}
	callback := libvirt.DomainEventCallback(
		func(c *libvirt.VirConnection, d *libvirt.VirDomain, eventDetails interface{}, _ func()) int {
			domain, err := NewDomain(d)
			if err != nil {
				// FIXME like described below libvirt needs to send the xml along side with the event. When this is done we can return an error
				// watcher.C <- watch.Event{Type: watch.Error, Object: &Domain{}}
				return 0
			}
			e, ok := eventDetails.(libvirt.DomainLifecycleEvent)
			if ok {
				switch e.Event {

				case libvirt.VIR_DOMAIN_EVENT_STOPPED,
					libvirt.VIR_DOMAIN_EVENT_SHUTDOWN,
					libvirt.VIR_DOMAIN_EVENT_CRASHED,
					libvirt.VIR_DOMAIN_EVENT_UNDEFINED:
					// We can't count on a domain xml in these cases
					status, err := d.GetState()
					if err != nil {
						domain.Status.Status = NoState
					} else {
						domain.Status.Status = LifeCycleTranslationMap[status[0]]
					}
				default:
					// TODO libvirt is racy there, between an event and fetching a domain xml everything can happen
					//      that is why we can't just report an error here, on the next resync we can compensate this missed event
					// TODO Fix libvirt regarding to event order and domain xml availability. Sometimes when a VM is defined the domainxml can't yet be fetched
					//      Libvirt is inherently inconsistent, see https://www.redhat.com/archives/libvir-list/2016-November/msg01318.html
					//      To fix this, an event should send its state and the current domain xml
					spec, err := NewDomainSpec(d)
					if err != nil {
						return 0
					}
					domain.Spec = *spec
					status, err := d.GetState()
					if err != nil {
						return 0
					}
					domain.Status.Status = LifeCycleTranslationMap[status[0]]
				}

				switch e.Event {
				case libvirt.VIR_DOMAIN_EVENT_STARTED:
					watcher.C <- watch.Event{Type: watch.Added, Object: domain}
				case libvirt.VIR_DOMAIN_EVENT_STOPPED, libvirt.VIR_DOMAIN_EVENT_SHUTDOWN, libvirt.VIR_DOMAIN_EVENT_CRASHED:
					watcher.C <- watch.Event{Type: watch.Deleted, Object: domain}
				case libvirt.VIR_DOMAIN_EVENT_DEFINED, libvirt.VIR_DOMAIN_EVENT_UNDEFINED:
					// kubevirt just cares about active domains, so ignore these events
				default:
					watcher.C <- watch.Event{Type: watch.Modified, Object: domain}
				}
			} else {
				watcher.C <- watch.Event{Type: watch.Modified, Object: domain}
			}
			return 0
		})
	for _, event := range events {
		c.DomainEventRegister(libvirt.VirDomain{}, event, &callback, nil)
	}
	return watcher, nil
}

func NewDomainSpec(dom *libvirt.VirDomain) (*DomainSpec, error) {
	domain := DomainSpec{}
	domxml, err := dom.GetXMLDesc(libvirt.VIR_DOMAIN_XML_MIGRATABLE)
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal([]byte(domxml), &domain)
	if err != nil {
		return nil, err
	}

	return &domain, nil
}

func NewDomain(dom *libvirt.VirDomain) (*Domain, error) {
	name, err := dom.GetName()
	if err != nil {
		return nil, err
	}
	uuid, err := dom.GetUUIDString()
	if err != nil {
		return nil, err
	}
	return &Domain{
		Spec: DomainSpec{},
		ObjectMeta: kubeapi.ObjectMeta{
			Name:      name,
			UID:       types.UID(uuid),
			Namespace: kubeapi.NamespaceDefault,
		},
		Status: DomainStatus{},
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "1.2.2",
			Kind:       "domains",
		},
	}, nil
}
