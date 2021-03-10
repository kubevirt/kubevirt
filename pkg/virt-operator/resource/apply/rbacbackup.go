package apply

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func (r *Reconciler) backupRbac() error {

	rbac := r.clientset.RbacV1()

	// Backup existing ClusterRoles
	objects := r.stores.ClusterRoleCache.List()
	for _, obj := range objects {
		cachedCr, ok := obj.(*rbacv1.ClusterRole)
		if !ok || !needsClusterRoleBackup(r.kv, r.stores, cachedCr) {
			continue
		}
		imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cachedCr.ObjectMeta)
		if !ok {
			continue
		}

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		cr := cachedCr.DeepCopy()
		cr.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCr.Name,
		}
		injectOperatorMetadata(r.kv, &cr.ObjectMeta, imageTag, imageRegistry, id, true)
		cr.Annotations[v1.EphemeralBackupObject] = string(cachedCr.UID)

		// Create backup
		r.expectations.ClusterRole.RaiseExpectations(r.kvKey, 1, 0)
		_, err := rbac.ClusterRoles().Create(context.Background(), cr, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ClusterRole.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create backup clusterrole %+v: %v", cr, err)
		}

		log.Log.V(2).Infof("backup clusterrole %v created", cr.GetName())
	}

	// Backup existing ClusterRoleBindings
	objects = r.stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		cachedCrb, ok := obj.(*rbacv1.ClusterRoleBinding)
		if !ok || !needsClusterRoleBindingBackup(r.kv, r.stores, cachedCrb) {
			continue
		}
		imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cachedCrb.ObjectMeta)
		if !ok {
			continue
		}

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		crb := cachedCrb.DeepCopy()
		crb.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCrb.Name,
		}
		injectOperatorMetadata(r.kv, &crb.ObjectMeta, imageTag, imageRegistry, id, true)
		crb.Annotations[v1.EphemeralBackupObject] = string(cachedCrb.UID)

		// Create backup
		r.expectations.ClusterRoleBinding.RaiseExpectations(r.kvKey, 1, 0)
		_, err := rbac.ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ClusterRoleBinding.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create backup clusterrolebinding %+v: %v", crb, err)
		}
		log.Log.V(2).Infof("backup clusterrolebinding %v created", crb.GetName())
	}

	// Backup existing Roles
	objects = r.stores.RoleCache.List()
	for _, obj := range objects {
		cachedCr, ok := obj.(*rbacv1.Role)
		if !ok || !needsRoleBackup(r.kv, r.stores, cachedCr) {
			continue
		}
		imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cachedCr.ObjectMeta)
		if !ok {
			continue
		}

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		cr := cachedCr.DeepCopy()
		cr.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCr.Name,
		}
		injectOperatorMetadata(r.kv, &cr.ObjectMeta, imageTag, imageRegistry, id, true)
		cr.Annotations[v1.EphemeralBackupObject] = string(cachedCr.UID)

		// Create backup
		r.expectations.Role.RaiseExpectations(r.kvKey, 1, 0)
		_, err := rbac.Roles(cachedCr.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
		if err != nil {
			r.expectations.Role.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create backup role %+v: %v", r, err)
		}
		log.Log.V(2).Infof("backup role %v created", cr.GetName())
	}

	// Backup existing RoleBindings
	objects = r.stores.RoleBindingCache.List()
	for _, obj := range objects {
		cachedRb, ok := obj.(*rbacv1.RoleBinding)
		if !ok || !needsRoleBindingBackup(r.kv, r.stores, cachedRb) {
			continue
		}
		imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cachedRb.ObjectMeta)
		if ok {
			continue
		}

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		rb := cachedRb.DeepCopy()
		rb.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedRb.Name,
		}
		injectOperatorMetadata(r.kv, &rb.ObjectMeta, imageTag, imageRegistry, id, true)
		rb.Annotations[v1.EphemeralBackupObject] = string(cachedRb.UID)

		// Create backup
		r.expectations.RoleBinding.RaiseExpectations(r.kvKey, 1, 0)
		_, err := rbac.RoleBindings(cachedRb.Namespace).Create(context.Background(), rb, metav1.CreateOptions{})
		if err != nil {
			r.expectations.RoleBinding.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create backup rolebinding %+v: %v", rb, err)
		}
		log.Log.V(2).Infof("backup rolebinding %v created", rb.GetName())
	}

	return nil
}

func needsClusterRoleBackup(kv *v1.KubeVirt, stores util.Stores, cr *rbacv1.ClusterRole) bool {
	imageTag, imageRegistry, id, shouldBackup := shouldBackupRBACObject(kv, &cr.ObjectMeta)
	if !shouldBackup {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := stores.ClusterRoleCache.List()
	for _, obj := range objects {
		cachedCr, ok := obj.(*rbacv1.ClusterRole)

		if !ok ||
			cachedCr.DeletionTimestamp != nil ||
			cr.Annotations == nil {
			continue
		}

		uid, ok := cachedCr.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(cr.UID) && objectMatchesVersion(&cachedCr.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func needsRoleBindingBackup(kv *v1.KubeVirt, stores util.Stores, rb *rbacv1.RoleBinding) bool {

	imageTag, imageRegistry, id, shouldBackup := shouldBackupRBACObject(kv, &rb.ObjectMeta)
	if !shouldBackup {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := stores.RoleBindingCache.List()
	for _, obj := range objects {
		cachedRb, ok := obj.(*rbacv1.RoleBinding)

		if !ok ||
			cachedRb.DeletionTimestamp != nil ||
			rb.Annotations == nil {
			continue
		}

		uid, ok := cachedRb.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(rb.UID) && objectMatchesVersion(&cachedRb.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func needsRoleBackup(kv *v1.KubeVirt, stores util.Stores, r *rbacv1.Role) bool {

	imageTag, imageRegistry, id, shouldBackup := shouldBackupRBACObject(kv, &r.ObjectMeta)
	if !shouldBackup {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := stores.RoleCache.List()
	for _, obj := range objects {
		cachedR, ok := obj.(*rbacv1.Role)

		if !ok ||
			cachedR.DeletionTimestamp != nil ||
			r.Annotations == nil {
			continue
		}

		uid, ok := cachedR.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(r.UID) && objectMatchesVersion(&cachedR.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func shouldBackupRBACObject(kv *v1.KubeVirt, objectMeta *metav1.ObjectMeta) (imageTag, imageRegistry, id string, shouldBackup bool) {
	curVersion, curImageRegistry, curID := getTargetVersionRegistryID(kv)
	shouldBackup = false

	if objectMatchesVersion(objectMeta, curVersion, curImageRegistry, curID, kv.GetGeneration()) {
		// matches current target version already, so doesn't need backup
		return
	}

	if objectMeta.Annotations == nil {
		return
	}

	_, ok := objectMeta.Annotations[v1.EphemeralBackupObject]
	if ok {
		// ephemeral backup objects don't need to be backed up because
		// they are the backup
		return
	}

	imageTag, imageRegistry, id, shouldBackup = getInstallStrategyAnnotations(objectMeta)

	return

}

func needsClusterRoleBindingBackup(kv *v1.KubeVirt, stores util.Stores, crb *rbacv1.ClusterRoleBinding) bool {

	imageTag, imageRegistry, id, shouldBackup := shouldBackupRBACObject(kv, &crb.ObjectMeta)
	if !shouldBackup {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		cachedCrb, ok := obj.(*rbacv1.ClusterRoleBinding)

		if !ok ||
			cachedCrb.DeletionTimestamp != nil ||
			crb.Annotations == nil {
			continue
		}

		uid, ok := cachedCrb.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(crb.UID) && objectMatchesVersion(&cachedCrb.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func getInstallStrategyAnnotations(meta *metav1.ObjectMeta) (imageTag, imageRegistry, id string, ok bool) {
	ok = false

	imageTag, ok = meta.Annotations[v1.InstallStrategyVersionAnnotation]
	if !ok {
		return
	}
	imageRegistry, ok = meta.Annotations[v1.InstallStrategyRegistryAnnotation]
	if !ok {
		return
	}
	id, ok = meta.Annotations[v1.InstallStrategyIdentifierAnnotation]
	if !ok {
		return
	}

	ok = true
	return
}
