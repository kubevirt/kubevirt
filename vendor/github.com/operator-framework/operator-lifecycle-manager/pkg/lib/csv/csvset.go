package csv

import (
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorlister"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

// NewSetGenerator returns a new instance of SetGenerator.
func NewSetGenerator(logger *logrus.Logger, lister operatorlister.OperatorLister) SetGenerator {
	return &csvSet{
		logger: logger,
		lister: lister,
	}
}

// SetGenerator is an interface that returns a map of ClusterServiceVersion
// objects that match a certain set of criteria.
//
// SetGenerator gathers all CSV(s) in the given namespace into a map keyed by
// CSV name; if metav1.NamespaceAll gets the set across all namespaces
type SetGenerator interface {
	WithNamespace(namespace string, phase v1alpha1.ClusterServiceVersionPhase) map[string]*v1alpha1.ClusterServiceVersion
	WithNamespaceAndLabels(namespace string, phase v1alpha1.ClusterServiceVersionPhase, selector labels.Selector) map[string]*v1alpha1.ClusterServiceVersion
}

type csvSet struct {
	lister operatorlister.OperatorLister
	logger *logrus.Logger
}

// WithNamespace returns all ClusterServiceVersion resource(s) that matches the
// specified phase from a given namespace.
func (s *csvSet) WithNamespace(namespace string, phase v1alpha1.ClusterServiceVersionPhase) map[string]*v1alpha1.ClusterServiceVersion {
	return s.with(namespace, phase, labels.Everything())
}

// WithNamespaceAndLabels returns all ClusterServiceVersion resource(s) that
// matches the specified phase and label selector from a given namespace.
func (s *csvSet) WithNamespaceAndLabels(namespace string, phase v1alpha1.ClusterServiceVersionPhase, selector labels.Selector) map[string]*v1alpha1.ClusterServiceVersion {
	return s.with(namespace, phase, selector)
}

func (s *csvSet) with(namespace string, phase v1alpha1.ClusterServiceVersionPhase, selector labels.Selector) map[string]*v1alpha1.ClusterServiceVersion {
	csvsInNamespace, err := s.lister.OperatorsV1alpha1().ClusterServiceVersionLister().ClusterServiceVersions(namespace).List(selector)

	if err != nil {
		s.logger.Warnf("could not list CSVs while constructing CSV set")
		return nil
	}

	csvs := make(map[string]*v1alpha1.ClusterServiceVersion, len(csvsInNamespace))
	for _, csv := range csvsInNamespace {
		if phase != v1alpha1.CSVPhaseAny && csv.Status.Phase != phase {
			continue
		}
		csvs[csv.Name] = csv.DeepCopy()
	}

	return csvs
}
