package hyperconverged

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

type EnsureResult struct {
	Updated     bool
	Created     bool
	UpgradeDone bool
	Err         error
	Type        string
	Name        string
}

func NewEnsureResult(resource runtime.Object) *EnsureResult {
	t := fmt.Sprintf("%T", resource)
	p := strings.LastIndex(t, ".")
	return &EnsureResult{Type: t[p+1:]}
}

func (r *EnsureResult) Error(err error) *EnsureResult {
	r.Err = err
	return r
}

func (r *EnsureResult) SetCreated() *EnsureResult {
	r.Created = true
	return r
}

func (r *EnsureResult) SetUpdated() *EnsureResult {
	r.Updated = true
	return r
}

func (r *EnsureResult) SetUpgradeDone(upgradeDone bool) *EnsureResult {
	r.UpgradeDone = upgradeDone
	return r
}

func (r *EnsureResult) SetName(name string) *EnsureResult {
	r.Name = name
	return r
}
