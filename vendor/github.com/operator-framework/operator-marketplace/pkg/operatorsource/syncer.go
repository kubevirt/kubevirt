package operatorsource

import (
	"time"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRegistrySyncer returns a new instance of RegistrySyncer interface.
func NewRegistrySyncer(client client.Client, initialWait time.Duration, resyncInterval time.Duration, updateNotificationSendWait time.Duration, sender PackageUpdateNotificationSender, refresher PackageRefreshNotificationSender) RegistrySyncer {
	return &registrySyncer{
		initialWait:    initialWait,
		resyncInterval: resyncInterval,
		poller:         NewPoller(client, updateNotificationSendWait, sender, refresher),
	}
}

// PackageUpdateNotificationSender is an interface that wraps the Send method.
//
// Send sends package update notification to the underlying channel that
// CatalogSourceConfig is waiting on. This method is expected to be a non
// blocking operation.
type PackageUpdateNotificationSender interface {
	Send(notification datastore.PackageUpdateNotification)
}

// PackageRefreshNotificationSender is an interface that wraps the SendRefresh method.
//
// SendRefresh sends package refresh notification to the underlying channel that
// CatalogSourceConfig is waiting on. This method is expected to be a non
// blocking operation. When this notification is sent, all non datastore
// catalogsourceconfigs check their version map in the status against the datastore.
type PackageRefreshNotificationSender interface {
	SendRefresh()
}

// RegistrySyncer is an interface that wraps the Sync method.
//
// Sync kicks off the registry sync operation every N (resync wait time)
// minutes. Sync will stop running once the stop channel is closed.
type RegistrySyncer interface {
	Sync(stop <-chan struct{})
}

// registrySyncer implements RegistrySyncer interface.
type registrySyncer struct {
	initialWait    time.Duration
	resyncInterval time.Duration
	poller         Poller
}

func (s *registrySyncer) Sync(stop <-chan struct{}) {
	log.Infof("[sync] Operator source sync loop will start after %s", s.initialWait)

	// Immediately after the operator process starts, it will spend time in
	// reconciling existing OperatorSource CR(s). Let's give the process a
	// grace period to reconcile and rebuild the local cache from existing CR(s).
	<-time.After(s.initialWait)

	s.poller.Initialize()

	log.Info("[sync] Operator source sync loop has started")
	for {
		select {
		case <-time.After(s.resyncInterval):
			log.Info("[sync] Checking for operator source update(s) in remote registry")
			s.poller.Poll()

		case <-stop:
			log.Info("[sync] Ending operator source watch loop")
			return
		}
	}
}
