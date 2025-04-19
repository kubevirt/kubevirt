package virt_handler_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spf13/afero"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k6tv1 "kubevirt.io/api/core/v1"

	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
)

var _ = Describe("PSI Metrics Collector", func() {
	var (
		psiMetrics *metrics.PSIMetrics
		mockFs     afero.Fs
		vmi        *k6tv1.VirtualMachineInstance
	)

	BeforeEach(func() {
		mockFs = afero.NewMemMapFs()
		vmi = &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testvmi",
				Namespace: "testns",
			},
			Status: k6tv1.VirtualMachineInstanceStatus{
				QOSClass: &[]k8sv1.PodQOSClass{k8sv1.PodQOSGuaranteed}[0],
				ActivePods: map[types.UID]string{
					"test-pod-uid": "test-pod",
				},
			},
		}

		// Create test directory structure
		podPath := fmt.Sprintf("/node/sys/fs/cgroup/kubepods.slice/kubepods-%s.slice/kubepods-%s-pod%s.slice/",
			"guaranteed", "guaranteed", "test_pod_uid")
		Expect(mockFs.MkdirAll(podPath, 0755)).To(Succeed())

		// Create test PSI files with sample content
		memoryContent := "some avg10=0.00 avg60=0.00 avg300=0.00 total=0\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n"
		cpuContent := "some avg10=1.23 avg60=0.45 avg300=0.67 total=1000000\nfull avg10=0.12 avg60=0.34 avg300=0.56 total=500000\n"
		ioContent := "some avg10=2.34 avg60=1.56 avg300=0.78 total=2000000\nfull avg10=1.23 avg60=0.45 avg300=0.67 total=1500000\n"

		Expect(afero.WriteFile(mockFs, podPath+metrics.PSIMemoryPressure, []byte(memoryContent), 0644)).To(Succeed())
		Expect(afero.WriteFile(mockFs, podPath+metrics.PSICpuPressure, []byte(cpuContent), 0644)).To(Succeed())
		Expect(afero.WriteFile(mockFs, podPath+metrics.PSIIoPressure, []byte(ioContent), 0644)).To(Succeed())

		psiMetrics = &metrics.PSIMetrics{
			FS:   mockFs,
			VMIs: []*k6tv1.VirtualMachineInstance{vmi},
		}
	})

	It("should collect PSI metrics from all VMIs", func() {
		results := psiMetrics.CollectPSIMetricsFromVMIs()

		// We expect 6 metrics (2 for each of memory, CPU, and IO - one for "some" and one for "full")
		Expect(results).To(HaveLen(6))

		memSomeFound := false
		memFullFound := false

		cpuSomeFound := false
		cpuFullFound := false

		ioSomeFound := false
		ioFullFound := false

		for _, result := range results {
			labels := result.Labels
			Expect(labels[0]).To(Equal("testns"))
			Expect(labels[1]).To(Equal("testvmi"))

			switch {
			case result.Metric.GetOpts().Name == "kubevirt_vmi_memory_pressure_seconds" && labels[2] == "some":
				memSomeFound = true
				Expect(result.Value).To(BeZero())
			case result.Metric.GetOpts().Name == "kubevirt_vmi_memory_pressure_seconds" && labels[2] == "full":
				memFullFound = true
				Expect(result.Value).To(BeZero())
			case result.Metric.GetOpts().Name == "kubevirt_vmi_cpu_pressure_seconds" && labels[2] == "some":
				cpuSomeFound = true
				Expect(result.Value).To(Equal(1.0))
			case result.Metric.GetOpts().Name == "kubevirt_vmi_cpu_pressure_seconds" && labels[2] == "full":
				cpuFullFound = true
				Expect(result.Value).To(Equal(0.5))
			case result.Metric.GetOpts().Name == "kubevirt_vmi_io_pressure_seconds" && labels[2] == "some":
				ioSomeFound = true
				Expect(result.Value).To(Equal(2.0))
			case result.Metric.GetOpts().Name == "kubevirt_vmi_io_pressure_seconds" && labels[2] == "full":
				ioFullFound = true
				Expect(result.Value).To(Equal(1.5))
			}
		}

		Expect(memSomeFound).To(BeTrue(), "Memory 'some' pressure metric not found")
		Expect(memFullFound).To(BeTrue(), "Memory 'full' pressure metric not found")
		Expect(cpuSomeFound).To(BeTrue(), "CPU 'some' pressure metric not found")
		Expect(cpuFullFound).To(BeTrue(), "CPU 'full' pressure metric not found")
		Expect(ioSomeFound).To(BeTrue(), "IO 'some' pressure metric not found")
		Expect(ioFullFound).To(BeTrue(), "IO 'full' pressure metric not found")
	})

	It("should handle missing PSI files", func() {
		// Remove the PSI files
		podPath := fmt.Sprintf("/node/sys/fs/cgroup/kubepods.slice/kubepods-%s.slice/kubepods-%s-pod%s.slice/",
			"guaranteed", "guaranteed", "test_pod_uid")
		Expect(mockFs.Remove(podPath + metrics.PSIMemoryPressure)).To(Succeed())
		Expect(mockFs.Remove(podPath + metrics.PSICpuPressure)).To(Succeed())
		Expect(mockFs.Remove(podPath + metrics.PSIIoPressure)).To(Succeed())

		results := psiMetrics.CollectPSIMetricsFromVMIs()
		Expect(results).To(BeEmpty())
	})

	It("should handle multiple VMIs and pods", func() {
		vmi2 := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testvmi2",
				Namespace: "testns",
			},
			Status: k6tv1.VirtualMachineInstanceStatus{
				QOSClass: &[]k8sv1.PodQOSClass{k8sv1.PodQOSBurstable}[0],
				ActivePods: map[types.UID]string{
					"pod-uid-1": "pod1",
					"pod-uid-2": "pod2",
				},
			},
		}
		psiMetrics.VMIs = append(psiMetrics.VMIs, vmi2)

		for _, podUid := range []string{"pod_uid_1", "pod_uid_2"} {
			podPath := fmt.Sprintf("/node/sys/fs/cgroup/kubepods.slice/kubepods-%s.slice/kubepods-%s-pod%s.slice/",
				"burstable", "burstable", podUid)
			Expect(mockFs.MkdirAll(podPath, 0755)).To(Succeed())

			content := "some avg10=0.00 avg60=0.00 avg300=0.00 total=1000000\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=500000\n"
			Expect(afero.WriteFile(mockFs, podPath+metrics.PSIMemoryPressure, []byte(content), 0644)).To(Succeed())
			Expect(afero.WriteFile(mockFs, podPath+metrics.PSICpuPressure, []byte(content), 0644)).To(Succeed())
			Expect(afero.WriteFile(mockFs, podPath+metrics.PSIIoPressure, []byte(content), 0644)).To(Succeed())
		}

		results := psiMetrics.CollectPSIMetricsFromVMIs()

		Expect(results).To(HaveLen(18))
	})
})
