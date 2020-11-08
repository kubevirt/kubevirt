package operands

import (
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cnaHandler genericOperand

func (h cnaHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	networkAddons := req.Instance.NewNetworkAddons()

	res := NewEnsureResult(networkAddons)
	key, err := client.ObjectKeyFromObject(networkAddons)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for Network Addons")
	}

	res.SetName(key.Name)
	found := &networkaddonsv1.NetworkAddonsConfig{}
	err = h.Client.Get(req.Ctx, key, found)

	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating Network Addons")
			err = h.Client.Create(req.Ctx, networkAddons)
			if err == nil {
				return res.SetUpdated()
			}
		}

		return res.Error(err)
	}

	existingOwners := found.GetOwnerReferences()

	// Previous versions used to have HCO-operator (scope namespace)
	// as the owner of NetworkAddons (scope cluster).
	// It's not legal, so remove that.
	if len(existingOwners) > 0 {
		req.Logger.Info("NetworkAddons has owners, removing...")
		found.SetOwnerReferences([]metav1.OwnerReference{})
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			req.Logger.Error(err, "Failed to remove NetworkAddons' previous owners")
		}
	}

	if !reflect.DeepEqual(found.Spec, networkAddons.Spec) && !req.UpgradeMode {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing Network Addons's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated Network Addons's Spec to its opinionated values")
			overwritten = true
		}
		networkAddons.Spec.DeepCopyInto(&found.Spec)
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			return res.Error(err)
		}
		if overwritten {
			res.SetOverwritten()
		}
		return res.SetUpdated()
	}

	req.Logger.Info("NetworkAddonsConfig already exists", "NetworkAddonsConfig.Namespace", found.Namespace, "NetworkAddonsConfig.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	// Handle conditions
	isReady := handleComponentConditions(req, "NetworkAddonsConfig", found.Status.Conditions)

	upgradeDone := req.ComponentUpgradeInProgress && isReady && checkComponentVersion(hcoutil.CnaoVersionEnvV, found.Status.ObservedVersion)

	return res.SetUpgradeDone(upgradeDone)

}
