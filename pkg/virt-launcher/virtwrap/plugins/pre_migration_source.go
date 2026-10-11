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

// ApplyPreMigrationSourceHooks lets plugins change the migration flags and
// parameters before they are handed to libvirt. An ignored failure leaves the
// last successful configuration intact.
func ApplyPreMigrationSourceHooks(ctx context.Context, plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, flags libvirt.DomainMigrateFlags, params *libvirt.DomainMigrateParameters) (libvirt.DomainMigrateFlags, *libvirt.DomainMigrateParameters, error) {
	return applyPreMigrationSourceHooks(ctx, plugins, vmi, flags, params, defaultSidecarCallOptions())
}

// PreMigrationSourceHooksSelected reports whether at least one PreMigrationSource hook
// is selected for vmi from the current plugin snapshot. Callers use it to decide whether
// the PreMigrationSource phase will actually run; a failed selection reports false
// because no hook would run then either.
func PreMigrationSourceHooksSelected(vmi *v1.VirtualMachineInstance) bool {
	pluginSnapshot := GetPlugins()
	if len(pluginSnapshot) == 0 {
		return false
	}
	hooks, err := selectHooks(pluginSnapshot, vmi, pluginv1alpha1.LauncherHookPreMigrationSource)
	return err == nil && len(hooks) > 0
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

	for _, hook := range hooks {
		// PreMigrationSource is sidecar-only, so selectHooks never returns a CEL hook here.
		newFlags, newParams, hookErr := applyPreMigrationSourceHook(ctx, hook, vmiJSON, currentFlags, currentParams, opts)
		if err := handleHookError(ctx, pluginv1alpha1.LauncherHookPreMigrationSource, hook, hookErr); err != nil {
			return currentFlags, currentParams, err
		}
		if hookErr == nil {
			currentFlags, currentParams = newFlags, newParams
		}
	}
	return currentFlags, currentParams, nil
}

// applyPreMigrationSourceHook sends one PreMigrationSource RPC carrying the current
// configuration and returns the configuration with the hook's response applied to it.
func applyPreMigrationSourceHook(ctx context.Context, hook resolvedHook, vmiJSON []byte, flags libvirt.DomainMigrateFlags, params *libvirt.DomainMigrateParameters, opts sidecarCallOptions) (libvirt.DomainMigrateFlags, *libvirt.DomainMigrateParameters, error) {
	conn, err := dialSidecar(ctx, hook, opts)
	if err != nil {
		return flags, params, err
	}
	defer conn.Close()

	callCtx, cancel := context.WithTimeout(ctx, hook.timeout)
	defer cancel()

	response, err := pluginsv1alpha1.NewLauncherHookServiceClient(conn).PreMigrationSource(callCtx, &pluginsv1alpha1.PreMigrationSourceRequest{
		Vmi: vmiJSON,
		MigrationConfig: &pluginsv1alpha1.PreMigrationSourceRequest_Libvirt{
			Libvirt: libvirtConfigToProto(flags, params),
		},
	})
	if err != nil {
		return flags, params, sidecarRPCError(err)
	}

	// A response variant unknown to this KubeVirt decodes as an unset oneof (the nil case) and is ignored; any other mismatched variant fails loudly.
	switch config := response.GetMigrationConfig().(type) {
	case nil:
		// An empty response changes nothing.
		return flags, params, nil
	case *pluginsv1alpha1.PreMigrationSourceResponse_Libvirt:
		mergedFlags, mergedParams, mergeErr := mergeLibvirtConfig(flags, params, config.Libvirt)
		if mergeErr != nil {
			return flags, params, mergeErr
		}
		return mergedFlags, mergedParams, nil
	default:
		return flags, params, fmt.Errorf("unsupported migration configuration variant %T", config)
	}
}

// libvirtConfigToProto converts libvirt's migration flags and parameters to the
// protobuf shape exchanged with PreMigrationSource hooks.
func libvirtConfigToProto(flags libvirt.DomainMigrateFlags, params *libvirt.DomainMigrateParameters) *pluginsv1alpha1.LibvirtMigrationConfig {
	return &pluginsv1alpha1.LibvirtMigrationConfig{
		Flags:  uint64(flags),
		Params: migrateParamsToProto(params),
	}
}

// migrateParamsToProto converts every non-deprecated migration parameter,
// translating libvirt's XxxSet presence flags into set wrapper messages, so
// hooks may also set parameters KubeVirt itself never sets. The exhaustiveness
// test in pre_migration_source_test.go keeps the field list explicit.
func migrateParamsToProto(params *libvirt.DomainMigrateParameters) *pluginsv1alpha1.LibvirtDomainMigrateParameters {
	if params == nil {
		return &pluginsv1alpha1.LibvirtDomainMigrateParameters{}
	}
	return &pluginsv1alpha1.LibvirtDomainMigrateParameters{
		Uri:                          optionalString(params.URI, params.URISet),
		DestName:                     optionalString(params.DestName, params.DestNameSet),
		DestXml:                      optionalString(params.DestXML, params.DestXMLSet),
		PersistXml:                   optionalString(params.PersistXML, params.PersistXMLSet),
		Bandwidth:                    optionalUint64(params.Bandwidth, params.BandwidthSet),
		ParallelConnections:          optionalInt32(params.ParallelConnections, params.ParallelConnectionsSet),
		MigrateDisks:                 stringList(params.MigrateDisks, params.MigrateDisksSet),
		MigrateDisksDetectZeroesList: stringList(params.MigrateDisksDetectZeroesList, params.MigrateDisksDetectZeroesSet),
		DisksUri:                     optionalString(params.DisksURI, params.DisksURISet),
		Compression:                  optionalString(params.Compression, params.CompressionSet),
		GraphicsUri:                  optionalString(params.GraphicsURI, params.GraphicsURISet),
		ListenAddress:                optionalString(params.ListenAddress, params.ListenAddressSet),
		DisksPort:                    optionalInt32(params.DisksPort, params.DisksPortSet),
		CompressionMtLevel:           optionalInt32(params.CompressionMTLevel, params.CompressionMTLevelSet),
		CompressionMtThreads:         optionalInt32(params.CompressionMTThreads, params.CompressionMTThreadsSet),
		CompressionMtDthreads:        optionalInt32(params.CompressionMTDThreads, params.CompressionMTDThreadsSet),
		CompressionXbzrleCache:       optionalUint64(params.CompressionXBZRLECache, params.CompressionXBZRLECacheSet),
		AutoConvergeInitial:          optionalInt32(params.AutoConvergeInitial, params.AutoConvergeInitialSet),
		AutoConvergeIncrement:        optionalInt32(params.AutoConvergeIncrement, params.AutoConvergeIncrementSet),
		TlsDestination:               optionalString(params.TLSDestination, params.TLSDestinationSet),
		CompressionZlibLevel:         optionalInt32(params.CompressionZlibLevel, params.CompressionZlibLevelSet),
		CompressionZstdLevel:         optionalInt32(params.CompressionZstdLevel, params.CompressionZstdLevelSet),
		BandwidthAvailSwitchover:     optionalUint64(params.BandwidthAvailSwitchover, params.BandwidthAvailSwitchoverSet),
		MigrateDisksTargetZero:       stringList(params.MigrateDisksTargetZero, params.MigrateDisksTargetZeroSet),
	}
}

// mergeLibvirtConfig applies a hook response to the current configuration: set_flags adds
// bits to the flags and clear_flags removes them (a bit present in both is invalid); for
// the parameters, an unset field keeps the current value, a set wrapper overrides it and
// an empty wrapper removes it.
func mergeLibvirtConfig(flags libvirt.DomainMigrateFlags, params *libvirt.DomainMigrateParameters, changes *pluginsv1alpha1.LibvirtMigrationConfigChanges) (libvirt.DomainMigrateFlags, *libvirt.DomainMigrateParameters, error) {
	if changes.GetSetFlags()&changes.GetClearFlags() != 0 {
		return flags, params, fmt.Errorf("invalid migration flag change: bits 0b%b are both set and cleared", changes.GetSetFlags()&changes.GetClearFlags())
	}
	flags = (flags | libvirt.DomainMigrateFlags(changes.GetSetFlags())) &^ libvirt.DomainMigrateFlags(changes.GetClearFlags())

	merged := libvirt.DomainMigrateParameters{}
	if params != nil {
		merged = *params
	}
	paramChanges := changes.GetParams()
	mergeString(paramChanges.GetUri(), &merged.URI, &merged.URISet)
	mergeString(paramChanges.GetDestName(), &merged.DestName, &merged.DestNameSet)
	mergeString(paramChanges.GetDestXml(), &merged.DestXML, &merged.DestXMLSet)
	mergeString(paramChanges.GetPersistXml(), &merged.PersistXML, &merged.PersistXMLSet)
	mergeUint64(paramChanges.GetBandwidth(), &merged.Bandwidth, &merged.BandwidthSet)
	mergeInt(paramChanges.GetParallelConnections(), &merged.ParallelConnections, &merged.ParallelConnectionsSet)
	mergeStrings(paramChanges.GetMigrateDisks(), &merged.MigrateDisks, &merged.MigrateDisksSet)
	mergeStrings(paramChanges.GetMigrateDisksDetectZeroesList(), &merged.MigrateDisksDetectZeroesList, &merged.MigrateDisksDetectZeroesSet)
	mergeString(paramChanges.GetDisksUri(), &merged.DisksURI, &merged.DisksURISet)
	mergeString(paramChanges.GetCompression(), &merged.Compression, &merged.CompressionSet)
	mergeString(paramChanges.GetGraphicsUri(), &merged.GraphicsURI, &merged.GraphicsURISet)
	mergeString(paramChanges.GetListenAddress(), &merged.ListenAddress, &merged.ListenAddressSet)
	mergeInt(paramChanges.GetDisksPort(), &merged.DisksPort, &merged.DisksPortSet)
	mergeInt(paramChanges.GetCompressionMtLevel(), &merged.CompressionMTLevel, &merged.CompressionMTLevelSet)
	mergeInt(paramChanges.GetCompressionMtThreads(), &merged.CompressionMTThreads, &merged.CompressionMTThreadsSet)
	mergeInt(paramChanges.GetCompressionMtDthreads(), &merged.CompressionMTDThreads, &merged.CompressionMTDThreadsSet)
	mergeUint64(paramChanges.GetCompressionXbzrleCache(), &merged.CompressionXBZRLECache, &merged.CompressionXBZRLECacheSet)
	mergeInt(paramChanges.GetAutoConvergeInitial(), &merged.AutoConvergeInitial, &merged.AutoConvergeInitialSet)
	mergeInt(paramChanges.GetAutoConvergeIncrement(), &merged.AutoConvergeIncrement, &merged.AutoConvergeIncrementSet)
	mergeString(paramChanges.GetTlsDestination(), &merged.TLSDestination, &merged.TLSDestinationSet)
	mergeInt(paramChanges.GetCompressionZlibLevel(), &merged.CompressionZlibLevel, &merged.CompressionZlibLevelSet)
	mergeInt(paramChanges.GetCompressionZstdLevel(), &merged.CompressionZstdLevel, &merged.CompressionZstdLevelSet)
	mergeUint64(paramChanges.GetBandwidthAvailSwitchover(), &merged.BandwidthAvailSwitchover, &merged.BandwidthAvailSwitchoverSet)
	mergeStrings(paramChanges.GetMigrateDisksTargetZero(), &merged.MigrateDisksTargetZero, &merged.MigrateDisksTargetZeroSet)
	return flags, &merged, nil
}

// optionalString returns a wrapper holding value when set is true, nil otherwise.
func optionalString(value string, set bool) *pluginsv1alpha1.OptionalString {
	if !set {
		return nil
	}
	return &pluginsv1alpha1.OptionalString{Kind: &pluginsv1alpha1.OptionalString_Value{Value: value}}
}

// optionalUint64 returns a wrapper holding value when set is true, nil otherwise.
func optionalUint64(value uint64, set bool) *pluginsv1alpha1.OptionalUint64 {
	if !set {
		return nil
	}
	return &pluginsv1alpha1.OptionalUint64{Kind: &pluginsv1alpha1.OptionalUint64_Value{Value: value}}
}

// optionalInt32 returns a wrapper holding value when set is true, nil otherwise.
func optionalInt32(value int, set bool) *pluginsv1alpha1.OptionalInt32 {
	if !set {
		return nil
	}
	return &pluginsv1alpha1.OptionalInt32{Kind: &pluginsv1alpha1.OptionalInt32_Value{Value: int32(value)}}
}

// stringList returns a list of values when set is true and the list is non-empty, nil
// otherwise: libvirt-go drops an empty string list, so a set-but-empty one would reach
// libvirt as absent.
func stringList(values []string, set bool) *pluginsv1alpha1.StringList {
	if !set || len(values) == 0 {
		return nil
	}
	return &pluginsv1alpha1.StringList{Values: values}
}

// mergeString applies a string change to value and set: an unset change keeps them, a set one
// overrides both, and an empty one removes the parameter.
func mergeString(change *pluginsv1alpha1.OptionalString, value *string, set *bool) {
	if change != nil {
		*value, *set = change.GetValue(), change.GetKind() != nil
	}
}

// mergeUint64 applies a uint64 change to value and set: an unset change keeps them, a set one
// overrides both, and an empty one removes the parameter.
func mergeUint64(change *pluginsv1alpha1.OptionalUint64, value *uint64, set *bool) {
	if change != nil {
		*value, *set = change.GetValue(), change.GetKind() != nil
	}
}

// mergeInt applies an int change to value and set, converting from the proto int32: an unset
// change keeps them, a set one overrides both, and an empty one removes the parameter.
func mergeInt(change *pluginsv1alpha1.OptionalInt32, value *int, set *bool) {
	if change != nil {
		*value, *set = int(change.GetValue()), change.GetKind() != nil
	}
}

// mergeStrings applies a list change to values and set: an unset change keeps them, a non-empty
// one overrides both, and an empty one removes the parameter.
func mergeStrings(change *pluginsv1alpha1.StringList, values *[]string, set *bool) {
	if change != nil {
		*values, *set = change.GetValues(), len(change.GetValues()) > 0
	}
}
