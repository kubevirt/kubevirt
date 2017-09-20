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

	"github.com/jeevatkm/go-model"
	"github.com/libvirt/libvirt-go"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"

	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cache"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
)

type DomainManager interface {
	SyncVMSecret(vm *v1.VirtualMachine, usageType string, usageID string, secretValue string) error
	RemoveVMSecrets(*v1.VirtualMachine) error
	SyncVM(*v1.VirtualMachine) (*api.DomainSpec, error)
	KillVM(*v1.VirtualMachine) error
}

type LibvirtDomainManager struct {
	virConn              cli.Connection
	recorder             record.EventRecorder
	secretCache          map[string][]string
	podIsolationDetector isolation.PodIsolationDetector
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
				logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Failed to get libvirt secret.")
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
				logging.DefaultLogger().Error().Reason(err).Msg("Defining the VM secret failed.")
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
			logging.DefaultLogger().Error().Reason(err).Msg("Setting secret value for the VM failed.")
			return err
		}

	default:
		return goerrors.New(fmt.Sprintf("unsupported disk auth usage type %s", usageType))
	}
	return nil
}

func (l *LibvirtDomainManager) SyncVM(vm *v1.VirtualMachine) (*api.DomainSpec, error) {
	var wantedSpec api.DomainSpec
	wantedSpec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	wantedSpec.Type = "qemu"
	mappingErrs := model.Copy(&wantedSpec, vm.Spec.Domain)

	if len(mappingErrs) > 0 {
		logging.DefaultLogger().Error().Msg("model copy failed.")
		return nil, errors.NewAggregate(mappingErrs)
	}

	res, err := l.podIsolationDetector.Detect(vm)
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).V(3).Msgf("Could not detect virt-launcher cgroups.")
		return nil, err
	}

	logging.DefaultLogger().Object(vm).Info().With("slice", res.Slice()).V(3).Msg("Detected cgroup slice.")
	wantedSpec.QEMUCmd = &api.Commandline{
		QEMUEnv: []api.Env{
			{Name: "SLICE", Value: res.Slice()},
			{Name: "CONTROLLERS", Value: strings.Join(res.Controller(), ",")},
			{Name: "PIDNS", Value: res.PidNS()},
		},
	}

	domName := cache.VMNamespaceKeyFunc(vm)
	wantedSpec.Name = domName
	wantedSpec.UUID = string(vm.GetObjectMeta().GetUID())
	dom, err := l.virConn.LookupDomainByName(domName)
	newDomain := false
	if err != nil {
		// We need the domain but it does not exist, so create it
		if domainerrors.IsNotFound(err) {
			newDomain = true
			dom, err = l.setDomainXML(vm, wantedSpec)
			if err != nil {
				return nil, err
			}
			logging.DefaultLogger().Object(vm).Info().Msg("Domain defined.")
			l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Created.String(), "VM defined.")
		} else {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain failed.")
			return nil, err
		}
	}
	defer dom.Free()
	domState, _, err := dom.GetState()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain state failed.")
		return nil, err
	}

	// To make sure, that we set the right qemu wrapper arguments,
	// we update the domain XML whenever a VM was already defined but not running
	if !newDomain && cli.IsDown(domState) {
		dom, err = l.setDomainXML(vm, wantedSpec)
		if err != nil {
			return nil, err
		}
	}

	// TODO Suspend, Pause, ..., for now we only support reaching the running state
	// TODO for migration and error detection we also need the state change reason
	// TODO blocked state
	if cli.IsDown(domState) {
		err := dom.Create()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Starting the VM failed.")
			return nil, err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain started.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Started.String(), "VM started.")
	} else if cli.IsPaused(domState) {
		// TODO: if state change reason indicates a system error, we could try something smarter
		err := dom.Resume()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Resuming the VM failed.")
			return nil, err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain resumed.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Resumed.String(), "VM resumed")
	} else {
		// Nothing to do
	}

	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	var newSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &newSpec)
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Parsing domain XML failed.")
		return nil, err
	}

	// TODO: check if VM Spec and Domain Spec are equal or if we have to sync
	return &newSpec, nil
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
				logging.DefaultLogger().Object(vm).Error().Reason(err).Msg(fmt.Sprintf("Failed to lookup secret with UUID %s.", secretUUID))
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
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VM does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()
	// TODO: Graceful shutdown
	domState, _, err := dom.GetState()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		err = dom.Destroy()
		if err != nil {
			logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Destroying the domain state failed.")
			return err
		}
		logging.DefaultLogger().Object(vm).Info().Msg("Domain stopped.")
		l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Stopped.String(), "VM stopped")
	}

	err = dom.Undefine()
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Undefining the domain state failed.")
		return err
	}
	logging.DefaultLogger().Object(vm).Info().Msg("Domain undefined.")
	l.recorder.Event(vm, kubev1.EventTypeNormal, v1.Deleted.String(), "VM undefined")
	return nil
}

func (l *LibvirtDomainManager) setDomainXML(vm *v1.VirtualMachine, wantedSpec api.DomainSpec) (cli.VirDomain, error) {
	xmlStr, err := xml.Marshal(&wantedSpec)
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Generating the domain XML failed.")
		return nil, err
	}
	logging.DefaultLogger().Object(vm).Info().V(3).With("xml", xmlStr).Msgf("Domain XML generated.")
	dom, err := l.virConn.DomainDefineXML(string(xmlStr))
	if err != nil {
		logging.DefaultLogger().Object(vm).Error().Reason(err).Msg("Defining the VM failed.")
		return nil, err
	}
	return dom, nil
}
