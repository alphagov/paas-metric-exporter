package locking_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	metricExporterPath string
)

func TestLocking(t *testing.T) {
	BeforeSuite(func() {
		var err error

		metricExporterPath, err = gexec.Build("github.com/alphagov/paas-metric-exporter")
		Expect(err).ToNot(HaveOccurred())
	})
	RegisterFailHandler(Fail)
	RunSpecs(t, "Locking Suite")
}
