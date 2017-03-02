package cache

import (
	"encoding/xml"
	"github.com/libvirt/libvirt-go"
	kubeapi "k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, namespace and field selector.
func newListWatchFromClient(c virtwrap.Connection, events ...int) *cache.ListWatch {
	listFunc := func(options kubev1.ListOptions) (runtime.Object, error) {
		doms, err := c.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
		if err != nil {
			return nil, err
		}
		list := virtwrap.DomainList{
			Items: []virtwrap.Domain{},
		}
		for _, dom := range doms {
			domain, err := NewDomain(dom)
			if err != nil {
				return nil, err
			}
			spec, err := NewDomainSpec(dom)
			if err != nil {
				return nil, err
			}
			domain.Spec = *spec
			status, reason, err := dom.GetState()
			if err != nil {
				return nil, err
			}
			domain.SetState(status, reason)
			list.Items = append(list.Items, *domain)
		}

		return &list, nil
	}
	watchFunc := func(options kubev1.ListOptions) (watch.Interface, error) {
		return newDomainWatcher(c, events...)
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

func newDomainWatcher(c virtwrap.Connection, events ...int) (watch.Interface, error) {
	watcher := &DomainWatcher{C: make(chan watch.Event)}
	callback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
		logging.DefaultLogger().Info().V(3).Msgf("Libvirt event %d with reason %d received", event.Event, event.Detail)
		callback(d, event, watcher)
	}
	_, err := c.DomainEventLifecycleRegister(nil, callback)
	if err != nil {
		logging.DefaultLogger().Info().V(2).Msg("Lifecycle event callback registered.")
	}
	return watcher, err
}

func NewDomainSpec(dom virtwrap.VirDomain) (*virtwrap.DomainSpec, error) {
	domain := virtwrap.DomainSpec{}
	domxml, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal([]byte(domxml), &domain)
	if err != nil {
		return nil, err
	}

	return &domain, nil
}

func NewDomain(dom virtwrap.VirDomain) (*virtwrap.Domain, error) {

	name, err := dom.GetName()
	if err != nil {
		return nil, err
	}
	uuid, err := dom.GetUUIDString()
	if err != nil {
		return nil, err
	}
	return &virtwrap.Domain{
		Spec: virtwrap.DomainSpec{},
		ObjectMeta: kubev1.ObjectMeta{
			Name:      name,
			UID:       types.UID(uuid),
			Namespace: kubeapi.NamespaceDefault,
		},
		Status: virtwrap.DomainStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "1.2.2",
			Kind:       "Domain",
		},
	}, nil
}

func NewSharedInformer(c virtwrap.Connection) (cache.SharedInformer, error) {
	lw := newListWatchFromClient(c)
	informer := cache.NewSharedInformer(lw, &virtwrap.Domain{}, 0)
	return informer, nil
}

func callback(d virtwrap.VirDomain, event *libvirt.DomainEventLifecycle, watcher *DomainWatcher) {
	domain, err := NewDomain(d)
	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("Could not create the Domain.")
		return
	}
	// TODO In case of other events, it might not be enough to just send state and domainxml, maybe we have to embed the event and the details too
	//      Think about device removal: First event is a DEFINED/UPDATED event and then we get the REMOVED event when it is done (is it that way?)
	switch event.Event {

	case libvirt.DOMAIN_EVENT_STOPPED,
		libvirt.DOMAIN_EVENT_SHUTDOWN,
		libvirt.DOMAIN_EVENT_CRASHED,
		libvirt.DOMAIN_EVENT_UNDEFINED:
		// We can't count on a domain xml in these cases, but let's try it
		if event.Event != libvirt.DOMAIN_EVENT_UNDEFINED {
			spec, err := NewDomainSpec(d)
			if err != nil {

				if err.(libvirt.Error).Code != libvirt.ERR_NO_DOMAIN {
					logging.DefaultLogger().Error().Reason(err).Msg("Could not fetch the Domain specification.")
				}
			} else {
				domain.Spec = *spec
			}
		}
		status, reason, err := d.GetState()
		if err != nil {

			if err.(libvirt.Error).Code != libvirt.ERR_NO_DOMAIN {
				logging.DefaultLogger().Error().Reason(err).Msg("Could not fetch the Domain state.")
			}
			domain.SetState(libvirt.DOMAIN_NOSTATE, reason)
		} else {
			domain.SetState(status, reason)
		}
	default:
		spec, err := NewDomainSpec(d)
		if err != nil {
			logging.DefaultLogger().Error().Reason(err).Msg("Could not fetch the Domain specification.")
			return
		}
		domain.Spec = *spec
		status, reason, err := d.GetState()
		if err != nil {
			logging.DefaultLogger().Error().Reason(err).Msg("Could not fetch the Domain state.")
			return
		}
		domain.SetState(status, reason)
	}

	switch event.Event {
	case libvirt.DOMAIN_EVENT_DEFINED:
		if libvirt.DomainEventDefinedDetailType(event.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
			watcher.C <- watch.Event{Type: watch.Added, Object: domain}
		} else {
			watcher.C <- watch.Event{Type: watch.Modified, Object: domain}
		}
	case libvirt.DOMAIN_EVENT_UNDEFINED:
		watcher.C <- watch.Event{Type: watch.Deleted, Object: domain}
	default:
		watcher.C <- watch.Event{Type: watch.Modified, Object: domain}
	}

}
