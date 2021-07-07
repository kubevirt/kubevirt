package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strings"
)

const (
	counterLabelCompName = "component_name"
	counterLabelAnnName  = "annotation_name"
)

// HcoMetrics wrapper for all hco metrics
var HcoMetrics = hcoMetrics{
	overwrittenModifications: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubevirt_hco_out_of_band_modifications_count",
			Help: "Count of out-of-band modifications overwritten by HCO",
		},
		[]string{counterLabelCompName},
	),
	unsafeModifications: prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubevirt_hco_unsafe_modification_count",
			Help: "count of unsafe modifications in the HyperConverged annotations",
		},
		[]string{counterLabelAnnName},
	),
}

// hcoMetrics holds all HCO metrics
type hcoMetrics struct {
	// overwrittenModifications counts out-of-band modifications overwritten by HCO
	overwrittenModifications *prometheus.CounterVec

	// unsafeModifications counts the modifications done using the jsonpatch annotations
	unsafeModifications *prometheus.GaugeVec
}

func init() {
	HcoMetrics.init()
}

func (hm *hcoMetrics) init() {
	metrics.Registry.MustRegister(hm.overwrittenModifications, hm.unsafeModifications)
}

// IncOverwrittenModifications increments counter by 1
func (hm *hcoMetrics) IncOverwrittenModifications(kind, name string) {
	hm.overwrittenModifications.With(getLabelsForObj(kind, name)).Inc()
}

// GetOverwrittenModificationsCount returns current value of counter. If error is not nil then value is undefined
func (hm *hcoMetrics) GetOverwrittenModificationsCount(kind, name string) (float64, error) {
	var m = &dto.Metric{}
	err := hm.overwrittenModifications.With(getLabelsForObj(kind, name)).Write(m)
	return m.Counter.GetValue(), err
}

// SetUnsafeModificationCount sets the counter to the required number
func (hm *hcoMetrics) SetUnsafeModificationCount(count int, unsafeAnnotation string) {
	hm.unsafeModifications.With(getLabelsForUnsafeAnnotation(unsafeAnnotation)).Set(float64(count))
}

// GetUnsafeModificationsCount returns current value of counter. If error is not nil then value is undefined
func (hm *hcoMetrics) GetUnsafeModificationsCount(unsafeAnnotation string) (float64, error) {
	var m = &dto.Metric{}
	err := hm.unsafeModifications.With(getLabelsForUnsafeAnnotation(unsafeAnnotation)).Write(m)
	return m.Gauge.GetValue(), err
}

func getLabelsForObj(kind string, name string) prometheus.Labels {
	return prometheus.Labels{counterLabelCompName: strings.ToLower(kind + "/" + name)}
}

func getLabelsForUnsafeAnnotation(unsafeAnnotation string) prometheus.Labels {
	return prometheus.Labels{counterLabelAnnName: strings.ToLower(unsafeAnnotation)}
}
