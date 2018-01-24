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

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/registry-disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type DomainManager interface {
	SyncVMSecret(vm *v1.VirtualMachine, usageType string, usageID string, secretValue string) error
	RemoveVMSecrets(*v1.VirtualMachine) error
	SyncVM(*v1.VirtualMachine, map[string]*kubev1.Secret) (*api.DomainSpec, error)
	KillVM(*v1.VirtualMachine) error
	SignalShutdownVM(*v1.VirtualMachine) error
	ListAllDomains() ([]*api.Domain, error)
}

type LibvirtDomainManager struct {
	virConn     cli.Connection
	secretCache map[string][]string
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

func NewLibvirtDomainManager(connection cli.Connection) (DomainManager, error) {
	manager := LibvirtDomainManager{
		virConn:     connection,
		secretCache: make(map[string][]string),
	}

	err := manager.initiateSecretCache()
	if err != nil {
		return nil, err
	}
	return &manager, nil
}

func (l *LibvirtDomainManager) SyncVMSecret(vm *v1.VirtualMachine, usageType string, usageID string, secretValue string) error {

	domName := api.VMNamespaceKeyFunc(vm)

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

// All local environment setup that needs to occur before VM starts
// can be done in this function. This includes things like...
//
// - storage prep
// - network prep
// - cloud-init
func (l *LibvirtDomainManager) preStartHook(vm *v1.VirtualMachine, domain *api.Domain) error {

	// ensure registry disk files have correct ownership privileges
	err := registrydisk.SetFilePermissions(vm)
	if err != nil {
		return err
	}

	// generate cloud-init data
	cloudInitData := cloudinit.GetCloudInitNoCloudSource(vm)
	if cloudInitData != nil {
		err := cloudinit.GenerateLocalData(vm.Name, vm.Namespace, cloudInitData)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *LibvirtDomainManager) SyncVM(vm *v1.VirtualMachine, secrets map[string]*kubev1.Secret) (*api.DomainSpec, error) {
	logger := log.Log.Object(vm)

	domain := &api.Domain{}

	// Map the VirtualMachine to the Domain
	c := &api.ConverterContext{
		VirtualMachine: vm,
		Secrets:        secrets,
	}
	if err := api.Convert_v1_VirtualMachine_To_api_Domain(vm, domain, c); err != nil {
		logger.Error("Conversion failed.")
		return nil, err
	}

	// Set defaults which are not comming from the cluster
	api.SetObjectDefaults_Domain(domain)

	dom, err := l.virConn.LookupDomainByName(domain.Spec.Name)
	newDomain := false
	if err != nil {
		// We need the domain but it does not exist, so create it
		if domainerrors.IsNotFound(err) {
			newDomain = true
			dom, err = util.SetDomainSpec(l.virConn, vm, domain.Spec)
			if err != nil {
				return nil, err
			}
			logger.Info("Domain defined.")
		} else {
			logger.Reason(err).Error("Getting the domain failed.")
			return nil, err
		}
	}
	defer dom.Free()
	domState, _, err := dom.GetState()
	if err != nil {
		logger.Reason(err).Error("Getting the domain state failed.")
		return nil, err
	}

	// To make sure, that we set the right qemu wrapper arguments,
	// we update the domain XML whenever a VM was already defined but not running
	if !newDomain && cli.IsDown(domState) {
		dom, err = util.SetDomainSpec(l.virConn, vm, domain.Spec)
		if err != nil {
			return nil, err
		}
	}

	// TODO Suspend, Pause, ..., for now we only support reaching the running state
	// TODO for migration and error detection we also need the state change reason
	// TODO blocked state
	if cli.IsDown(domState) {
		err = l.preStartHook(vm, domain)
		if err != nil {
			logger.Reason(err).Error("pre start setup for VM failed.")
			return nil, err
		}
		err = dom.Create()
		if err != nil {
			logger.Reason(err).Error("Starting the VM failed.")
			return nil, err
		}
		logger.Info("Domain started.")
	} else if cli.IsPaused(domState) {
		// TODO: if state change reason indicates a system error, we could try something smarter
		err := dom.Resume()
		if err != nil {
			logger.Reason(err).Error("Resuming the VM failed.")
			return nil, err
		}
		logger.Info("Domain resumed.")
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
		logger.Reason(err).Error("Parsing domain XML failed.")
		return nil, err
	}

	// TODO: check if VM Spec and Domain Spec are equal or if we have to sync
	return &newSpec, nil
}

func (l *LibvirtDomainManager) RemoveVMSecrets(vm *v1.VirtualMachine) error {
	domName := api.VMNamespaceKeyFunc(vm)

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

func (l *LibvirtDomainManager) getDomainSpec(dom cli.VirDomain) (*api.DomainSpec, error) {
	return util.GetDomainSpec(dom)
}

func (l *LibvirtDomainManager) SignalShutdownVM(vm *v1.VirtualMachine) error {
	domName := util.VMNamespaceKeyFunc(vm)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VM does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vm).Reason(err).Error("Getting the domain failed during graceful shutdown.")
			return err
		}
	}
	defer dom.Free()

	domState, _, err := dom.GetState()
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		domSpec, err := l.getDomainSpec(dom)
		if err != nil {
			log.Log.Object(vm).Reason(err).Error("Unable to retrieve domain xml")
			return err
		}

		if domSpec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp == nil {
			err = dom.Shutdown()
			if err != nil {
				log.Log.Object(vm).Reason(err).Error("Signalling graceful shutdown failed.")
				return err
			}
			log.Log.Object(vm).Infof("Signaled graceful shutdown for %s", vm.GetObjectMeta().GetName())

			now := k8sv1.Now()
			domSpec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp = &now
			_, err = util.SetDomainSpec(l.virConn, vm, *domSpec)
			if err != nil {
				log.Log.Object(vm).Reason(err).Error("Unable to update grace period start time on domain xml")
				return err
			}
		}
	}

	return nil
}

func (l *LibvirtDomainManager) KillVM(vm *v1.VirtualMachine) error {
	domName := api.VMNamespaceKeyFunc(vm)
	dom, err := l.virConn.LookupDomainByName(domName)
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
	}

	err = dom.Undefine()
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Undefining the domain state failed.")
		return err
	}
	log.Log.Object(vm).Info("Domain undefined.")
	return nil
}

func (l *LibvirtDomainManager) ListAllDomains() ([]*api.Domain, error) {

	doms, err := l.virConn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, err
	}

	var list []*api.Domain
	for _, dom := range doms {
		domain, err := util.NewDomain(dom)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		spec, err := util.GetDomainSpec(dom)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		domain.Spec = *spec
		status, reason, err := dom.GetState()
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		domain.SetState(util.ConvState(status), util.ConvReason(status, reason))
		list = append(list, domain)
		dom.Free()
	}

	return list, nil
}
