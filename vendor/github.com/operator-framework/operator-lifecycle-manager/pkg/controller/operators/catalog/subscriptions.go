package catalog

import (
	"errors"
	"fmt"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ErrNilSubscription = errors.New("invalid Subscription object: <nil>")
)

const (
	PackageLabel = "alm-package"
	CatalogLabel = "alm-catalog"
	ChannelLabel = "alm-channel"
)

// FIXME(alecmerdler): Rewrite this whole block to be more clear
func (o *Operator) syncSubscription(in *v1alpha1.Subscription) (*v1alpha1.Subscription, error) {
	if in == nil || in.Spec == nil {
		return nil, ErrNilSubscription
	}
	out := in.DeepCopy()
	out = ensureLabels(out)

	// Only sync if catalog has been updated since last sync time
	if o.sourcesLastUpdate.Before(&out.Status.LastUpdated) && out.Status.State == v1alpha1.SubscriptionStateAtLatest {
		log.Infof("skipping sync: no new updates to catalog since last sync at %s",
			out.Status.LastUpdated.String())
		return nil, nil
	}

	o.sourcesLock.Lock()
	defer o.sourcesLock.Unlock()

	catalogNamespace := out.Spec.CatalogSourceNamespace
	if catalogNamespace == "" {
		catalogNamespace = o.namespace
	}
	catalog, ok := o.sources[registry.ResourceKey{Name: out.Spec.CatalogSource, Namespace: catalogNamespace}]
	if !ok {
		out.Status.State = v1alpha1.SubscriptionStateAtLatest
		out.Status.Reason = v1alpha1.SubscriptionReasonInvalidCatalog
		return out, fmt.Errorf("unknown catalog source %s in namespace %s", out.Spec.CatalogSource, catalogNamespace)
	}

	// Find latest CSV if no CSVs are installed already
	if out.Status.CurrentCSV == "" {
		if out.Spec.StartingCSV != "" {
			out.Status.CurrentCSV = out.Spec.StartingCSV
		} else {
			csv, err := catalog.FindCSVForPackageNameUnderChannel(out.Spec.Package, out.Spec.Channel)
			if err != nil {
				return out, fmt.Errorf("failed to find CSV for package %s in channel %s: %v", out.Spec.Package, out.Spec.Channel, err)
			}
			if csv == nil {
				return out, fmt.Errorf("failed to find CSV for package %s in channel %s: nil CSV", out.Spec.Package, out.Spec.Channel)
			}
			out.Status.CurrentCSV = csv.GetName()
		}
		out.Status.State = v1alpha1.SubscriptionStateUpgradeAvailable
		return out, nil
	}

	// Check that desired CSV has been installed
	csv, err := o.client.OperatorsV1alpha1().ClusterServiceVersions(out.GetNamespace()).Get(out.Status.CurrentCSV, metav1.GetOptions{})
	if err != nil || csv == nil {
		log.Infof("error fetching CSV %s via k8s api: %v", out.Status.CurrentCSV, err)
		if out.Status.Install != nil && out.Status.Install.Name != "" {
			ip, err := o.client.OperatorsV1alpha1().InstallPlans(out.GetNamespace()).Get(out.Status.Install.Name, metav1.GetOptions{})
			if err != nil {
				log.Errorf("get installplan %s error: %v", out.Status.Install.Name, err)
			}
			if err == nil && ip != nil {
				log.Infof("installplan for %s already exists", out.Status.CurrentCSV)
				return out, nil
			}
			log.Infof("installplan %s not found: creating new plan", out.Status.Install.Name)
			out.Status.Install = nil
		}

		// Install CSV if doesn't exist
		out.Status.State = v1alpha1.SubscriptionStateUpgradePending
		ip := &v1alpha1.InstallPlan{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: v1alpha1.InstallPlanSpec{
				ClusterServiceVersionNames: []string{out.Status.CurrentCSV},
				Approval:                   out.GetInstallPlanApproval(),
			},
		}
		ownerutil.AddNonBlockingOwner(ip, out)
		ip.SetGenerateName(fmt.Sprintf("install-%s-", out.Status.CurrentCSV))
		ip.SetNamespace(out.GetNamespace())

		// Inherit the subscription's catalog source
		ip.Spec.CatalogSource = out.Spec.CatalogSource
		ip.Spec.CatalogSourceNamespace = out.Spec.CatalogSourceNamespace

		res, err := o.client.OperatorsV1alpha1().InstallPlans(out.GetNamespace()).Create(ip)
		if err != nil {
			return out, fmt.Errorf("failed to ensure current CSV %s installed: %v", out.Status.CurrentCSV, err)
		}
		if res == nil {
			return out, errors.New("unexpected installplan returned by k8s api on create: <nil>")
		}
		out.Status.Install = &v1alpha1.InstallPlanReference{
			UID:        res.GetUID(),
			Name:       res.GetName(),
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.InstallPlanKind,
		}
		return out, nil
	}

	// Set the installed CSV
	out.Status.InstalledCSV = out.Status.CurrentCSV

	// Poll catalog for an update
	repl, err := catalog.FindReplacementCSVForPackageNameUnderChannel(out.Spec.Package, out.Spec.Channel, out.Status.CurrentCSV)
	if err != nil {
		out.Status.State = v1alpha1.SubscriptionStateAtLatest
		return out, fmt.Errorf("failed to lookup replacement CSV for %s: %v", out.Status.CurrentCSV, err)
	}
	if repl == nil {
		out.Status.State = v1alpha1.SubscriptionStateAtLatest
		return out, fmt.Errorf("nil replacement CSV for %s returned from catalog", out.Status.CurrentCSV)
	}

	// Update subscription with new latest
	out.Status.CurrentCSV = repl.GetName()
	out.Status.Install = nil
	out.Status.State = v1alpha1.SubscriptionStateUpgradeAvailable
	return out, nil
}

func ensureLabels(sub *v1alpha1.Subscription) *v1alpha1.Subscription {
	labels := sub.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[PackageLabel] = sub.Spec.Package
	labels[CatalogLabel] = sub.Spec.CatalogSource
	labels[ChannelLabel] = sub.Spec.Channel
	sub.SetLabels(labels)
	return sub
}
