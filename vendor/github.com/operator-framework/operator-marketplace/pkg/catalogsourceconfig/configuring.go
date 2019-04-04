package catalogsourceconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultRegistryServerImage is the registry image to be used in the absence of
// the command line parameter.
const DefaultRegistryServerImage = "quay.io/openshift/origin-operator-registry"

// RegistryServerImage is the image used for creating the operator registry pod.
// This gets set in the cmd/manager/main.go.
var RegistryServerImage string

// NewConfiguringReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object in the "Configuring" phase.
func NewConfiguringReconciler(log *logrus.Entry, reader datastore.Reader, client client.Client, cache Cache) Reconciler {
	return &configuringReconciler{
		log:    log,
		reader: reader,
		client: client,
		cache:  cache,
	}
}

// configuringReconciler is an implementation of Reconciler interface that
// reconciles a CatalogSourceConfig object in the "Configuring" phase.
type configuringReconciler struct {
	log    *logrus.Entry
	reader datastore.Reader
	client client.Client
	cache  Cache
}

// Reconcile reconciles a CatalogSourceConfig object that is in the
// "Configuring" phase. It ensures that a corresponding CatalogSource object
// exists.
//
// Upon success, it returns "Succeeded" as the next and final desired phase.
// On error, the function returns "Failed" as the next desired phase
// and Message is set to the appropriate error message.
func (r *configuringReconciler) Reconcile(ctx context.Context, in *marketplace.CatalogSourceConfig) (out *marketplace.CatalogSourceConfig, nextPhase *marketplace.Phase, err error) {
	if in.Status.CurrentPhase.Name != phase.Configuring {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in

	// Populate the cache before we reconcile to preserve previous data
	// in case of a failure.
	r.cache.Set(out)

	err = r.reconcileCatalogSource(in)
	if err != nil {
		nextPhase = phase.GetNextWithMessage(phase.Configuring, err.Error())
		return
	}

	r.ensurePackagesInStatus(out)

	nextPhase = phase.GetNext(phase.Succeeded)

	r.log.Info("The object has been successfully reconciled")
	return
}

// reconcileCatalogSource ensures a CatalogSource exists with all the
// resources it requires.
func (r *configuringReconciler) reconcileCatalogSource(csc *marketplace.CatalogSourceConfig) error {
	// Ensure that the packages in the spec are available in the datastore
	err := r.checkPackages(csc)
	if err != nil {
		return err
	}

	// Ensure that a registry deployment is available
	registry := NewRegistry(r.log, r.client, r.reader, csc, RegistryServerImage)
	err = registry.Ensure()
	if err != nil {
		return err
	}

	// Check if the CatalogSource already exists
	catalogSourceGet := new(CatalogSourceBuilder).WithTypeMeta().CatalogSource()
	key := client.ObjectKey{
		Name:      csc.Name,
		Namespace: csc.Spec.TargetNamespace,
	}
	err = r.client.Get(context.TODO(), key, catalogSourceGet)

	// Update the CatalogSource if it exists else create one.
	if err == nil {
		catalogSourceGet.Spec.Address = registry.GetAddress()
		r.log.Infof("Updating CatalogSource %s", catalogSourceGet.Name)
		err = r.client.Update(context.TODO(), catalogSourceGet)
		if err != nil {
			r.log.Errorf("Failed to update CatalogSource : %v", err)
			return err
		}
		r.log.Infof("Updated CatalogSource %s", catalogSourceGet.Name)
	} else {
		// Create the CatalogSource structure
		catalogSource := newCatalogSource(csc, registry.GetAddress())
		r.log.Infof("Creating CatalogSource %s", catalogSource.Name)
		err = r.client.Create(context.TODO(), catalogSource)
		if err != nil && !errors.IsAlreadyExists(err) {
			r.log.Errorf("Failed to create CatalogSource : %v", err)
			return err
		}
		r.log.Infof("Created CatalogSource %s", catalogSource.Name)
	}

	return nil
}

// ensurePackagesInStatus makes sure that the csc's status.PackageRepositioryVersions
// field is updated at the end of the configuring phase if successful. It iterates
// over the list of packages and creates a new map of PackageName:Version for each
// package in the spec.
func (r *configuringReconciler) ensurePackagesInStatus(csc *marketplace.CatalogSourceConfig) {
	newPackageRepositioryVersions := make(map[string]string)
	packageIDs := csc.GetPackageIDs()
	for _, packageID := range packageIDs {
		version, err := r.reader.ReadRepositoryVersion(packageID)
		if err != nil {
			r.log.Errorf("Failed to find package: %v", err)
			version = "-1"
		}

		newPackageRepositioryVersions[packageID] = version
	}

	csc.Status.PackageRepositioryVersions = newPackageRepositioryVersions
}

// checkPackages returns an error if there are packages missing from the
// datastore but listed in the spec.
func (r *configuringReconciler) checkPackages(csc *marketplace.CatalogSourceConfig) error {
	missingPackages := []string{}
	packageIDs := csc.GetPackageIDs()
	for _, packageID := range packageIDs {
		if _, err := r.reader.Read(packageID); err != nil {
			missingPackages = append(missingPackages, packageID)
			continue
		}
	}

	if len(missingPackages) > 0 {
		return fmt.Errorf(
			"Still resolving package(s) - %s. Please make sure these are valid packages.",
			strings.Join(missingPackages, ","),
		)
	}
	return nil
}

// newCatalogSource returns a CatalogSource object.
func newCatalogSource(csc *marketplace.CatalogSourceConfig, address string) *olm.CatalogSource {
	builder := new(CatalogSourceBuilder).
		WithOwnerLabel(csc).
		WithMeta(csc.Name, csc.Spec.TargetNamespace).
		WithSpec(olm.SourceTypeGrpc, address, csc.Spec.DisplayName, csc.Spec.Publisher)

	// Check if the operatorsource.DatastoreLabel is "true" which indicates that
	// the CatalogSource is the datastore for an OperatorSource. This is a hint
	// for us to set the "olm-visibility" label in the CatalogSource so that it
	// is not visible in the OLM Packages UI. In addition we will set the
	// "openshift-marketplace" label which will be used by the Marketplace UI
	// to filter out global CatalogSources.
	cscLabels := csc.ObjectMeta.GetLabels()
	datastoreLabel, found := cscLabels[operatorsource.DatastoreLabel]
	if found && strings.ToLower(datastoreLabel) == "true" {
		builder.WithOLMLabels(cscLabels)
	}

	return builder.CatalogSource()
}
