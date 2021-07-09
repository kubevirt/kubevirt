package apply

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

func (r *Reconciler) backupRBACs() error {

	// Backup existing ClusterRoles
	objects := r.stores.ClusterRoleCache.List()
	for _, obj := range objects {
		cachedCr, ok := obj.(*rbacv1.ClusterRole)
		if !ok || !needsBackup(r.kv, r.stores.ClusterRoleCache, &cachedCr.ObjectMeta) {
			continue
		}
		imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cachedCr.ObjectMeta)
		if !ok {
			continue
		}

		err := r.backupRBAC(cachedCr.DeepCopy(), cachedCr.Name, string(cachedCr.UID), imageTag, imageRegistry, id)
		if err != nil {
			return err
		}
	}

	// Backup existing ClusterRoleBindings
	objects = r.stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		cachedCrb, ok := obj.(*rbacv1.ClusterRoleBinding)
		if !ok || !needsBackup(r.kv, r.stores.ClusterRoleBindingCache, &cachedCrb.ObjectMeta) {
			continue
		}
		imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cachedCrb.ObjectMeta)
		if !ok {
			continue
		}

		err := r.backupRBAC(cachedCrb.DeepCopy(), cachedCrb.Name, string(cachedCrb.UID), imageTag, imageRegistry, id)
		if err != nil {
			return err
		}
	}

	// Backup existing Roles
	objects = r.stores.RoleCache.List()
	for _, obj := range objects {
		cachedCr, ok := obj.(*rbacv1.Role)
		if !ok || !needsBackup(r.kv, r.stores.RoleCache, &cachedCr.ObjectMeta) {
			continue
		}
		imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cachedCr.ObjectMeta)
		if !ok {
			continue
		}

		err := r.backupRBAC(cachedCr.DeepCopy(), cachedCr.Name, string(cachedCr.UID), imageTag, imageRegistry, id)
		if err != nil {
			return err
		}
	}

	// Backup existing RoleBindings
	objects = r.stores.RoleBindingCache.List()
	for _, obj := range objects {
		cachedRb, ok := obj.(*rbacv1.RoleBinding)
		if !ok || !needsBackup(r.kv, r.stores.RoleBindingCache, &cachedRb.ObjectMeta) {
			continue
		}
		imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(&cachedRb.ObjectMeta)
		if ok {
			continue
		}

		err := r.backupRBAC(cachedRb.DeepCopy(), cachedRb.Name, string(cachedRb.UID), imageTag, imageRegistry, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) backupRBAC(obj runtime.Object, name, UID, imageTag, imageRegistry, id string) error {
	meta := getMetaObject(obj)
	*meta = metav1.ObjectMeta{
		GenerateName: name,
	}
	injectOperatorMetadata(r.kv, meta, imageTag, imageRegistry, id, true)
	meta.Annotations[v1.EphemeralBackupObject] = UID

	// Create backup
	createRole := r.getCreateFunction(obj)
	err := createRole()
	if err != nil {
		return err
	}

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	log.Log.V(2).Infof("backup %v %v created", kind, name)
	return nil
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

func needsBackup(kv *v1.KubeVirt, cache cache.Store, meta *metav1.ObjectMeta) bool {
	shouldBackup := shouldBackupRBACObject(kv, meta)
	imageTag, imageRegistry, id, ok := getInstallStrategyAnnotations(meta)
	if !shouldBackup || !ok {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := cache.List()
	for _, obj := range objects {
		cachedObj, ok := obj.(*metav1.ObjectMeta)

		if !ok ||
			cachedObj.DeletionTimestamp != nil ||
			meta.Annotations == nil {
			continue
		}

		uid, ok := cachedObj.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(meta.UID) && objectMatchesVersion(cachedObj, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}
