package operatorsource

import (
	"fmt"
	"time"

	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cscRefresher PackageRefreshNotificationSender
)

// NewPoller returns a new instance of Poller interface.
func NewPoller(client client.Client, updateNotificationSendWait time.Duration, sender PackageUpdateNotificationSender, refresher PackageRefreshNotificationSender) Poller {
	poller := &poller{
		datastore: datastore.Cache,
		sender:    sender,
		refresher: refresher,
		helper: &pollHelper{
			factory:      appregistry.NewClientFactory(),
			datastore:    datastore.Cache,
			client:       client,
			transitioner: phase.NewTransitioner(),
		},
	}

	cscRefresher = refresher

	return poller
}

// Poller is an interface that wraps the Poll method.
//
// Poll iterates through all available operator source(s) that are in the
// underlying datastore and performs the following action(s):
//   a) It polls the remote registry namespace to check if there are any
//      update(s) available.
//
//   b) If there is an update available then it triggers a purge and rebuild
//      operation for the specified OperatorSource object.
//
// On any error during each iteration it logs the error encountered and moves
// on to the next OperatorSource object.
type Poller interface {
	Poll()

	// Initialize is the method that is called on the poller when the poller
	// is first started. It sends a package refresh notification to the
	// catalogsourceconfig syncer to force a comparison with the datastore
	// to refresh invalid state.
	Initialize()
}

// poller implements the Poller interface.
type poller struct {
	helper                     PollHelper
	datastore                  datastore.Writer
	sender                     PackageUpdateNotificationSender
	refresher                  PackageRefreshNotificationSender
	updateNotificationSendWait time.Duration
}

func (p *poller) Initialize() {
	log.Info("[sync] sending initial package update notification on start.")
	p.refresher.SendRefresh()
}

func (p *poller) Poll() {
	sources := p.datastore.GetAllOperatorSources()

	aggregator := datastore.NewPackageUpdateAggregator()

	for _, source := range sources {
		result, err := p.helper.HasUpdate(source)
		if err != nil {
			log.Errorf("[sync] error checking for updates [%s] - %v", source.Name, err)
			continue
		}

		if !result.RegistryHasUpdate {
			continue
		}

		log.Infof("operator source[%s] has updates: %s", source.Name, result)
		aggregator.Add(result)

		if err := p.trigger(source, result); err != nil {
			log.Errorf("%v", err)
		}
	}

	// We have a list of operator(s) that have either been removed or have new
	// version(s). We should kick off CatalogSourceConfig reconciliation.
	if !aggregator.IsUpdatedOrRemoved() {
		return
	}

	// TODO: This is a stop gap measure. We should not need this any longer when
	// CatalogSourceConfig has the version stored.
	<-time.After(p.updateNotificationSendWait)

	log.Infof("[sync] sending package update notification - %s", aggregator)
	p.sender.Send(aggregator)
}

func (p *poller) trigger(source *datastore.OperatorSourceKey, result *datastore.UpdateResult) error {
	log.Infof("[sync] remote registry has update(s) - purging OperatorSource [%s]", source.Name)
	deleted, err := p.helper.TriggerPurge(source)
	if err != nil {
		return fmt.Errorf("[sync] error updating object [%s] - %v", source.Name, err)
	}

	if deleted {
		log.Infof("[sync] object deleted [%s] - no action taken", source.Name)
	}

	return nil
}
