package fuzz

import (
	"fmt"
	"reflect"
	"testing"

	gofuzz "github.com/google/gofuzz"
	admissionv1 "k8s.io/api/admission/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type testCase struct {
	name      string
	fuzzFuncs []interface{}
	gvk       v12.GroupVersionResource
	objType   interface{}
	admit     func(config *virtconfig.ClusterConfig, request *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse
	debug     bool
}

// FuzzAdmitter tests the Validation webhook execution logic with random input: It does schema validation (syntactic check), followed by executing the domain specific validation logic (semantic checks).
func FuzzAdmitter(f *testing.F) {
	testCases := []testCase{
		{
			name:    "VirtualMachineInstance",
			gvk:     webhooks.VirtualMachineInstanceGroupVersionResource,
			objType: &v1.VirtualMachineInstance{},
			admit: func(config *virtconfig.ClusterConfig, request *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
				adm := &admitters.VMICreateAdmitter{ClusterConfig: config}
				return adm.Admit(request)
			},
		},
		{
			name:    "VirtualMachine",
			gvk:     webhooks.VirtualMachineGroupVersionResource,
			objType: &v1.VirtualMachine{},
			admit: func(config *virtconfig.ClusterConfig, request *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
				adm := &admitters.VMsAdmitter{ClusterConfig: config}
				return adm.Admit(request)
			},
		},
	}
	f.Add(int64(1), int64(1))
	f.Add(int64(2), int64(2))
	f.Add(int64(3), int64(3))
	f.Fuzz(func(t *testing.T, objSeed int64, configSeed int64) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				newObj := reflect.New(reflect.TypeOf(tc.objType))
				obj := newObj.Interface()

				fuzzFuncs := append(defaultFuzzFuncs(), tc.fuzzFuncs...)
				gofuzz.NewWithSeed(objSeed).NumElements(0, 15).Funcs(
					fuzzFuncs...,
				).Fuzz(obj)
				request := toAdmissionReview(obj, tc.gvk)
				config := fuzzKubeVirtConfig(configSeed)
				response := tc.admit(config, request)
				if tc.debug && !response.Allowed {
					fmt.Println(response.Result.Message)
				}
			})
		}
	})
}

func toAdmissionReview(obj interface{}, gvr v12.GroupVersionResource) *admissionv1.AdmissionReview {
	raw, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource: gvr,
			Object: runtime.RawExtension{
				Raw: raw,
			},
		},
	}
}

func fuzzKubeVirtConfig(seed int64) *virtconfig.ClusterConfig {
	kv := &v1.KubeVirt{}
	gofuzz.NewWithSeed(seed).Funcs(
		func(dc *v1.DeveloperConfiguration, c gofuzz.Continue) {
			c.FuzzNoCustom(dc)
			featureGates := []string{
				virtconfig.ExpandDisksGate,
				virtconfig.CPUManager,
				virtconfig.NUMAFeatureGate,
				virtconfig.IgnitionGate,
				virtconfig.LiveMigrationGate,
				virtconfig.SRIOVLiveMigrationGate,
				virtconfig.CPUNodeDiscoveryGate,
				virtconfig.HypervStrictCheckGate,
				virtconfig.SidecarGate,
				virtconfig.GPUGate,
				virtconfig.HostDevicesGate,
				virtconfig.SnapshotGate,
				virtconfig.VMExportGate,
				virtconfig.HotplugVolumesGate,
				virtconfig.HostDiskGate,
				virtconfig.VirtIOFSGate,
				virtconfig.MacvtapGate,
				virtconfig.PasstGate,
				virtconfig.DownwardMetricsFeatureGate,
				virtconfig.NonRootDeprecated,
				virtconfig.NonRoot,
				virtconfig.Root,
				virtconfig.ClusterProfiler,
				virtconfig.WorkloadEncryptionSEV,
				virtconfig.DockerSELinuxMCSWorkaround,
				virtconfig.PSA,
				virtconfig.VSOCKGate,
			}

			idxs := c.Perm(c.Int() % len(featureGates))
			for idx := range idxs {
				dc.FeatureGates = append(dc.FeatureGates, featureGates[idx])
			}
		},
	).Fuzz(kv)
	config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)
	return config
}

func defaultFuzzFuncs() []interface{} {
	return []interface{}{
		func(e *v12.FieldsV1, c gofuzz.Continue) {},
		func(vmi *v1.VirtualMachineInstance, c gofuzz.Continue) {
			c.FuzzNoCustom(vmi)
			vmi.Spec.TerminationGracePeriodSeconds = nil
		},
	}
}
