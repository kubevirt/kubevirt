package api

import (
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
)

type VM struct {
	unversioned.TypeMeta
	api.ObjectMeta
	Spec VMSpec
}

type VMList struct {
	unversioned.TypeMeta
	unversioned.ListMeta
	VMs []VM
}

type VMSpec struct {
	NodeSelector map[string]string
}
