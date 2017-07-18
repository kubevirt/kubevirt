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

package designer

import (
	"fmt"
	"net"
	"strings"

	"github.com/jeevatkm/go-model"
	k8sv1 "k8s.io/api/core/v1"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

func convertDeviceDiskPVCISCSI(vm *v1.VM, src *v1.Disk, pv *k8sv1.PersistentVolume) (*api.Disk, error) {
	logging.DefaultLogger().Object(vm).Info().Msg("Mapping iSCSI PVC")

	dst := &api.Disk{}
	dst.Type = "network"
	dst.Device = "disk"
	dst.Target = api.DiskTarget{
		Bus:    src.Target.Bus,
		Device: src.Target.Device,
	}
	dst.Driver = new(api.DiskDriver)
	dst.Driver.Type = "raw"
	dst.Driver.Name = "qemu"

	dst.Source.Name = fmt.Sprintf("%s/%d", pv.Spec.ISCSI.IQN, pv.Spec.ISCSI.Lun)
	dst.Source.Protocol = "iscsi"

	hostPort := strings.Split(pv.Spec.ISCSI.TargetPortal, ":")
	ipAddrs, err := net.LookupIP(hostPort[0])
	if err != nil || len(ipAddrs) < 1 {
		logging.DefaultLogger().Error().Reason(err).Msgf("Unable to resolve host '%s'", hostPort[0])
		return nil, fmt.Errorf("Unable to resolve host '%s': %s", hostPort[0], err)
	}

	dst.Source.Host = &api.DiskSourceHost{}
	dst.Source.Host.Name = ipAddrs[0].String()
	if len(hostPort) > 1 {
		dst.Source.Host.Port = hostPort[1]
	}

	return dst, nil
}

func convertDeviceDiskPVC(vm *v1.VM, src *v1.Disk, restClient cache.Getter) (*api.Disk, error) {
	logging.DefaultLogger().V(3).Info().Object(vm).Msgf("Mapping PersistentVolumeClaim: %s", src.Source.Name)

	// Look up existing persistent volume
	obj, err := restClient.Get().Namespace(vm.ObjectMeta.Namespace).Resource("persistentvolumeclaims").Name(src.Source.Name).Do().Get()

	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("unable to look up persistent volume claim")
		return nil, fmt.Errorf("unable to look up persistent volume claim: %v", err)
	}

	pvc := obj.(*k8sv1.PersistentVolumeClaim)
	if pvc.Status.Phase != k8sv1.ClaimBound {
		logging.DefaultLogger().Error().Msg("attempted use of unbound persistent volume")
		return nil, fmt.Errorf("attempted use of unbound persistent volume claim: %s", pvc.Name)
	}

	// Look up the PersistentVolume this PVC is bound to
	// Note: This call is not namespaced!
	obj, err = restClient.Get().Resource("persistentvolumes").Name(pvc.Spec.VolumeName).Do().Get()

	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("unable to access persistent volume record")
		return nil, fmt.Errorf("unable to access persistent volume record: %v", err)
	}
	pv := obj.(*k8sv1.PersistentVolume)

	if pv.Spec.ISCSI != nil {
		return convertDeviceDiskPVCISCSI(vm, src, pv)
	} else {
		logging.DefaultLogger().Object(vm).Error().Msg(fmt.Sprintf("Referenced PV %v is backed by an unsupported storage type", pv))
		return nil, fmt.Errorf("Referenced PV %v is backed by an unsupported storage type", pv)
	}
}

func convertDeviceDiskNetwork(vm *v1.VM, src *v1.Disk, restClient cache.Getter) (*api.Disk, error) {
	dst := &api.Disk{}
	model.Copy(dst, src)

	if src.Source.Host == nil {
		logging.DefaultLogger().Error().Msg("Missing disk source host")
		return nil, fmt.Errorf("Missing disk source host")
	}

	ipAddrs, err := net.LookupIP(src.Source.Host.Name)
	if err != nil || ipAddrs == nil || len(ipAddrs) < 1 {
		logging.DefaultLogger().Error().Reason(err).Msgf("Unable to resolve host '%s'", src.Source.Host.Name)
		return nil, fmt.Errorf("Unable to resolve host '%s': %s", src.Source.Host.Name, err)
	}

	dst.Source.Host.Name = ipAddrs[0].String()

	return dst, nil
}

func convertDeviceDisk(vm *v1.VM, src *v1.Disk, restClient cache.Getter) (*api.Disk, error) {

	if src.Type == "PersistentVolumeClaim" {
		return convertDeviceDiskPVC(vm, src, restClient)
	} else if src.Type == "network" {
		return convertDeviceDiskNetwork(vm, src, restClient)
	} else {
		logging.DefaultLogger().Error().Msgf("Unsupported disk source type %s", src.Type)
		return nil, fmt.Errorf("Unsupported disk source type %s", src.Type)
	}
}

func MapDomainSpec(vm *v1.VM, restClient cache.Getter) (*api.DomainSpec, error) {
	var domSpec api.DomainSpec
	mappingErrs := model.Copy(&domSpec, vm.Spec.Domain)
	if len(mappingErrs) > 0 {
		return nil, errutil.NewAggregate(mappingErrs)
	}

	for idx, disk := range vm.Spec.Domain.Devices.Disks {
		dst, err := convertDeviceDisk(vm, &disk, restClient)
		if err != nil {
			return nil, err
		}

		domSpec.Devices.Disks[idx] = *dst
	}

	return &domSpec, nil
}
