package hyperconverged

import (
	"fmt"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"sync"
)

// CRDRemover removes old CRD we have to delete
type CRDRemover struct {
	client client.Client

	// Old v2v CRDs that need to be removed from the cluster.
	// Removal of the CRDs also leads to removal of its CRs.
	crdsToRemove []schema.GroupKind

	// Old v2v CRs that need to be removed from the list of related objects.
	// This list is maintained separately from the list of CRDs to remove,
	// so we can retry to delete them upon HCO status update failures, even
	// when the CRD itself has been successfully removed.
	relatedObjectsToRemove []schema.GroupKind
}

func (c *CRDRemover) Remove(req *common.HcoRequest) error {
	// Attempt to remove old CRDs
	if len(c.crdsToRemove) > 0 {
		c.removeCRDs(req)
	}

	if len(c.relatedObjectsToRemove) > 0 {
		// CRDs that were removed from the related objects list are kept around,
		// so that the next reconciliation loop can validate that they were actually removed,
		// and not lost in some failing status update.
		err := c.removeRelatedObjects(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CRDRemover) Done() bool {
	if len(c.crdsToRemove) > 0 || len(c.relatedObjectsToRemove) > 0 {
		return false
	}
	return true
}

// This function creates a CRD name with the form <plural>.<group>,
// where <plural> is the lowercase plural form of the Kind.
// Note that this doesn't use any generic pluralization, so may not
// work for CRDs other than the old SSP CRDs we're trying to handle here.
func (c *CRDRemover) groupKindToCRDName(gk schema.GroupKind) string {
	plural := strings.ToLower(gk.Kind) + "s"
	return fmt.Sprintf("%s.%s", plural, gk.Group)
}

// returns true if not found or if deletion succeeded, and false otherwise.
func (c *CRDRemover) removeCRD(req *common.HcoRequest, crd schema.GroupKind) bool {
	crdName := c.groupKindToCRDName(crd)
	found := &apiextensionsv1.CustomResourceDefinition{}
	key := client.ObjectKey{Namespace: hcoutil.UndefinedNamespace, Name: crdName}
	err := c.client.Get(req.Ctx, key, found)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			req.Logger.Error(err, fmt.Sprintf("failed to read the %s CRD; %s", crdName, err.Error()))
			return false
		}
	} else {
		err = c.client.Delete(req.Ctx, found)
		if err != nil {
			req.Logger.Error(err, fmt.Sprintf("failed to remove the %s CRD; %s", crdName, err.Error()))
			return false
		}

		req.Logger.Info("successfully removed CRD", "CRD Name", crdName)
	}

	return true
}

// returns a slice of CRDs that were successfully removed or don't exist, and a slice of CRDs that failed to remove.
func (c *CRDRemover) removeCRDs(req *common.HcoRequest) {
	removed := make([]schema.GroupKind, 0, len(c.crdsToRemove))
	unremoved := make([]schema.GroupKind, 0, len(c.crdsToRemove))

	// The deletion is performed concurrently for all CRDs.
	var mutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(c.crdsToRemove))

	for _, crdToBeRemoved := range c.crdsToRemove {
		go func(crd schema.GroupKind) {
			isRemoved := c.removeCRD(req, crd)

			mutex.Lock()
			defer mutex.Unlock()

			if isRemoved {
				removed = append(removed, crd)
			} else {
				unremoved = append(unremoved, crd)
			}

			wg.Done()
		}(crdToBeRemoved)
	}

	wg.Wait()

	// For CRDs that failed to remove, we'll retry in the next reconciliation loop.
	c.crdsToRemove = unremoved

	// For CRDs that were successfully removed, we can proceed to remove
	// their corresponding entries from HCO.Status.RelatedObjects.
	c.relatedObjectsToRemove = append(c.relatedObjectsToRemove, removed...)
}

// Removes CRs of the given CRDs from the list of related objects in HCO status.
// Returns a list of CRDs that were removed from the related objects list.
func (c *CRDRemover) removeRelatedObjects(req *common.HcoRequest) error {
	// Collect CRD GKs into a set
	roSet := make(map[schema.GroupKind]struct{})
	for _, crd := range c.relatedObjectsToRemove {
		roSet[crd] = struct{}{}
	}

	objRefsToRemove := make([]corev1.ObjectReference, 0, len(c.relatedObjectsToRemove))
	removedROs := make([]schema.GroupKind, 0, len(c.relatedObjectsToRemove))

	// Find related objects to be removed
	for _, objRef := range req.Instance.Status.RelatedObjects {
		objGK := objRef.GroupVersionKind().GroupKind()
		if _, exist := roSet[objGK]; exist {
			objRefsToRemove = append(objRefsToRemove, objRef)
			removedROs = append(removedROs, objGK)
		}
	}

	// Remove from related objects
	if len(objRefsToRemove) > 0 {
		req.StatusDirty = true
		for _, objRef := range objRefsToRemove {
			err := objectreferencesv1.RemoveObjectReference(&req.Instance.Status.RelatedObjects, objRef)
			if err != nil {
				// This shouldn't really happen, but...
				req.Logger.Error(err, "Failed removing object reference from HCO.Status.RelatedObjects",
					"ObjectReference.Name", objRef.Name,
					"ObjectReference.Namespace", objRef.Namespace,
					"ObjectReference.Kind", objRef.Kind,
					"ObjectReference.APIVersion", objRef.APIVersion)
				return err
			}
		}

	}
	c.relatedObjectsToRemove = removedROs
	return nil
}
