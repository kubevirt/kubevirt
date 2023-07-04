package components

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prometheus", func() {
	BeforeEach(func() {
		os.Unsetenv(runbookURLTemplateEnv)
	})

	AfterEach(func() {
		os.Unsetenv(runbookURLTemplateEnv)
	})

	It("should use the default runbook URL template when no ENV Variable is set", func() {
		promRule := NewPrometheusRuleCR("mynamespace")

		for _, group := range promRule.Spec.Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					if rule.Annotations["runbook_url"] != "" {
						Expect(rule.Annotations["runbook_url"]).To(Equal(fmt.Sprintf(defaultRunbookURLTemplate, rule.Alert)))
					}
				}
			}
		}
	})

	It("should use the desired runbook URL template when its ENV Variable is set", func() {
		desiredRunbookURLTemplate := "desired/runbookURL/template/%s"
		os.Setenv(runbookURLTemplateEnv, desiredRunbookURLTemplate)

		promRule := NewPrometheusRuleCR("mynamespace")

		for _, group := range promRule.Spec.Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					if rule.Annotations["runbook_url"] != "" {
						Expect(rule.Annotations["runbook_url"]).To(Equal(fmt.Sprintf(desiredRunbookURLTemplate, rule.Alert)))
					}
				}
			}
		}
	})
})
