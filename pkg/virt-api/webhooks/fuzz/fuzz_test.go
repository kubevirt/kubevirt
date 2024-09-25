package fuzz

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	gofuzz "github.com/google/gofuzz"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/deprecation"
)

type fuzzOption int

const withSyntaxErrors fuzzOption = 1

type testCase struct {
	name      string
	fuzzFuncs []interface{}
	gvk       metav1.GroupVersionResource
	objType   interface{}
	admit     func(config *virtconfig.ClusterConfig, request *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse
	debug     bool
	focus     bool
}

// FuzzAdmitter tests the Validation webhook execution logic with random input: It does schema validation (syntactic check), followed by executing the domain specific validation logic (semantic checks).
func FuzzAdmitter(f *testing.F) {
	testCases := []testCase{
		{
			name:    "SyntacticVirtualMachineInstanceFuzzing",
			gvk:     webhooks.VirtualMachineInstanceGroupVersionResource,
			objType: &v1.VirtualMachineInstance{},
			admit: func(config *virtconfig.ClusterConfig, request *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
				adm := &admitters.VMICreateAdmitter{ClusterConfig: config}
				return adm.Admit(context.Background(), request)
			},
			fuzzFuncs: fuzzFuncs(withSyntaxErrors),
		},
		{
			name:    "SemanticVirtualMachineInstanceFuzzing",
			gvk:     webhooks.VirtualMachineInstanceGroupVersionResource,
			objType: &v1.VirtualMachineInstance{},
			admit: func(config *virtconfig.ClusterConfig, request *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
				adm := &admitters.VMICreateAdmitter{ClusterConfig: config}
				return adm.Admit(context.Background(), request)
			},
			fuzzFuncs: fuzzFuncs(),
		},
		{
			name:    "SyntacticVirtualMachineFuzzing",
			gvk:     webhooks.VirtualMachineGroupVersionResource,
			objType: &v1.VirtualMachine{},
			admit: func(config *virtconfig.ClusterConfig, request *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
				adm := &admitters.VMsAdmitter{
					ClusterConfig:       config,
					InstancetypeMethods: testutils.NewMockInstancetypeMethods(),
				}
				return adm.Admit(context.Background(), request)
			},
			fuzzFuncs: fuzzFuncs(withSyntaxErrors),
		},
		{
			name:    "SemanticVirtualMachineFuzzing",
			gvk:     webhooks.VirtualMachineGroupVersionResource,
			objType: &v1.VirtualMachine{},
			admit: func(config *virtconfig.ClusterConfig, request *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
				adm := &admitters.VMsAdmitter{
					ClusterConfig:       config,
					InstancetypeMethods: testutils.NewMockInstancetypeMethods(),
				}
				return adm.Admit(context.Background(), request)
			},
			fuzzFuncs: fuzzFuncs(),
		},
	}

	for i := 0; i < 500; i++ {
		f.Add(int64(i))
	}
	f.Fuzz(func(t *testing.T, seed int64) {
		var focused []testCase
		for idx, tc := range testCases {
			if tc.focus {
				focused = append(focused, testCases[idx])
			}
		}

		if len(focused) == 0 {
			focused = testCases
		}

		timeoutDuration := 500 * time.Millisecond

		for _, tc := range focused {
			t.Run(tc.name, func(t *testing.T) {
				newObj := reflect.New(reflect.TypeOf(tc.objType))
				obj := newObj.Interface()

				gofuzz.NewWithSeed(seed).NilChance(0.1).NumElements(0, 15).Funcs(
					tc.fuzzFuncs...,
				).Fuzz(obj)
				request := toAdmissionReview(obj, tc.gvk)
				config := fuzzKubeVirtConfig(seed)
				startTime := time.Now()
				response := tc.admit(config, request)
				endTime := time.Now()
				if startTime.Add(timeoutDuration).Before(endTime) {
					fmt.Printf("Execution time %v is more than %v\n", endTime.Sub(startTime), timeoutDuration)
					fmt.Println(response.Result.Message)
					j, err := json.MarshalIndent(obj, "", "  ")
					if err != nil {
						panic(err)
					}
					fmt.Println(string(j))
					t.Fail()
				}

				if tc.debug && !response.Allowed {
					fmt.Println(response.Result.Message)
					j, err := json.MarshalIndent(obj, "", "  ")
					if err != nil {
						panic(err)
					}
					fmt.Println(string(j))
				}
			})
		}
	})
}

func toAdmissionReview(obj interface{}, gvr metav1.GroupVersionResource) *admissionv1.AdmissionReview {
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
				deprecation.NUMAFeatureGate,
				virtconfig.IgnitionGate,
				deprecation.LiveMigrationGate,
				deprecation.SRIOVLiveMigrationGate,
				deprecation.CPUNodeDiscoveryGate,
				virtconfig.HypervStrictCheckGate,
				virtconfig.SidecarGate,
				virtconfig.GPUGate,
				virtconfig.HostDevicesGate,
				virtconfig.SnapshotGate,
				virtconfig.VMExportGate,
				virtconfig.HotplugVolumesGate,
				virtconfig.HostDiskGate,
				virtconfig.VirtIOFSGate,
				deprecation.MacvtapGate,
				deprecation.PasstGate,
				virtconfig.DownwardMetricsFeatureGate,
				deprecation.NonRoot,
				virtconfig.Root,
				virtconfig.ClusterProfiler,
				virtconfig.WorkloadEncryptionSEV,
				deprecation.DockerSELinuxMCSWorkaround,
				deprecation.PSA,
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

func fuzzFuncs(options ...fuzzOption) []interface{} {
	addSyntaxErrors := false
	for _, opt := range options {
		if opt == withSyntaxErrors {
			addSyntaxErrors = true
		}
	}

	enumFuzzers := []interface{}{
		func(e *metav1.FieldsV1, c gofuzz.Continue) {},
		func(objectmeta *metav1.ObjectMeta, c gofuzz.Continue) {
			c.FuzzNoCustom(objectmeta)
			objectmeta.DeletionGracePeriodSeconds = nil
			objectmeta.Generation = 0
			objectmeta.ManagedFields = nil
		},
		func(obj *corev1.URIScheme, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.URIScheme{corev1.URISchemeHTTP, corev1.URISchemeHTTPS}, c)
		},
		func(obj *corev1.TaintEffect, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.TaintEffect{corev1.TaintEffectNoExecute, corev1.TaintEffectNoSchedule, corev1.TaintEffectPreferNoSchedule}, c)
		},
		func(obj *corev1.NodeInclusionPolicy, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.NodeInclusionPolicy{corev1.NodeInclusionPolicyHonor, corev1.NodeInclusionPolicyIgnore}, c)
		},
		func(obj *corev1.UnsatisfiableConstraintAction, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.UnsatisfiableConstraintAction{corev1.DoNotSchedule, corev1.ScheduleAnyway}, c)
		},
		func(obj *corev1.PullPolicy, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.PullPolicy{corev1.PullAlways, corev1.PullNever, corev1.PullIfNotPresent}, c)
		},
		func(obj *corev1.NodeSelectorOperator, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.NodeSelectorOperator{corev1.NodeSelectorOpDoesNotExist, corev1.NodeSelectorOpExists, corev1.NodeSelectorOpGt, corev1.NodeSelectorOpIn, corev1.NodeSelectorOpLt, corev1.NodeSelectorOpNotIn}, c)
		},
		func(obj *corev1.TolerationOperator, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.TolerationOperator{corev1.TolerationOpExists, corev1.TolerationOpEqual}, c)
		},
		func(obj *corev1.PodQOSClass, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.PodQOSClass{corev1.PodQOSBestEffort, corev1.PodQOSGuaranteed, corev1.PodQOSBurstable}, c)
		},
		func(obj *corev1.PersistentVolumeMode, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.PersistentVolumeMode{corev1.PersistentVolumeBlock, corev1.PersistentVolumeFilesystem}, c)
		},
		func(obj *corev1.DNSPolicy, c gofuzz.Continue) {
			pickType(addSyntaxErrors, obj, []corev1.DNSPolicy{corev1.DNSClusterFirst, corev1.DNSClusterFirstWithHostNet, corev1.DNSDefault, corev1.DNSNone}, c)
		},
		func(obj *corev1.TypedObjectReference, c gofuzz.Continue) {
			c.FuzzNoCustom(obj)
			str := c.RandString()
			obj.APIGroup = &str
		},
		func(obj *corev1.TypedLocalObjectReference, c gofuzz.Continue) {
			c.FuzzNoCustom(obj)
			str := c.RandString()
			obj.APIGroup = &str
		},
	}

	typeFuzzers := []interface{}{}
	if !addSyntaxErrors {
		typeFuzzers = []interface{}{
			func(obj *int, c gofuzz.Continue) {
				*obj = c.Intn(100000)
			},
			func(obj *uint, c gofuzz.Continue) {
				*obj = uint(c.Intn(100000))
			},
			func(obj *int32, c gofuzz.Continue) {
				*obj = int32(c.Intn(100000))
			},
			func(obj *int64, c gofuzz.Continue) {
				*obj = int64(c.Intn(100000))
			},
			func(obj *uint64, c gofuzz.Continue) {
				*obj = uint64(c.Intn(100000))
			},
			func(obj *uint32, c gofuzz.Continue) {
				*obj = uint32(c.Intn(100000))
			},
		}
	}

	return append(enumFuzzers, typeFuzzers...)
}

func pickType(withSyntaxError bool, target interface{}, arr interface{}, c gofuzz.Continue) {
	arrPtr := reflect.ValueOf(arr)
	targetPtr := reflect.ValueOf(target)

	if withSyntaxError {
		arrPtr = reflect.Append(arrPtr, reflect.ValueOf("fake").Convert(targetPtr.Elem().Type()))
	}

	idx := c.Int() % arrPtr.Len()

	targetPtr.Elem().Set(arrPtr.Index(idx))
}
