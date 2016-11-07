package api

import (
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/meta"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
)

type VM struct {
	unversioned.TypeMeta
	ObjectMeta api.ObjectMeta
	Spec       VMSpec
}

type VMList struct {
	unversioned.TypeMeta
	unversioned.ListMeta
	VMs []VM
}

type VMSpec struct {
	NodeSelector map[string]string
}

// Required to satisfy Object interface
func (v *VM) GetObjectKind() unversioned.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VM) GetObjectMeta() meta.Object {
	return &v.ObjectMeta
}

// Required to satisfy Object interface
func (vl *VMList) GetObjectKind() unversioned.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VMList) GetListMeta() unversioned.List {
	return &vl.ListMeta
}
