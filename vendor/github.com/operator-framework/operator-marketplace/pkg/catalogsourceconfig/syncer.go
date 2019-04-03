package catalogsourceconfig

import (
	"time"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	Syncer *catalogSyncer
)

// InitializeStaticSyncer encapsulates the NewCatalogSyncer constructor
// to create a global instance of the CatalogSyncer type.
func InitializeStaticSyncer(client client.Client, initialWait time.Duration) {
	Syncer = NewCatalogSyncer(client, initialWait)
}

// NewCatalogSyncer returns a new instance of CatalogSyncer interface.
func NewCatalogSyncer(client client.Client, initialWait time.Duration) *catalogSyncer {
	return &catalogSyncer{
		initialWait:    initialWait,
		triggerer:      NewTriggerer(client, datastore.Cache),
		notificationCh: make(chan datastore.PackageUpdateNotification),
	}
}

// CatalogSyncer is an interface that wraps the Sync method.
//
// Sync is a loop that waits on a specified channel for package update
// notification. Once notification is received it triggers reconciliation of
// CatalogSourceConfig object(s) that needs to update catalog source(s).
type CatalogSyncer interface {
	Sync(stop <-chan struct{})
}

// catalogSyncer implements CatalogSyncer interface.
type catalogSyncer struct {
	initialWait    time.Duration
	triggerer      Triggerer
	notificationCh chan datastore.PackageUpdateNotification
}

func (s *catalogSyncer) Sync(stop <-chan struct{}) {
	log.Infof("[sync] CatalogSourceConfig sync loop will start after %s", s.initialWait)

	// Immediately after the operator process starts, it will spend time in
	// reconciling existing CR(s). Let's give the process a grace period to
	// reconcile and rebuild the local cache from existing CR(s).
	<-time.After(s.initialWait)

	log.Info("[sync] CatalogSourceConfig sync loop has started")
	for {
		select {
		case notification := <-s.notificationCh:
			if notification == nil {
				log.Error("[sync] package update notification cannot be <nil>")
				break
			}

			log.Info("[sync] received list of package(s) with new version, syncing CatalogSourceConfig object(s)")
			if err := s.triggerer.Trigger(notification); err != nil {
				log.Errorf("[sync] CatalogSourceConfig sync error- %v", err)
			}

		case <-stop:
			log.Info("[sync] Ending CatalogSourceConfig sync loop")
			return
		}
	}
}

// Send sends the specified update notification to the underlying channel.
func (s *catalogSyncer) Send(notification datastore.PackageUpdateNotification) {
	go func() {
		log.Info("[sync] sending list of package(s) with new version")
		s.notificationCh <- notification
	}()
}

// SendRefresh sends a refresh notification to trigger refresh of all packages
// if their datastore and status versions differ
func (s *catalogSyncer) SendRefresh() {
	refreshNotification := datastore.NewPackageRefreshNotification()
	s.Send(refreshNotification)
}
