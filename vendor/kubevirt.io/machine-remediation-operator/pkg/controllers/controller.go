package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, opts manager.Options, fnList ...func(manager.Manager, manager.Options) error) error {
	for _, f := range fnList {
		if err := f(m, opts); err != nil {
			return err
		}
	}
	return nil
}
