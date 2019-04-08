package catalogsourceconfig

import (
	"context"
	"fmt"
	"strings"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/phase"

	log "github.com/sirupsen/logrus"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewTriggerer returns a new instance of Triggerer interface.
func NewTriggerer(client client.Client, reader datastore.Reader) Triggerer {
	return &triggerer{
		client:       client,
		reader:       reader,
		transitioner: phase.NewTransitioner(),
	}
}

// Triggerer is an interface that wraps the Trigger method.
//
// Trigger iterates through the list of all CatalogSourceConfig object(s) and
// applies the following logic:
//
// a. Compare the list of package(s) specified in Spec.Packages with the update
//    notification list and determine if the given CatalogSourceConfig specifies
//    a package that has either been removed or has a new version available.
//
// b. If the above is true then update the given CatalogSourceConfig object in
//    order to kick off a new reconciliation. This way it will get the latest
//    package manifest from datastore.
//
// The list call applies the label selector [opsrc-datastore!=true] to exclude
// CatalogSourceConfig object which is used as datastore for marketplace.
type Triggerer interface {
	Trigger(notification datastore.PackageUpdateNotification) error
}

// triggerer implements the Triggerer interface.
type triggerer struct {
	reader       datastore.Reader
	client       client.Client
	transitioner phase.Transitioner
}

func (t *triggerer) Trigger(notification datastore.PackageUpdateNotification) error {
	options := &client.ListOptions{}
	options.SetLabelSelector(fmt.Sprintf("%s!=true", operatorsource.DatastoreLabel))

	cscs := &marketplace.CatalogSourceConfigList{}
	if err := t.client.List(context.TODO(), options, cscs); err != nil {
		return err
	}

	allErrors := []error{}
	for _, instance := range cscs.Items {
		// Needed because sdk does not get the gvk.
		instance.EnsureGVK()

		packages, updateNeeded := t.setPackages(&instance, notification)
		if !updateNeeded {
			continue
		}

		if err := t.update(&instance, packages); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	return utilerrors.NewAggregate(allErrors)
}

func (t *triggerer) setPackages(instance *marketplace.CatalogSourceConfig, notification datastore.PackageUpdateNotification) (packages string, updateNeeded bool) {
	packageList := make([]string, 0)
	for _, pkg := range instance.GetPackageIDs() {
		// If this is a refresh notification, we need to access the datastore to determine
		// what catalogsourceconfigs need to be refreshed.
		if notification.IsRefreshNotification() {
			datastoreVersion, err := t.reader.ReadRepositoryVersion(pkg)
			if err != nil {
				log.Errorf(
					"Unable to resolve package %s associated with csc %s in datastore. Removing.",
					pkg, fmt.Sprintf("Name: %s Namespace: %s", instance.Name, instance.Namespace),
				)
				continue
			}

			packageList = append(packageList, pkg)

			if pkgVersion, available := instance.Status.PackageRepositioryVersions[pkg]; !available {
				// If the package isn't in the status, we need to force an update.
				updateNeeded = true
			} else {
				// If the status has a different version than the datastore, we need
				// to force an update.
				if pkgVersion != datastoreVersion {
					updateNeeded = true
				}
			}
		} else {
			// Otherwise, lets look at the notification to see what packages were updated/removed
			if notification.IsRemoved(pkg) {
				updateNeeded = true

				// The package specified has been removed from the registry. We will
				// remove it from the spec.
				continue
			}

			packageList = append(packageList, pkg)

			if notification.IsUpdated(pkg) {
				updateNeeded = true
			}
		}
	}

	packages = strings.Join(packageList, ",")
	return
}

func (t *triggerer) update(instance *marketplace.CatalogSourceConfig, packages string) error {
	out := instance.DeepCopy()

	// We want to Set the phase to Initial to kick off reconciliation anew.
	nextPhase := &marketplace.Phase{
		Name:    phase.Initial,
		Message: "Package(s) have update(s), scheduling for reconciliation",
	}
	t.transitioner.TransitionInto(&out.Status.CurrentPhase, nextPhase)

	out.Spec.Packages = packages

	if err := t.client.Update(context.TODO(), out); err != nil {
		return err
	}

	return nil
}
