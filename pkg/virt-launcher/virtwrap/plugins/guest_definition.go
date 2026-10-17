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

	libvirtxml "libvirt.org/go/libvirtxml"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	pluginsv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/translate"
	celutil "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins/cel"
)

func ApplyGuestDefinitionHooks(plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, spec *virtwrapApi.DomainSpec, invocationContext pluginv1alpha1.InvocationContext) (*virtwrapApi.DomainSpec, string, error) {
	return applyGuestDefinitionHooks(plugins, vmi, spec, invocationContext, defaultSidecarCallOptions())
}

func applyGuestDefinitionHooks(plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, spec *virtwrapApi.DomainSpec, invocationContext pluginv1alpha1.InvocationContext, opts sidecarCallOptions) (*virtwrapApi.DomainSpec, string, error) {
	if len(plugins) == 0 {
		return spec, "", nil
	}

	domain, err := translate.ToLibvirtDomain(spec)
	if err != nil {
		return nil, "", fmt.Errorf("converting DomainSpec to libvirtxml: %w", err)
	}

	hooks, err := selectHooks(plugins, vmi, pluginv1alpha1.LauncherHookGuestDefinition)
	if err != nil {
		return nil, "", err
	}

	// No plugin contributed a guest definition hook (e.g. a plugin only declares NodeHooks).
	// Return the spec untouched; an empty XML string signals to the caller that no hooks ran
	// and it should keep its own domain XML rather than adopting this round-trip.
	if len(hooks) == 0 {
		return spec, "", nil
	}

	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		return nil, "", fmt.Errorf("marshal VMI: %w", err)
	}

	evaluator := celutil.GetEvaluator()

	err = runHooks(context.Background(), pluginv1alpha1.LauncherHookGuestDefinition, hooks, func(ctx context.Context, hook resolvedHook) error {
		if hook.hook.CEL != nil {
			mutated, err := evaluator.EvaluateMutation(hook.hook.CEL.Expression, vmi, domain)
			if err != nil {
				return err
			}
			domain = mutated
			return nil
		}
		if hook.hook.Sidecar == nil {
			return fmt.Errorf("plugin %s hook %d defines neither cel nor sidecar", hook.pluginName, hook.index)
		}

		domainXML, err := domain.Marshal()
		if err != nil {
			return fmt.Errorf("marshal domain: %w", err)
		}

		return callSidecar(ctx, hook, opts, func(ctx context.Context, client pluginsv1alpha1.LauncherHookServiceClient) error {
			resp, err := client.GuestDefinition(ctx, &pluginsv1alpha1.GuestDefinitionRequest{
				DomainType: domainTypeLibvirt,
				Domain:     []byte(domainXML),
				Vmi:        vmiJSON,
				SidecarContext: &pluginsv1alpha1.SidecarContext{
					InvocationContext: string(invocationContext),
				},
			})
			if err != nil {
				return err
			}

			mutated := &libvirtxml.Domain{}
			if err := mutated.Unmarshal(string(resp.Domain)); err != nil {
				return fmt.Errorf("unmarshal sidecar response: %w", err)
			}
			domain = mutated
			return nil
		})
	})
	if err != nil {
		return nil, "", err
	}

	xmlStr, err := domain.Marshal()
	if err != nil {
		return nil, "", fmt.Errorf("marshaling domain to XML: %w", err)
	}

	updatedSpec, err := translate.FromLibvirtDomain(domain)
	if err != nil {
		return nil, "", fmt.Errorf("converting domain back to DomainSpec: %w", err)
	}

	log.Log.Infof("Processed %d guest definition hook(s)", len(hooks))
	return updatedSpec, xmlStr, nil
}
