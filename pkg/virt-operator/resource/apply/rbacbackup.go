package apply

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func (r *Reconciler) backupRBACs() error {

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

		err := r.backupRBAC(cachedCr.DeepCopy(), cachedCr.Name, string(cachedCr.UID), imageTag, imageRegistry, id, TypeClusterRole)
		if err != nil {
			return err
		}
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

		err := r.backupRBAC(cachedCrb.DeepCopy(), cachedCrb.Name, string(cachedCrb.UID), imageTag, imageRegistry, id, TypeClusterRole)
		if err != nil {
			return err
		}
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

		err := r.backupRBAC(cachedCr.DeepCopy(), cachedCr.Name, string(cachedCr.UID), imageTag, imageRegistry, id, TypeClusterRole)
		if err != nil {
			return err
		}
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

		err := r.backupRBAC(cachedRb.DeepCopy(), cachedRb.Name, string(cachedRb.UID), imageTag, imageRegistry, id, TypeClusterRole)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) backupRBAC(obj interface{}, name, UID, imageTag, imageRegistry, id string, roleType RoleType) error {
	meta := getRoleMetaObject(obj, roleType)
	*meta = metav1.ObjectMeta{
		GenerateName: name,
	}
	injectOperatorMetadata(r.kv, meta, imageTag, imageRegistry, id, true)
	meta.Annotations[v1.EphemeralBackupObject] = UID

	// Create backup
	createRole := r.getRoleCreateFunction(obj, roleType)
	err := createRole()
	if err != nil {
		return err
	}

	log.Log.V(2).Infof("backup %v %v created", getRoleTypeName(roleType), name)
	return nil
}

func needsClusterRoleBackup(kv *v1.KubeVirt, stores util.Stores, cr *rbacv1.ClusterRole) bool {

	shouldBackup := shouldBackupRBACObject(kv, &cr.ObjectMeta)
	imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cr.ObjectMeta)
	if !shouldBackup || !ok {
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

	shouldBackup := shouldBackupRBACObject(kv, &rb.ObjectMeta)
	imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&rb.ObjectMeta)
	if !shouldBackup || !ok {
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

	shouldBackup := shouldBackupRBACObject(kv, &r.ObjectMeta)
	imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&r.ObjectMeta)
	if !shouldBackup || !ok {
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

func shouldBackupRBACObject(kv *v1.KubeVirt, objectMeta *metav1.ObjectMeta) bool {
	curVersion, curImageRegistry, curID := getTargetVersionRegistryID(kv)

	if objectMatchesVersion(objectMeta, curVersion, curImageRegistry, curID, kv.GetGeneration()) {
		// matches current target version already, so doesn't need backup
		return false
	}

	if objectMeta.Annotations == nil {
		return false
	}

	_, ok := objectMeta.Annotations[v1.EphemeralBackupObject]
	if ok {
		// ephemeral backup objects don't need to be backed up because
		// they are the backup
		return false
	}

	return true

}

func needsClusterRoleBindingBackup(kv *v1.KubeVirt, stores util.Stores, crb *rbacv1.ClusterRoleBinding) bool {

	shouldBackup := shouldBackupRBACObject(kv, &crb.ObjectMeta)
	imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&crb.ObjectMeta)
	if !shouldBackup || !ok {
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
