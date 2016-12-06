package cache

import (
	"encoding/xml"
	"github.com/rgbkrk/libvirt-go"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/types"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/tools/cache"
	kubevirt "kubevirt.io/core/pkg/libvirt"
)

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, namespace and field selector.
func NewListWatchFromClient(c libvirt.VirConnection, events ...int) *cache.ListWatch {
	if len(events) == 0 {
		events = []int{libvirt.VIR_DOMAIN_EVENT_ID_LIFECYCLE}
	}
	listFunc := func(options kubeapi.ListOptions) (runtime.Object, error) {
		doms, err := c.ListAllDomains(libvirt.VIR_CONNECT_LIST_DOMAINS_ACTIVE | libvirt.VIR_CONNECT_LIST_DOMAINS_INACTIVE)
		if err != nil {
			return nil, err
		}
		list := kubevirt.DomainList{
			Items: []kubevirt.Domain{},
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
			domain.Status.Status = kubevirt.LifeCycleTranslationMap[status[0]]
			list.Items = append(list.Items, *domain)
		}

		return &list, nil
	}
	watchFunc := func(options kubeapi.ListOptions) (watch.Interface, error) {
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
				// TODO proper logging
				// FIXME like described below libvirt needs to send the xml along side with the event. When this is done we can return an error
				// watcher.C <- watch.Event{Type: watch.Error, Object: &Domain{}}
				return 0
			}
			// TODO In case of other events, it might not be enough to just send state and domainxml, maybe we have to embed the event and the details too
			//      Think about device removal: First event is a DEFINED/UPDATED event and then we get the REMOVED event when it is done (is it that way?)
			e, ok := eventDetails.(libvirt.DomainLifecycleEvent)
			if ok {
				switch e.Event {

				case libvirt.VIR_DOMAIN_EVENT_STOPPED,
					libvirt.VIR_DOMAIN_EVENT_SHUTDOWN,
					libvirt.VIR_DOMAIN_EVENT_CRASHED,
					libvirt.VIR_DOMAIN_EVENT_UNDEFINED:
					// We can't count on a domain xml in these cases, but let's try it
					if e.Event != libvirt.VIR_DOMAIN_EVENT_UNDEFINED {
						spec, err := NewDomainSpec(d)
						if err != nil {
							// TODO proper logging
						} else {
							domain.Spec = *spec
						}
					}
					status, err := d.GetState()
					if err != nil {
						// TODO proper logging
						domain.Status.Status = kubevirt.NoState
					} else {
						domain.Status.Status = kubevirt.LifeCycleTranslationMap[status[0]]
					}
				default:
					// TODO libvirt is racy there, between an event and fetching a domain xml everything can happen
					//      that is why we can't just report an error here, on the next resync we can compensate this missed event
					// TODO Fix libvirt regarding to event order and domain xml availability. Sometimes when a VM is defined the domainxml can't yet be fetched
					//      Libvirt is inherently inconsistent, see https://www.redhat.com/archives/libvir-list/2016-November/msg01318.html
					//      To fix this, an event should send its state and the current domain xml
					spec, err := NewDomainSpec(d)
					if err != nil {
						// TODO proper logging
						return 0
					}
					domain.Spec = *spec
					status, err := d.GetState()
					if err != nil {
						// TODO proper logging
						return 0
					}
					domain.Status.Status = kubevirt.LifeCycleTranslationMap[status[0]]
				}

				switch e.Event {
				case libvirt.VIR_DOMAIN_EVENT_DEFINED:
					if e.Detail == libvirt.VIR_DOMAIN_EVENT_DEFINED_ADDED {
						watcher.C <- watch.Event{Type: watch.Added, Object: domain}
					} else {
						watcher.C <- watch.Event{Type: watch.Modified, Object: domain}
					}
				case libvirt.VIR_DOMAIN_EVENT_UNDEFINED:
					watcher.C <- watch.Event{Type: watch.Deleted, Object: domain}
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

func NewDomainSpec(dom *libvirt.VirDomain) (*kubevirt.DomainSpec, error) {
	domain := kubevirt.DomainSpec{}
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

func NewDomain(dom *libvirt.VirDomain) (*kubevirt.Domain, error) {
	name, err := dom.GetName()
	if err != nil {
		return nil, err
	}
	uuid, err := dom.GetUUIDString()
	if err != nil {
		return nil, err
	}
	return &kubevirt.Domain{
		Spec: kubevirt.DomainSpec{},
		ObjectMeta: kubeapi.ObjectMeta{
			Name:      name,
			UID:       types.UID(uuid),
			Namespace: kubeapi.NamespaceDefault,
		},
		Status: kubevirt.DomainStatus{},
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "1.2.2",
			Kind:       "domains",
		},
	}, nil
}

func NewDomainCache(c libvirt.VirConnection) (cache.SharedInformer, error) {
	domainCacheSource := NewListWatchFromClient(c, libvirt.VIR_DOMAIN_EVENT_ID_LIFECYCLE)
	informer := cache.NewSharedInformer(domainCacheSource, &kubevirt.Domain{}, 0)
	return informer, nil
}
