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

package admitters

import (
	"context"
	"encoding/json"
	"runtime"
	"slices"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/defaults"
	ephemeraldiskfake "kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/libvmi"
	netadmitter "kubevirt.io/kubevirt/pkg/network/admitter"
	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

func TestRegisteredFeatureGateNames(t *testing.T) {
	const registeredForTest = "FuzzRegisteredFeatureGate"
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{
		Name:  registeredForTest,
		State: featuregate.Alpha,
	})
	t.Cleanup(func() {
		featuregate.UnregisterFeatureGate(registeredForTest)
	})

	names := registeredFeatureGateNames()
	if !slices.IsSorted(names) {
		t.Fatalf("feature gate names are not sorted: %v", names)
	}
	if !slices.Contains(names, registeredForTest) {
		t.Fatalf("registered feature gate %q is absent from %v", registeredForTest, names)
	}
}

type fuzzTestReporter interface {
	Helper()
	Fatalf(format string, args ...any)
}

func FuzzVMIAdmission(f *testing.F) {
	clusterConfig := newFuzzClusterConfig(nil)
	vmi := libvmi.New(
		libvmi.WithName("fuzz-vmi"),
		libvmi.WithNamespace("default"),
		libvmi.WithMemoryRequest("128Mi"),
		libvmi.WithArchitecture(runtime.GOARCH),
	)
	if err := defaults.SetDefaultVirtualMachineInstance(clusterConfig, vmi); err != nil {
		f.Fatalf("failed to default the seed VMI: %v", err)
	}
	seed, err := json.Marshal(vmi)
	if err != nil {
		f.Fatalf("failed to marshal the seed VMI: %v", err)
	}

	dedicatedVMI := libvmi.New(
		libvmi.WithName("dedicated-fuzz-vmi"),
		libvmi.WithNamespace("default"),
		libvmi.WithMemoryRequest("128Mi"),
		libvmi.WithArchitecture(runtime.GOARCH),
		libvmi.WithCPUCount(2, 1, 1),
		libvmi.WithCPURequest("2"),
		libvmi.WithCPULimit("2"),
		libvmi.WithDedicatedCPUPlacement(),
	)
	if err := defaults.SetDefaultVirtualMachineInstance(clusterConfig, dedicatedVMI); err != nil {
		f.Fatalf("failed to default the dedicated CPU seed VMI: %v", err)
	}
	dedicatedSeed, err := json.Marshal(dedicatedVMI)
	if err != nil {
		f.Fatalf("failed to marshal the dedicated CPU seed VMI: %v", err)
	}

	assertAdmittedVMIConverts(f, seed, clusterConfig)
	assertAdmittedVMIConverts(f, dedicatedSeed, clusterConfig)

	gates := registeredFeatureGateNames()
	f.Add(seed, uint64(0))
	f.Add(dedicatedSeed, uint64(0))
	f.Add([]byte("{"), uint64(0))
	for i := range gates {
		f.Add(seed, uint64(i+1))
	}

	f.Fuzz(func(t *testing.T, raw []byte, gateIndex uint64) {
		var enabledGates []string
		if gateIndex > 0 && len(gates) > 0 {
			enabledGates = []string{gates[(gateIndex-1)%uint64(len(gates))]}
		}
		clusterConfig := newFuzzClusterConfig(enabledGates)

		response, accepted := admitFuzzVMI(t, raw, clusterConfig)
		if !response.Allowed {
			return
		}

		domain, err := convertFuzzVMI(accepted)
		if err != nil {
			// Conversion also depends on node- and launcher-derived state that is
			// deliberately absent here. Reaching the real converter without a
			// panic is the property for arbitrary admitted inputs.
			return
		}
		if domain.Spec.Name == "" {
			t.Fatal("domain conversion succeeded without a domain name")
		}
	})
}

func admitFuzzVMI(t fuzzTestReporter, raw []byte, clusterConfig *virtconfig.ClusterConfig) (*admissionv1.AdmissionResponse, *v1.VirtualMachineInstance) {
	t.Helper()
	admitter := &VMICreateAdmitter{
		ClusterConfig:           clusterConfig,
		KubeVirtServiceAccounts: webhooks.KubeVirtServiceAccounts("kubevirt"),
		SpecValidators: []SpecValidator{
			func(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
				return netadmitter.Validate(field, spec, config)
			},
		},
	}
	response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource:  webhooks.VirtualMachineInstanceGroupVersionResource,
			Operation: admissionv1.Create,
			Object: k8sruntime.RawExtension{
				Raw: raw,
			},
		},
	})
	if response == nil {
		t.Fatalf("admission returned a nil response")
	}
	if !response.Allowed {
		return response, nil
	}

	accepted := &v1.VirtualMachineInstance{}
	if err := json.Unmarshal(raw, accepted); err != nil {
		t.Fatalf("admission accepted an undecodable VMI: %v", err)
	}
	if err := defaults.SetDefaultVirtualMachineInstance(clusterConfig, accepted); err != nil {
		t.Fatalf("admission accepted a VMI that cannot be defaulted: %v", err)
	}
	return response, accepted
}

func assertAdmittedVMIConverts(t fuzzTestReporter, raw []byte, clusterConfig *virtconfig.ClusterConfig) {
	t.Helper()
	response, accepted := admitFuzzVMI(t, raw, clusterConfig)
	if !response.Allowed {
		message := ""
		if response.Result != nil {
			message = response.Result.Message
		}
		t.Fatalf("known-valid VMI seed was rejected by admission: %s", message)
	}

	domain, err := convertFuzzVMI(accepted)
	if err != nil {
		t.Fatalf("known-valid admitted VMI cannot be converted: %v", err)
	}
	if domain.Spec.Name == "" {
		t.Fatalf("known-valid admitted VMI converted without a domain name")
	}
}

func convertFuzzVMI(vmi *v1.VirtualMachineInstance) (*api.Domain, error) {
	domain := &api.Domain{}
	err := converter.Convert_v1_VirtualMachineInstance_To_api_Domain(
		vmi,
		domain,
		newFuzzConverterContext(vmi),
	)
	return domain, err
}

func newFuzzConverterContext(vmi *v1.VirtualMachineInstance) *convertertypes.ConverterContext {
	context := &convertertypes.ConverterContext{
		Architecture:                    archconverter.NewConverter(vmi.Spec.Architecture),
		AllowEmulation:                  true,
		HypervisorDeviceAvailable:       true,
		IsBlockPVC:                      map[string]bool{},
		IsBlockDV:                       map[string]bool{},
		ApplyCBT:                        map[string]string{},
		HotplugVolumes:                  map[string]v1.VolumeStatus{},
		PermanentVolumes:                map[string]v1.VolumeStatus{},
		MigratedVolumes:                 map[string]string{},
		DisksInfo:                       map[string]*disk.DiskInfo{},
		EphemeraldiskCreator:            &ephemeraldiskfake.MockEphemeralDiskImageCreator{},
		DomainAttachmentByInterfaceName: map[string]string{},
		HostArchitecture:                runtime.GOARCH,
		Topology:                        &cmdv1.Topology{},
	}

	if !vmi.IsCPUDedicated() {
		return context
	}

	const maxSyntheticCPUs = 1024
	cpuCount := hardware.GetDedicatedHostCPUs(vmi)
	if cpuCount <= 0 || cpuCount > maxSyntheticCPUs {
		return context
	}

	context.CPUSet = make([]int, cpuCount)
	cpus := make([]*cmdv1.CPU, cpuCount)
	for i := range context.CPUSet {
		context.CPUSet[i] = i
		cpus[i] = &cmdv1.CPU{Id: uint32(i)}
	}
	context.Topology.NumaCells = []*cmdv1.Cell{{Id: 0, Cpus: cpus}}
	return context
}

func newFuzzClusterConfig(enabledGates []string) *virtconfig.ClusterConfig {
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{
			FeatureGates: enabledGates,
		},
	})
	return config
}

func registeredFeatureGateNames() []string {
	registered := featuregate.GetRegisteredFeatureGates()
	names := make([]string, 0, len(registered))
	for name := range registered {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
