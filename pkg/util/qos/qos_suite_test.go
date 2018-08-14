package qos_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestQos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Qos Suite")
}
