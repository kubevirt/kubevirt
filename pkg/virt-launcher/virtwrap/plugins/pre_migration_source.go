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
 * Copyright The KubeVirt Authors.
 *
 */

package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	pluginsv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
)

// ApplyPreMigrationSourceHooks lets plugins rewrite the migration flags and
// parameters before they are handed to libvirt. An ignored failure leaves the
// last successful configuration intact.
func ApplyPreMigrationSourceHooks(ctx context.Context, plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, flags libvirt.DomainMigrateFlags, params *libvirt.DomainMigrateParameters) (libvirt.DomainMigrateFlags, *libvirt.DomainMigrateParameters, error) {
	return applyPreMigrationSourceHooks(ctx, plugins, vmi, flags, params, defaultSidecarCallOptions())
}

func applyPreMigrationSourceHooks(ctx context.Context, plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, flags libvirt.DomainMigrateFlags, params *libvirt.DomainMigrateParameters, opts sidecarCallOptions) (libvirt.DomainMigrateFlags, *libvirt.DomainMigrateParameters, error) {
	if len(plugins) == 0 {
		return flags, params, nil
	}

	hooks, err := selectHooks(plugins, vmi, pluginv1alpha1.LauncherHookPreMigrationSource)
	if err != nil {
		return flags, params, err
	}
	if len(hooks) == 0 {
		return flags, params, nil
	}

	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		return flags, params, fmt.Errorf("marshal VMI: %w", err)
	}

	currentFlags, currentParams := flags, params

	err = runHooks(ctx, pluginv1alpha1.LauncherHookPreMigrationSource, hooks, func(ctx context.Context, hook resolvedHook) error {
		// PreMigrationSource is sidecar-only, so selectHooks never returns a CEL hook here.
		return callSidecar(ctx, hook, opts, func(ctx context.Context, client pluginsv1alpha1.LauncherHookServiceClient) error {
			response, err := client.PreMigrationSource(ctx, &pluginsv1alpha1.PreMigrationSourceRequest{
				Vmi: vmiJSON,
				MigrationConfig: &pluginsv1alpha1.PreMigrationSourceRequest_Libvirt{
					Libvirt: libvirtConfigToProto(currentFlags, currentParams),
				},
			})
			if err != nil {
				return err
			}
			// The response must echo the request's migration configuration variant.
			responseConfig, ok := response.MigrationConfig.(*pluginsv1alpha1.PreMigrationSourceResponse_Libvirt)
			if !ok || responseConfig.Libvirt == nil {
				return fmt.Errorf("unsupported migration configuration variant %T", response.MigrationConfig)
			}

			currentFlags, currentParams = libvirtConfigFromProto(responseConfig.Libvirt)
			return nil
		})
	})
	return currentFlags, currentParams, err
}

// libvirtConfigToProto converts libvirt's migration flags and parameters to the
// protobuf shape exchanged with PreMigrationSource hooks.
func libvirtConfigToProto(flags libvirt.DomainMigrateFlags, params *libvirt.DomainMigrateParameters) *pluginsv1alpha1.LibvirtMigrationConfig {
	return &pluginsv1alpha1.LibvirtMigrationConfig{
		Flags:  uint64(flags),
		Params: migrateParamsToProto(params),
	}
}

// libvirtConfigFromProto is the inverse of libvirtConfigToProto.
func libvirtConfigFromProto(config *pluginsv1alpha1.LibvirtMigrationConfig) (libvirt.DomainMigrateFlags, *libvirt.DomainMigrateParameters) {
	return libvirt.DomainMigrateFlags(config.Flags), migrateParamsFromProto(config.Params)
}

// migrateParamsToProto converts every non-deprecated migration parameter,
// translating libvirt's XxxSet presence flags into set wrapper messages, so
// hooks may also set parameters KubeVirt itself never sets. The exhaustiveness
// test in pre_migration_source_test.go keeps the field list explicit.
func migrateParamsToProto(params *libvirt.DomainMigrateParameters) *pluginsv1alpha1.LibvirtDomainMigrateParameters {
	protoParams := &pluginsv1alpha1.LibvirtDomainMigrateParameters{}
	if params == nil {
		return protoParams
	}
	if params.URISet {
		protoParams.Uri = &pluginsv1alpha1.OptionalString{Value: params.URI}
	}
	if params.DestNameSet {
		protoParams.DestName = &pluginsv1alpha1.OptionalString{Value: params.DestName}
	}
	if params.DestXMLSet {
		protoParams.DestXml = &pluginsv1alpha1.OptionalString{Value: params.DestXML}
	}
	if params.PersistXMLSet {
		protoParams.PersistXml = &pluginsv1alpha1.OptionalString{Value: params.PersistXML}
	}
	if params.BandwidthSet {
		protoParams.Bandwidth = &pluginsv1alpha1.OptionalUint64{Value: params.Bandwidth}
	}
	if params.ParallelConnectionsSet {
		protoParams.ParallelConnections = &pluginsv1alpha1.OptionalInt32{Value: int32(params.ParallelConnections)}
	}
	if params.MigrateDisksSet {
		protoParams.MigrateDisks = params.MigrateDisks
	}
	if params.MigrateDisksDetectZeroesSet {
		protoParams.MigrateDisksDetectZeroesList = params.MigrateDisksDetectZeroesList
	}
	if params.DisksURISet {
		protoParams.DisksUri = &pluginsv1alpha1.OptionalString{Value: params.DisksURI}
	}
	if params.CompressionSet {
		protoParams.Compression = &pluginsv1alpha1.OptionalString{Value: params.Compression}
	}
	if params.GraphicsURISet {
		protoParams.GraphicsUri = &pluginsv1alpha1.OptionalString{Value: params.GraphicsURI}
	}
	if params.ListenAddressSet {
		protoParams.ListenAddress = &pluginsv1alpha1.OptionalString{Value: params.ListenAddress}
	}
	if params.DisksPortSet {
		protoParams.DisksPort = &pluginsv1alpha1.OptionalInt32{Value: int32(params.DisksPort)}
	}
	if params.CompressionMTLevelSet {
		protoParams.CompressionMtLevel = &pluginsv1alpha1.OptionalInt32{Value: int32(params.CompressionMTLevel)}
	}
	if params.CompressionMTThreadsSet {
		protoParams.CompressionMtThreads = &pluginsv1alpha1.OptionalInt32{Value: int32(params.CompressionMTThreads)}
	}
	if params.CompressionMTDThreadsSet {
		protoParams.CompressionMtDthreads = &pluginsv1alpha1.OptionalInt32{Value: int32(params.CompressionMTDThreads)}
	}
	if params.CompressionXBZRLECacheSet {
		protoParams.CompressionXbzrleCache = &pluginsv1alpha1.OptionalUint64{Value: params.CompressionXBZRLECache}
	}
	if params.AutoConvergeInitialSet {
		protoParams.AutoConvergeInitial = &pluginsv1alpha1.OptionalInt32{Value: int32(params.AutoConvergeInitial)}
	}
	if params.AutoConvergeIncrementSet {
		protoParams.AutoConvergeIncrement = &pluginsv1alpha1.OptionalInt32{Value: int32(params.AutoConvergeIncrement)}
	}
	if params.TLSDestinationSet {
		protoParams.TlsDestination = &pluginsv1alpha1.OptionalString{Value: params.TLSDestination}
	}
	if params.CompressionZlibLevelSet {
		protoParams.CompressionZlibLevel = &pluginsv1alpha1.OptionalInt32{Value: int32(params.CompressionZlibLevel)}
	}
	if params.CompressionZstdLevelSet {
		protoParams.CompressionZstdLevel = &pluginsv1alpha1.OptionalInt32{Value: int32(params.CompressionZstdLevel)}
	}
	if params.BandwidthAvailSwitchoverSet {
		protoParams.BandwidthAvailSwitchover = &pluginsv1alpha1.OptionalUint64{Value: params.BandwidthAvailSwitchover}
	}
	if params.MigrateDisksTargetZeroSet {
		protoParams.MigrateDisksTargetZero = params.MigrateDisksTargetZero
	}
	return protoParams
}

// migrateParamsFromProto is the inverse of migrateParamsToProto: a set wrapper
// message becomes a value with its XxxSet flag, an unset one stays unset.
func migrateParamsFromProto(protoParams *pluginsv1alpha1.LibvirtDomainMigrateParameters) *libvirt.DomainMigrateParameters {
	params := &libvirt.DomainMigrateParameters{}
	if protoParams == nil {
		return params
	}
	if protoParams.Uri != nil {
		params.URI, params.URISet = protoParams.Uri.Value, true
	}
	if protoParams.DestName != nil {
		params.DestName, params.DestNameSet = protoParams.DestName.Value, true
	}
	if protoParams.DestXml != nil {
		params.DestXML, params.DestXMLSet = protoParams.DestXml.Value, true
	}
	if protoParams.PersistXml != nil {
		params.PersistXML, params.PersistXMLSet = protoParams.PersistXml.Value, true
	}
	if protoParams.Bandwidth != nil {
		params.Bandwidth, params.BandwidthSet = protoParams.Bandwidth.Value, true
	}
	if protoParams.ParallelConnections != nil {
		params.ParallelConnections, params.ParallelConnectionsSet = int(protoParams.ParallelConnections.Value), true
	}
	if len(protoParams.MigrateDisks) > 0 {
		params.MigrateDisks, params.MigrateDisksSet = protoParams.MigrateDisks, true
	}
	if len(protoParams.MigrateDisksDetectZeroesList) > 0 {
		params.MigrateDisksDetectZeroesList, params.MigrateDisksDetectZeroesSet = protoParams.MigrateDisksDetectZeroesList, true
	}
	if protoParams.DisksUri != nil {
		params.DisksURI, params.DisksURISet = protoParams.DisksUri.Value, true
	}
	if protoParams.Compression != nil {
		params.Compression, params.CompressionSet = protoParams.Compression.Value, true
	}
	if protoParams.GraphicsUri != nil {
		params.GraphicsURI, params.GraphicsURISet = protoParams.GraphicsUri.Value, true
	}
	if protoParams.ListenAddress != nil {
		params.ListenAddress, params.ListenAddressSet = protoParams.ListenAddress.Value, true
	}
	if protoParams.DisksPort != nil {
		params.DisksPort, params.DisksPortSet = int(protoParams.DisksPort.Value), true
	}
	if protoParams.CompressionMtLevel != nil {
		params.CompressionMTLevel, params.CompressionMTLevelSet = int(protoParams.CompressionMtLevel.Value), true
	}
	if protoParams.CompressionMtThreads != nil {
		params.CompressionMTThreads, params.CompressionMTThreadsSet = int(protoParams.CompressionMtThreads.Value), true
	}
	if protoParams.CompressionMtDthreads != nil {
		params.CompressionMTDThreads, params.CompressionMTDThreadsSet = int(protoParams.CompressionMtDthreads.Value), true
	}
	if protoParams.CompressionXbzrleCache != nil {
		params.CompressionXBZRLECache, params.CompressionXBZRLECacheSet = protoParams.CompressionXbzrleCache.Value, true
	}
	if protoParams.AutoConvergeInitial != nil {
		params.AutoConvergeInitial, params.AutoConvergeInitialSet = int(protoParams.AutoConvergeInitial.Value), true
	}
	if protoParams.AutoConvergeIncrement != nil {
		params.AutoConvergeIncrement, params.AutoConvergeIncrementSet = int(protoParams.AutoConvergeIncrement.Value), true
	}
	if protoParams.TlsDestination != nil {
		params.TLSDestination, params.TLSDestinationSet = protoParams.TlsDestination.Value, true
	}
	if protoParams.CompressionZlibLevel != nil {
		params.CompressionZlibLevel, params.CompressionZlibLevelSet = int(protoParams.CompressionZlibLevel.Value), true
	}
	if protoParams.CompressionZstdLevel != nil {
		params.CompressionZstdLevel, params.CompressionZstdLevelSet = int(protoParams.CompressionZstdLevel.Value), true
	}
	if protoParams.BandwidthAvailSwitchover != nil {
		params.BandwidthAvailSwitchover, params.BandwidthAvailSwitchoverSet = protoParams.BandwidthAvailSwitchover.Value, true
	}
	if len(protoParams.MigrateDisksTargetZero) > 0 {
		params.MigrateDisksTargetZero, params.MigrateDisksTargetZeroSet = protoParams.MigrateDisksTargetZero, true
	}
	return params
}
