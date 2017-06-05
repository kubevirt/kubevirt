/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubev1 "k8s.io/client-go/pkg/api/v1"
)

type LifeCycle string
type StateChangeReason string

const (
	NoState     LifeCycle = "NoState"
	Running     LifeCycle = "Running"
	Blocked     LifeCycle = "Blocked"
	Paused      LifeCycle = "Paused"
	Shutdown    LifeCycle = "ShuttingDown"
	Shutoff     LifeCycle = "Shutoff"
	Crashed     LifeCycle = "Crashed"
	PMSuspended LifeCycle = "PMSuspended"

	// Common reasons
	ReasonUnknown StateChangeReason = "Unknown"

	// ShuttingDown reasons
	ReasonUser StateChangeReason = "User"

	// Shutoff reasons
	ReasonShutdown     StateChangeReason = "Shutdown"
	ReasonDestroyed    StateChangeReason = "Destroyed"
	ReasonMigrated     StateChangeReason = "Migrated"
	ReasonCrashed      StateChangeReason = "Crashed"
	ReasonPanicked     StateChangeReason = "Panicked"
	ReasonSaved        StateChangeReason = "Saved"
	ReasonFailed       StateChangeReason = "Failed"
	ReasonFromSnapshot StateChangeReason = "FromSnapshot"
)

type Domain struct {
	metav1.TypeMeta
	ObjectMeta kubev1.ObjectMeta
	Status     DomainStatus
}

type DomainStatus struct {
	Status LifeCycle
	Reason StateChangeReason
}

type DomainList struct {
	metav1.TypeMeta
	ListMeta metav1.ListMeta
	Items    []Domain
}

func NewMinimalDomain(name string) *Domain {
	return NewMinimalDomainWithNS(kubev1.NamespaceDefault, name)
}

func NewMinimalDomainWithNS(namespace string, name string) *Domain {
	return NewDomainReferenceFromName(namespace, name)
}

func NewDomainReferenceFromName(namespace string, name string) *Domain {
	return &Domain{
		ObjectMeta: kubev1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: DomainStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "1.2.2",
			Kind:       "Domain",
		},
	}
}

func (d *Domain) SetState(state LifeCycle, reason StateChangeReason) {
	d.Status.Status = state
	d.Status.Reason = reason
}

// Required to satisfy Object interface
func (d *Domain) GetObjectKind() schema.ObjectKind {
	return &d.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (d *Domain) GetObjectMeta() metav1.Object {
	return &d.ObjectMeta
}

// Required to satisfy Object interface
func (dl *DomainList) GetObjectKind() schema.ObjectKind {
	return &dl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (dl *DomainList) GetListMeta() metav1.List {
	return &dl.ListMeta
}
