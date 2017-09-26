package migration

import (
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

func NewStrategy(typer runtime.ObjectTyper) kubevirtStrategy {
	return kubevirtStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	migration, ok := obj.(*v1.Migration)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a Migration.")
	}
	return labels.Set(migration.ObjectMeta.Labels), migrationToSelectableFields(migration), migration.ObjectMeta.Initializers != nil, nil
}

// MatchMigration is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchMigration(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// migrationToSelectableFields returns a field set that represents the object.
func migrationToSelectableFields(obj *v1.Migration) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

type kubevirtStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (kubevirtStrategy) NamespaceScoped() bool {
	return true
}

func (kubevirtStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {
}

func (kubevirtStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
}

func (kubevirtStrategy) Validate(ctx genericapirequest.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (kubevirtStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (kubevirtStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (kubevirtStrategy) Canonicalize(obj runtime.Object) {
}

func (kubevirtStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
