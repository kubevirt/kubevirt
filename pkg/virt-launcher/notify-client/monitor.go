package eventsclient

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"libvirt.org/go/libvirt"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

type targetMigrationMonitor struct {
	c             cli.Connection
	events        chan watch.Event
	vmi           *v1.VirtualMachineInstance
	domain        *api.Domain
	metadataCache *metadata.Cache
	client        *Notifier
}

func newTargetMigrationMonitor(
	c cli.Connection,
	events chan watch.Event,
	vmi *v1.VirtualMachineInstance,
	domain *api.Domain,
	metadataCache *metadata.Cache,
	client *Notifier,
) *targetMigrationMonitor {
	return &targetMigrationMonitor{
		c:             c,
		events:        events,
		vmi:           vmi,
		domain:        domain,
		metadataCache: metadataCache,
		client:        client}
}

var retryDelays = []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second}

func (m *targetMigrationMonitor) startMonitor() {
	go func() {
		var err error
		domName := api.VMINamespaceKeyFunc(m.vmi)
		for attempt := 0; attempt <= len(retryDelays); attempt++ {
			err = virtwait.PollImmediately(50*time.Millisecond, 30*time.Second, func(context context.Context) (bool, error) {
				dom, err := m.c.LookupDomainByName(domName)
				if err != nil {
					return false, err
				}
				defer dom.Free()
				jobInfo, err := dom.GetJobInfo()
				if err != nil {
					return false, err
				}
				if jobInfo.Type == libvirt.DOMAIN_JOB_NONE {
					return true, nil
				}
				log.Log.Object(m.vmi).V(4).Infof("Incoming migration job active (type %d), waiting...", jobInfo.Type)
				return false, nil
			})
			if err == nil || !errors.IsTimeout(err) || attempt == len(retryDelays) {
				break
			}
			log.Log.Object(m.vmi).Info("A migration job is still active, retrying after delay")
			time.Sleep(retryDelays[attempt])
		}
		if err != nil {
			log.Log.Object(m.vmi).Info("Error polling libvirt, setting EndTimestamp anyway to unblock migration")
		} else {
			log.Log.Object(m.vmi).Info("Incoming migration job completed, setting EndTimestamp")
		}
		setEndTimestamp(m.metadataCache)
		event := watch.Event{Type: watch.Modified, Object: m.domain}
		m.client.SendDomainEvent(event)
		updateEvents(event, m.domain, m.events)
	}()
}

func setEndTimestamp(metadataCache *metadata.Cache) {
	migrationMetadata, exists := metadataCache.Migration.Load()
	if exists && migrationMetadata.EndTimestamp == nil {
		metadataCache.Migration.WithSafeBlock(func(migrationMetadata *api.MigrationMetadata, _ bool) {
			migrationMetadata.EndTimestamp = pointer.P(metav1.Now())
		})
	} else if !exists {
		migrationMetadata := api.MigrationMetadata{
			EndTimestamp: pointer.P(metav1.Now()),
		}
		metadataCache.Migration.Store(migrationMetadata)
	}
}
