/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package cache

import (
	"encoding/xml"
	"strings"

	"github.com/libvirt/libvirt-go"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
)

var LifeCycleTranslationMap = map[libvirt.DomainState]api.LifeCycle{
	libvirt.DOMAIN_NOSTATE:     api.NoState,
	libvirt.DOMAIN_RUNNING:     api.Running,
	libvirt.DOMAIN_BLOCKED:     api.Blocked,
	libvirt.DOMAIN_PAUSED:      api.Paused,
	libvirt.DOMAIN_SHUTDOWN:    api.Shutdown,
	libvirt.DOMAIN_SHUTOFF:     api.Shutoff,
	libvirt.DOMAIN_CRASHED:     api.Crashed,
	libvirt.DOMAIN_PMSUSPENDED: api.PMSuspended,
}

var ShutdownReasonTranslationMap = map[libvirt.DomainShutdownReason]api.StateChangeReason{
	libvirt.DOMAIN_SHUTDOWN_UNKNOWN: api.ReasonUnknown,
	libvirt.DOMAIN_SHUTDOWN_USER:    api.ReasonUser,
}

var ShutoffReasonTranslationMap = map[libvirt.DomainShutoffReason]api.StateChangeReason{
	libvirt.DOMAIN_SHUTOFF_UNKNOWN:       api.ReasonUnknown,
	libvirt.DOMAIN_SHUTOFF_SHUTDOWN:      api.ReasonShutdown,
	libvirt.DOMAIN_SHUTOFF_DESTROYED:     api.ReasonDestroyed,
	libvirt.DOMAIN_SHUTOFF_CRASHED:       api.ReasonCrashed,
	libvirt.DOMAIN_SHUTOFF_MIGRATED:      api.ReasonMigrated,
	libvirt.DOMAIN_SHUTOFF_SAVED:         api.ReasonSaved,
	libvirt.DOMAIN_SHUTOFF_FAILED:        api.ReasonFailed,
	libvirt.DOMAIN_SHUTOFF_FROM_SNAPSHOT: api.ReasonFromSnapshot,
}

var CrashedReasonTranslationMap = map[libvirt.DomainCrashedReason]api.StateChangeReason{
	libvirt.DOMAIN_CRASHED_UNKNOWN:  api.ReasonUnknown,
	libvirt.DOMAIN_CRASHED_PANICKED: api.ReasonPanicked,
}

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, namespace and field selector.
func newListWatchFromClient(c cli.Connection, events ...int) *cache.ListWatch {
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		logging.DefaultLogger().Info().V(3).Msg("Synchronizing domains")
		doms, err := c.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
		if err != nil {
			return nil, err
		}
		list := api.DomainList{
			Items: []api.Domain{},
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
			domain.SetState(convState(status), convReason(status, reason))
			list.Items = append(list.Items, *domain)
		}

		return &list, nil
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		return newDomainWatcher(c, events...)
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

type DomainWatcher struct {
	C chan watch.Event
}

func (d *DomainWatcher) Stop() {
	close(d.C)
}

func (d *DomainWatcher) ResultChan() <-chan watch.Event {
	return d.C
}

func newDomainWatcher(c cli.Connection, events ...int) (watch.Interface, error) {
	watcher := &DomainWatcher{C: make(chan watch.Event)}
	callback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {

		// check for reconnects, and emit an error to force a resync
		if event == nil {
			watcher.C <- watch.Event{Type: watch.Error, Object: &v1.Status{Status: v1.StatusFailure, Message: "Libvirt reconnected"}}
			return
		}
		logging.DefaultLogger().Info().V(3).Msgf("Libvirt event %d with reason %d received", event.Event, event.Detail)
		callback(d, event, watcher.C)
	}
	err := c.DomainEventLifecycleRegister(callback)
	if err != nil {
		logging.DefaultLogger().Info().V(2).Msg("Lifecycle event callback registered.")
	}
	return watcher, err
}

func NewDomainSpec(dom cli.VirDomain) (*api.DomainSpec, error) {
	domain := api.DomainSpec{}
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

// SplitVMNamespaceKey returns the namespace and name that is encoded in the
// domain name.
func SplitVMNamespaceKey(domainName string) (namespace, name string) {
	splitName := strings.SplitN(domainName, "_", 2)
	if len(splitName) == 1 {
		return k8sv1.NamespaceDefault, splitName[0]
	}
	return splitName[0], splitName[1]
}

func NewDomain(dom cli.VirDomain) (*api.Domain, error) {

	name, err := dom.GetName()
	if err != nil {
		return nil, err
	}
	namespace, name := SplitVMNamespaceKey(name)
	uuid, err := dom.GetUUIDString()
	if err != nil {
		return nil, err
	}

	domain := api.NewDomainReferenceFromName(namespace, name)
	domain.GetObjectMeta().SetUID(types.UID(uuid))
	return domain, nil
}

func NewSharedInformer(c cli.Connection) (cache.SharedInformer, error) {
	lw := newListWatchFromClient(c)
	informer := cache.NewSharedInformer(lw, &api.Domain{}, 0)
	return informer, nil
}

func callback(d cli.VirDomain, event *libvirt.DomainEventLifecycle, watcher chan watch.Event) {
	domain, err := NewDomain(d)
	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("Could not create the Domain.")
		watcher <- watch.Event{Type: watch.Error, Object: &v1.Status{Status: v1.StatusFailure, Message: err.Error()}}
		return
	}
	logging.DefaultLogger().Info().Msgf("event received: %v:%v", event.Event, event.Detail)
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
					watcher <- watch.Event{Type: watch.Error, Object: &v1.Status{Status: v1.StatusFailure, Message: err.Error()}}
					return
				}
			} else {
				domain.Spec = *spec
			}
		}
		status, reason, err := d.GetState()
		if err != nil {

			if err.(libvirt.Error).Code != libvirt.ERR_NO_DOMAIN {
				logging.DefaultLogger().Error().Reason(err).Msg("Could not fetch the Domain state.")
				watcher <- watch.Event{Type: watch.Error, Object: &v1.Status{Status: v1.StatusFailure, Message: err.Error()}}
				return
			}
			domain.SetState(api.NoState, api.ReasonUnknown)
		} else {
			domain.SetState(convState(status), convReason(status, reason))
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
		domain.SetState(convState(status), convReason(status, reason))
	}

	switch event.Event {
	case libvirt.DOMAIN_EVENT_DEFINED:
		if libvirt.DomainEventDefinedDetailType(event.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
			watcher <- watch.Event{Type: watch.Added, Object: domain}
		} else {
			watcher <- watch.Event{Type: watch.Modified, Object: domain}
		}
	case libvirt.DOMAIN_EVENT_UNDEFINED:
		watcher <- watch.Event{Type: watch.Deleted, Object: domain}
	default:
		watcher <- watch.Event{Type: watch.Modified, Object: domain}
	}

}

func convState(status libvirt.DomainState) api.LifeCycle {
	return LifeCycleTranslationMap[status]
}

func convReason(status libvirt.DomainState, reason int) api.StateChangeReason {
	switch status {
	case libvirt.DOMAIN_SHUTDOWN:
		return ShutdownReasonTranslationMap[libvirt.DomainShutdownReason(reason)]
	case libvirt.DOMAIN_SHUTOFF:
		return ShutoffReasonTranslationMap[libvirt.DomainShutoffReason(reason)]
	case libvirt.DOMAIN_CRASHED:
		return CrashedReasonTranslationMap[libvirt.DomainCrashedReason(reason)]
	default:
		return api.ReasonUnknown
	}
}
