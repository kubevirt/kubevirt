package operatormetrics

import (
	"errors"
	"time"
)

type CollectorResult struct {
	Metric      Metric
	Labels      []string
	ConstLabels map[string]string
	Value       float64
	Timestamp   time.Time
}

func (cr CollectorResult) GetLabelValue(key string) (string, error) {
	for i, label := range cr.Metric.GetOpts().labels {
		if label == key {
			return cr.Labels[i], nil
		}
	}

	return "", errors.New("label not found")
}
