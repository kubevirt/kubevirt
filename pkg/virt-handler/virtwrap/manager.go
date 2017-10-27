/*
 * This file is part of the KubeVirt project
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

package virtwrap

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/xml"
	goerrors "errors"
	"fmt"

	"github.com/libvirt/libvirt-go"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cache"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
	cli "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/libvirt"
)

type DomainManager interface {
	SyncVMSecret(vm *v1.VirtualMachine, usageType string, usageID string, secretValue string) error
	RemoveVMSecrets(*v1.VirtualMachine) error
	KillVM(*v1.VirtualMachine) error

	// XXX: Temporary methods, efforts are currently don to remove
	// manager.go, methods used during the transition.
	GetVirConn() cli.Connection
	GetPodIsolationDetector() isolation.PodIsolationDetector
}

type LibvirtDomainManager struct {
	virConn              cli.Connection
	recorder             record.EventRecorder
	secretCache          map[string][]string
	podIsolationDetector isolation.PodIsolationDetector
}

func (l *LibvirtDomainManager) GetVirConn() cli.Connection {
	return l.virConn
}

func (l *LibvirtDomainManager) GetPodIsolationDetector() isolation.PodIsolationDetector {
	return l.podIsolationDetector
}

func (l *LibvirtDomainManager) initiateSecretCache() error {
	secrets, err := l.virConn.ListSecrets()
	if err != nil {
		if err.(libvirt.Error).Code == libvirt.ERR_NO_SECRET {
			return nil
		} else {
			return err
		}
	}

	for _, secretUUID := range secrets {
		var secretSpec api.SecretSpec

		secret, err := l.virConn.LookupSecretByUUIDString(secretUUID)
		if err != nil {
			return err
		}
		defer secret.Free()

		xmlstr, err := secret.GetXMLDesc(0)
		if err != nil {
			return err
		}

		err = xml.Unmarshal([]byte(xmlstr), &secretSpec)
		if err != nil {
			return err
		}

		if secretSpec.Description == "" {
			continue
		}
		domName := secretSpec.Description
		l.secretCache[domName] = append(l.secretCache[domName], secretUUID)
	}

	return nil
}

func NewLibvirtDomainManager(connection cli.Connection, recorder record.EventRecorder, isolationDetector isolation.PodIsolationDetector) (DomainManager, error) {
	manager := LibvirtDomainManager{
		virConn:              connection,
		recorder:             recorder,
		secretCache:          make(map[string][]string),
		podIsolationDetector: isolationDetector,
	}

	err := manager.initiateSecretCache()
	if err != nil {
		return nil, err
	}
	return &manager, nil
}

func (l *LibvirtDomainManager) SyncVMSecret(vm *v1.VirtualMachine, usageType string, usageID string, secretValue string) error {

	domName := cache.VMNamespaceKeyFunc(vm)

	switch usageType {
	case "iscsi":
		libvirtSecret, err := l.virConn.LookupSecretByUsage(libvirt.SECRET_USAGE_TYPE_ISCSI, usageID)

		// If the secret doesn't exist, make it
		if err != nil {
			if err.(libvirt.Error).Code != libvirt.ERR_NO_SECRET {
				log.Log.Object(vm).Reason(err).Error("Failed to get libvirt secret.")
				return err

			}
			secretSpec := &api.SecretSpec{
				Ephemeral:   "no",
				Private:     "yes",
				Description: domName,
				Usage: api.SecretUsage{
					Type:   usageType,
					Target: usageID,
				},
			}

			xmlStr, err := xml.Marshal(&secretSpec)
			libvirtSecret, err = l.virConn.SecretDefineXML(string(xmlStr))
			if err != nil {
				log.Log.Reason(err).Error("Defining the VM secret failed.")
				return err
			}

			secretUUID, err := libvirtSecret.GetUUIDString()
			if err != nil {
				// This error really shouldn't occur. The UUID should be known
				// locally by the libvirt client. If this fails, we make a best
				// effort attempt at removing the secret from libvirt.
				libvirtSecret.Undefine()
				libvirtSecret.Free()
				return err
			}
			l.secretCache[domName] = append(l.secretCache[domName], secretUUID)
		}
		defer libvirtSecret.Free()

		err = libvirtSecret.SetValue([]byte(secretValue), 0)
		if err != nil {
			log.Log.Reason(err).Error("Setting secret value for the VM failed.")
			return err
		}

	default:
		return goerrors.New(fmt.Sprintf("unsupported disk auth usage type %s", usageType))
	}
	return nil
}

func (l *LibvirtDomainManager) RemoveVMSecrets(vm *v1.VirtualMachine) error {
	domName := cache.VMNamespaceKeyFunc(vm)

	secretUUIDs, ok := l.secretCache[domName]
	if ok == false {
		return nil
	}

	for _, secretUUID := range secretUUIDs {
		secret, err := l.virConn.LookupSecretByUUIDString(secretUUID)
		if err != nil {
			if err.(libvirt.Error).Code != libvirt.ERR_NO_SECRET {
				log.Log.Object(vm).Reason(err).Errorf("Failed to lookup secret with UUID %s.", secretUUID)
				return err
			}
			continue
		}
		defer secret.Free()

		err = secret.Undefine()
		if err != nil {
			return err
		}
	}

	delete(l.secretCache, domName)
	return nil
}

func (l *LibvirtDomainManager) KillVM(vm *v1.VirtualMachine) error {
	domName := cache.VMNamespaceKeyFunc(vm)
	dom, err := l.virConn.LookupGuestByName(domName)
	if err != nil {
		// If the VM does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vm).Reason(err).Error("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()
	// TODO: Graceful shutdown
	domState, _, err := dom.GetState()
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		err = dom.Destroy()
		if err != nil {
			log.Log.Object(vm).Reason(err).Error("Destroying the domain state failed.")
			return err
		}
		log.Log.Object(vm).Info("Domain stopped.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Stopped.String(), "VM stopped")
	}

	err = dom.Undefine()
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Undefining the domain state failed.")
		return err
	}
	log.Log.Object(vm).Info("Domain undefined.")
	l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Deleted.String(), "VM undefined")
	return nil
}
