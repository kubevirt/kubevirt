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
	"encoding/json"
	"fmt"
	"time"

	libvirtxml "libvirt.org/go/libvirtxml"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/translate"
	celutil "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins/cel"
)

type hookApplier interface {
	Apply(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain, invocationContext string) (*libvirtxml.Domain, error)
}

type celHookApplier struct {
	evaluator  *celutil.Evaluator
	expression string
}

func (c *celHookApplier) Apply(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain, _ string) (*libvirtxml.Domain, error) {
	return c.evaluator.EvaluateMutation(c.expression, vmi, domain)
}

type sidecarHookApplier struct {
	socketPath       string
	pluginName       string
	timeout          time.Duration
	deadline         time.Time
	readinessTimeout time.Duration
}

func (s *sidecarHookApplier) Apply(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain, invocationContext string) (*libvirtxml.Domain, error) {
	if err := waitForSidecarSocket(s.socketPath, s.deadline, s.readinessTimeout); err != nil {
		return nil, err
	}

	domainXML, err := domain.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal domain: %w", err)
	}
	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		return nil, fmt.Errorf("marshal VMI: %w", err)
	}

	resultXML, err := callSidecarHook(s.socketPath, s.pluginName, []byte(domainXML), vmiJSON, invocationContext, s.timeout)
	if err != nil {
		return nil, err
	}

	mutated := &libvirtxml.Domain{}
	if err := mutated.Unmarshal(string(resultXML)); err != nil {
		return nil, fmt.Errorf("unmarshal sidecar response: %w", err)
	}
	return mutated, nil
}

func ApplyGuestDefinitionHooks(plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, spec *virtwrapApi.DomainSpec, invocationContext pluginv1alpha1.InvocationContext) (*virtwrapApi.DomainSpec, string, error) {
	return applyGuestDefinitionHooks(plugins, vmi, spec, invocationContext, defaultSidecarReadinessTimeout)
}

func applyGuestDefinitionHooks(plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, spec *virtwrapApi.DomainSpec, invocationContext pluginv1alpha1.InvocationContext, readinessTimeout time.Duration) (*virtwrapApi.DomainSpec, string, error) {
	if len(plugins) == 0 {
		return spec, "", nil
	}

	domain, err := translate.ToLibvirtDomain(spec)
	if err != nil {
		return nil, "", fmt.Errorf("converting DomainSpec to libvirtxml: %w", err)
	}

	evaluator := celutil.GetEvaluator()
	evalCondition := func(expr string) (bool, error) {
		return evaluator.EvaluateCondition(expr, vmi, domain)
	}

	resolvedHooks, err := selectHooks(plugins, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
	if err != nil {
		return nil, "", err
	}

	// No plugin contributed a guest definition hook (e.g. a plugin only declares NodeHooks).
	// Return the spec untouched; an empty XML string signals to the caller that no hooks ran
	// and it should keep its own domain XML rather than adopting this round-trip.
	if len(resolvedHooks) == 0 {
		return spec, "", nil
	}

	// All sidecars are containers of the same virt-launcher pod and start at roughly the same
	// time, so readiness is bounded once for the whole pipeline rather than per hook. Otherwise
	// N sidecars could delay startup by N*readinessTimeout and the first hook would be
	// judged most strictly. This is the readiness budget only; per-hook execution time is
	// bounded separately by the hook timeout below.
	sidecarReadinessDeadline := time.Now().Add(readinessTimeout)

	for _, resolved := range resolvedHooks {
		var applier hookApplier
		switch {
		case resolved.hook.CEL != nil:
			applier = &celHookApplier{evaluator: evaluator, expression: resolved.hook.CEL.Expression}
		case resolved.hook.Sidecar != nil:
			deadline := sidecarReadinessDeadline
			timeout := defaultSidecarCallTimeout
			if resolved.timeout != nil {
				timeout = resolved.timeout.Duration
			}
			applier = &sidecarHookApplier{
				socketPath:       resolved.hook.Sidecar.SocketPath,
				pluginName:       resolved.pluginName,
				timeout:          timeout,
				deadline:         deadline,
				readinessTimeout: readinessTimeout,
			}
		default:
			return nil, "", fmt.Errorf("plugin %s hook %d defines neither cel nor sidecar", resolved.pluginName, resolved.index)
		}

		mutated, err := applier.Apply(vmi, domain, string(invocationContext))
		if err != nil {
			if resolved.failureStrategy == pluginv1alpha1.FailureStrategyIgnore {
				log.Log.Warningf("Plugin %s hook %d failed (ignored): %v", resolved.pluginName, resolved.index, err)
				continue
			}
			return nil, "", fmt.Errorf("plugin %s hook %d failed: %w", resolved.pluginName, resolved.index, err)
		}
		domain = mutated
	}

	xmlStr, err := domain.Marshal()
	if err != nil {
		return nil, "", fmt.Errorf("marshaling domain to XML: %w", err)
	}

	updatedSpec, err := translate.FromLibvirtDomain(domain)
	if err != nil {
		return nil, "", fmt.Errorf("converting domain back to DomainSpec: %w", err)
	}

	log.Log.Infof("Successfully applied %d guest definition hook(s)", len(resolvedHooks))
	return updatedSpec, xmlStr, nil
}
