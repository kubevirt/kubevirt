package metrics

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	counterLabelCompName = "component_name"
	counterLabelAnnName  = "annotation_name"

	HCOMetricOverwrittenModifications = "overwrittenModifications"
	HCOMetricUnsafeModifications      = "unsafeModifications"
	HCOMetricHyperConvergedExists     = "HyperConvergedCRExists"

	HyperConvergedExists    = float64(1)
	HyperConvergedNotExists = float64(0)
)

type metricDesc struct {
	fqName          string
	help            string
	mType           string
	constLabelPairs []string
	initFunc        func(metricDesc) prometheus.Collector
}

func (md metricDesc) init() prometheus.Collector {
	return md.initFunc(md)
}

// HcoMetrics wrapper for all hco metrics
var HcoMetrics = func() hcoMetrics {
	metricDescList := map[string]metricDesc{
		HCOMetricOverwrittenModifications: {
			fqName:          "kubevirt_hco_out_of_band_modifications_count",
			help:            "Count of out-of-band modifications overwritten by HCO",
			mType:           "Counter",
			constLabelPairs: []string{counterLabelCompName},
			initFunc: func(md metricDesc) prometheus.Collector {
				return prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: md.fqName,
						Help: md.help,
					},
					md.constLabelPairs,
				)
			},
		},
		HCOMetricUnsafeModifications: {
			fqName:          "kubevirt_hco_unsafe_modification_count",
			help:            "Count of unsafe modifications in the HyperConverged annotations",
			mType:           "Gauge",
			constLabelPairs: []string{counterLabelAnnName},
			initFunc: func(md metricDesc) prometheus.Collector {
				return prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: md.fqName,
						Help: md.help,
					},
					md.constLabelPairs,
				)
			},
		},
		HCOMetricHyperConvergedExists: {
			fqName:          "kubevirt_hco_hyperconverged_cr_exists",
			help:            "Indicates whether the HyperConverged custom resource exists (1) or not (0)",
			mType:           "Gauge",
			constLabelPairs: []string{counterLabelAnnName},
			initFunc: func(md metricDesc) prometheus.Collector {
				return prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: md.fqName,
						Help: md.help,
					},
				)
			},
		},
	}

	metricList := make(map[string]prometheus.Collector)
	for k, md := range metricDescList {
		metricList[k] = md.init()
	}

	return hcoMetrics{
		metricDescList: metricDescList,
		metricList:     metricList,
	}
}()

// hcoMetrics holds all HCO metrics
type hcoMetrics struct {
	// overwrittenModifications counts out-of-band modifications overwritten by HCO
	metricDescList map[string]metricDesc
	metricList     map[string]prometheus.Collector
}

func init() {
	HcoMetrics.init()
}

func (hm hcoMetrics) init() {
	collectors := make([]prometheus.Collector, len(hm.metricList))
	i := 0
	for _, v := range hm.metricList {
		collectors[i] = v
		i++
	}
	metrics.Registry.MustRegister(collectors...)
}

func (hm *hcoMetrics) GetMetricValue(metricName string, label prometheus.Labels) (float64, error) {
	var res = &dto.Metric{}
	metric, found := hm.metricList[metricName]
	if !found {
		return 0, unknownMetricNameError(metricName)
	}
	switch m := metric.(type) {
	case *prometheus.CounterVec:
		err := m.With(label).Write(res)
		if err != nil {
			return 0, err
		}
		return res.Counter.GetValue(), nil

	case *prometheus.GaugeVec:
		err := m.With(label).Write(res)
		if err != nil {
			return 0, err
		}
		return res.Gauge.GetValue(), nil

	case prometheus.Gauge:
		err := m.Write(res)
		if err != nil {
			return 0, err
		}
		return res.Gauge.GetValue(), nil

	default:
		return 0, unknownMetricTypeError(metricName)
	}
}

func (hm *hcoMetrics) IncMetric(metricName string, label prometheus.Labels) error {
	metric, found := hm.metricList[metricName]
	if !found {
		return unknownMetricNameError(metricName)
	}
	switch m := metric.(type) {
	case *prometheus.CounterVec:
		m.With(label).Inc()

	case *prometheus.GaugeVec:
		m.With(label).Inc()

	case prometheus.Gauge:
		m.Inc()

	default:
		return unknownMetricTypeError(metricName)
	}
	return nil
}

func (hm *hcoMetrics) SetMetric(metricName string, label prometheus.Labels, value float64) error {
	metric, found := hm.metricList[metricName]
	if !found {
		return unknownMetricNameError(metricName)
	}

	switch m := metric.(type) {
	case *prometheus.GaugeVec:
		m.With(label).Set(value)

	case prometheus.Gauge:
		m.Set(value)

	default:
		return unknownMetricTypeError(metricName)
	}

	return nil
}

// IncOverwrittenModifications increments counter by 1
func (hm *hcoMetrics) IncOverwrittenModifications(kind, name string) error {
	return hm.IncMetric(HCOMetricOverwrittenModifications, getLabelsForObj(kind, name))
}

// GetOverwrittenModificationsCount returns current value of counter. If error is not nil then value is undefined
func (hm *hcoMetrics) GetOverwrittenModificationsCount(kind, name string) (float64, error) {
	return hm.GetMetricValue(HCOMetricOverwrittenModifications, getLabelsForObj(kind, name))
}

// SetUnsafeModificationCount sets the counter to the required number
func (hm *hcoMetrics) SetUnsafeModificationCount(count int, unsafeAnnotation string) error {
	return hm.SetMetric(HCOMetricUnsafeModifications, getLabelsForUnsafeAnnotation(unsafeAnnotation), float64(count))
}

// GetUnsafeModificationsCount returns current value of counter. If error is not nil then value is undefined
func (hm *hcoMetrics) GetUnsafeModificationsCount(unsafeAnnotation string) (float64, error) {
	return hm.GetMetricValue(HCOMetricUnsafeModifications, getLabelsForUnsafeAnnotation(unsafeAnnotation))
}

// SetHCOMetricHyperConvergedExists sets the counter to 1 (true)
func (hm *hcoMetrics) SetHCOMetricHyperConvergedExists() error {
	return hm.SetMetric(HCOMetricHyperConvergedExists, nil, HyperConvergedExists)
}

// SetHCOMetricHyperConvergedNotExists sets the counter to 0 (false)
func (hm *hcoMetrics) SetHCOMetricHyperConvergedNotExists() error {
	return hm.SetMetric(HCOMetricHyperConvergedExists, nil, HyperConvergedNotExists)
}

// IsHCOMetricHyperConvergedExists returns true if the HyperConverged custom resource exists; else, return false
func (hm *hcoMetrics) IsHCOMetricHyperConvergedExists() (bool, error) {

	val, err := hm.GetMetricValue(HCOMetricHyperConvergedExists, nil)
	if err != nil {
		return false, err
	}

	return val == HyperConvergedExists, nil
}

func getLabelsForObj(kind string, name string) prometheus.Labels {
	return prometheus.Labels{counterLabelCompName: strings.ToLower(kind + "/" + name)}
}

func getLabelsForUnsafeAnnotation(unsafeAnnotation string) prometheus.Labels {
	return prometheus.Labels{counterLabelAnnName: strings.ToLower(unsafeAnnotation)}
}

type MetricDescription struct {
	FqName string
	Help   string
	Type   string
}

func (hm hcoMetrics) GetMetricDesc() []MetricDescription {
	var res []MetricDescription
	for _, md := range hm.metricDescList {
		res = append(res, MetricDescription{FqName: md.fqName, Help: md.help, Type: md.mType})
	}
	return res
}

func unknownMetricNameError(metricName string) error {
	return fmt.Errorf("unknown metric name %s", metricName)
}

func unknownMetricTypeError(metricName string) error {
	return fmt.Errorf("%s is with unknown metric type", metricName)
}
