package virt_handler

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	afero "github.com/spf13/afero"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	// args: QOS class, QOS class, pod UID (with _ instead of -)
	pathFormat = "/node/sys/fs/cgroup/kubepods.slice/kubepods-%s.slice/kubepods-%s-pod%s.slice/"

	// PSI metrics
	PSIMemoryPressure = "memory.pressure"
	PSICpuPressure    = "cpu.pressure"
	PSIIoPressure     = "io.pressure"
)

var (
	PSIMetricsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			psiMemoryPressureMetric,
			psiCpuPressureMetric,
			psiIoPressureMetric,
		},
		CollectCallback: PSIMetricsCollectorCallback,
	}

	psiMemoryPressureMetric = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_pressure_seconds",
			Help: "Time tasks were delayed due to lack of memory resources. " +
				"Type can be some or full (some is the walltime for some (one or more) tasks while full is the walltime for all tasks).",
		},
		[]string{"namespace", "name", "type"},
	)

	psiCpuPressureMetric = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_cpu_pressure_seconds",
			Help: "Time tasks were delayed due to lack of CPU resources. " +
				"Type can be some or full (some is the walltime for some (one or more) tasks while full is the walltime for all tasks).",
		},
		[]string{"namespace", "name", "type"},
	)

	psiIoPressureMetric = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_io_pressure_seconds",
			Help: "Time tasks were delayed due to lack of IO resources. " +
				"Type can be some or full (some is the walltime for some (one or more) tasks while full is the walltime for all tasks).",
		},
		[]string{"namespace", "name", "type"},
	)
)

type PSIMetrics struct {
	FS   afero.Fs
	VMIs []*k6tv1.VirtualMachineInstance
}

type PSIData struct {
	Avg10  float64 `json:"avg10"`
	Avg60  float64 `json:"avg60"`
	Avg300 float64 `json:"avg300"`
	Total  uint64  `json:"total"`
}

type PSIStats struct {
	Some PSIData `json:"some,omitempty"`
	Full PSIData `json:"full,omitempty"`
}

func PSIMetricsCollectorCallback() []operatormetrics.CollectorResult {
	cachedObjs := vmiInformer.GetIndexer().List()
	if len(cachedObjs) == 0 {
		log.Log.V(4).Infof("No VMIs detected")
		return []operatormetrics.CollectorResult{}
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	PSIMetrics := &PSIMetrics{
		FS:   afero.NewOsFs(),
		VMIs: vmis,
	}

	return PSIMetrics.CollectPSIMetricsFromVMIs()
}

func (p *PSIMetrics) CollectPSIMetricsFromVMIs() []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult

	for _, vmi := range p.VMIs {
		for podUid := range vmi.Status.ActivePods {
			qosClass := strings.ToLower(string(*vmi.Status.QOSClass))
			results = append(results, p.collectPSIMetricsFromPod(vmi, string(podUid), qosClass)...)
		}
	}

	return results
}

func (p *PSIMetrics) collectPSIMetricsFromPod(vmi *k6tv1.VirtualMachineInstance, podUid string, qosClass string) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult

	filePath := path.Join(fmt.Sprintf(pathFormat, qosClass, qosClass, strings.ReplaceAll(podUid, "-", "_")))

	psiMemoryPressureValue, err := p.statPSI(filePath, PSIMemoryPressure)
	if err != nil {
		log.Log.Errorf("Failed to collect PSI memory pressure metrics: %v", err)
	} else if psiMemoryPressureValue == nil {
		log.Log.V(4).Infof("No PSI memory pressure metrics available")
	} else {
		results = append(results, operatormetrics.CollectorResult{
			Metric: psiMemoryPressureMetric,
			Labels: []string{vmi.Namespace, vmi.Name, "some"},
			Value:  microSecondsToSeconds(psiMemoryPressureValue.Some.Total),
		})
		results = append(results, operatormetrics.CollectorResult{
			Metric: psiMemoryPressureMetric,
			Labels: []string{vmi.Namespace, vmi.Name, "full"},
			Value:  microSecondsToSeconds(psiMemoryPressureValue.Full.Total),
		})
	}

	psiCpuPressureValue, err := p.statPSI(filePath, PSICpuPressure)
	if err != nil {
		log.Log.Errorf("Failed to collect PSI CPU pressure metrics: %v", err)
	} else if psiCpuPressureValue == nil {
		log.Log.V(4).Infof("No PSI CPU pressure metrics available")
	} else {
		results = append(results, operatormetrics.CollectorResult{
			Metric: psiCpuPressureMetric,
			Labels: []string{vmi.Namespace, vmi.Name, "some"},
			Value:  microSecondsToSeconds(psiCpuPressureValue.Some.Total),
		})
		results = append(results, operatormetrics.CollectorResult{
			Metric: psiCpuPressureMetric,
			Labels: []string{vmi.Namespace, vmi.Name, "full"},
			Value:  microSecondsToSeconds(psiCpuPressureValue.Full.Total),
		})
	}

	psiIoPressureValue, err := p.statPSI(filePath, PSIIoPressure)
	if err != nil {
		log.Log.Errorf("Failed to collect PSI IO pressure metrics: %v", err)
	} else if psiIoPressureValue == nil {
		log.Log.V(4).Infof("No PSI IO pressure metrics available")
	} else {
		results = append(results, operatormetrics.CollectorResult{
			Metric: psiIoPressureMetric,
			Labels: []string{vmi.Namespace, vmi.Name, "some"},
			Value:  microSecondsToSeconds(psiIoPressureValue.Some.Total),
		})
		results = append(results, operatormetrics.CollectorResult{
			Metric: psiIoPressureMetric,
			Labels: []string{vmi.Namespace, vmi.Name, "full"},
			Value:  microSecondsToSeconds(psiIoPressureValue.Full.Total),
		})
	}

	return results
}

func (p *PSIMetrics) statPSI(dirPath string, file string) (*PSIStats, error) {
	f, err := p.FS.OpenFile(path.Join(dirPath, file), os.O_RDONLY, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Kernel < 4.20, or CONFIG_PSI is not set,
			// or PSI stats are turned off for the cgroup
			// ("echo 0 > cgroup.pressure", kernel >= 6.1).
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var psistats PSIStats
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		var pv *PSIData
		switch parts[0] {
		case "some":
			pv = &psistats.Some
		case "full":
			pv = &psistats.Full
		}
		if pv != nil {
			*pv, err = parsePSIData(parts[1:])
			if err != nil {
				return nil, fmt.Errorf("failed to parse PSI data: %w", err)
			}
		}
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, unix.ENOTSUP) {
			// Some kernels (e.g. CS9) may return ENOTSUP on read
			// if psi=1 kernel cmdline parameter is required.
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read PSI stats: %w", err)
	}
	return &psistats, nil
}

func parsePSIData(psi []string) (PSIData, error) {
	data := PSIData{}
	for _, f := range psi {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) != 2 {
			return data, fmt.Errorf("invalid psi data: %q", f)
		}
		var pv *float64
		switch kv[0] {
		case "avg10":
			pv = &data.Avg10
		case "avg60":
			pv = &data.Avg60
		case "avg300":
			pv = &data.Avg300
		case "total":
			v, err := strconv.ParseUint(kv[1], 10, 64)
			if err != nil {
				return data, fmt.Errorf("invalid %s PSI value: %w", kv[0], err)
			}
			data.Total = v
		}
		if pv != nil {
			v, err := strconv.ParseFloat(kv[1], 64)
			if err != nil {
				return data, fmt.Errorf("invalid %s PSI value: %w", kv[0], err)
			}
			*pv = v
		}
	}
	return data, nil
}

func microSecondsToSeconds(microSeconds uint64) float64 {
	return float64(microSeconds) / 1e6
}
