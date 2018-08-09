package senders_test

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	. "github.com/alphagov/paas-metric-exporter/senders"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsExpirer", func() {
	var (
		expirer        *MetricsExpirer
		deletedMetrics map[string]prometheus.Labels
	)

	BeforeEach(func() {
		deletedMetrics = map[string]prometheus.Labels{}
		expirer = NewMetricsExpirer(
			func(name string, labels prometheus.Labels) {
				deletedMetrics[name] = labels
			},
			100*time.Millisecond,
			50*time.Millisecond,
		)
	})

	AfterEach(func() {
		expirer.Stop()
	})

	It("does not expire metrics recently refreshed", func() {
		Consistently(
			func() map[string]prometheus.Labels {
				expirer.SeenMetric("a_metric", prometheus.Labels{"label1": "val1"})
				return deletedMetrics
			},
			500*time.Millisecond,
			50*time.Millisecond,
		).Should(
			HaveLen(0),
		)
	})

	It("expires old metrics", func() {
		expirer.SeenMetric("a_metric", prometheus.Labels{"label1": "val1"})
		expirer.SeenMetric("other_metric", prometheus.Labels{"label1": "val1"})
		expirer.SeenMetric("other_metric", prometheus.Labels{"label2": "val2"})

		Eventually(
			func() map[string]prometheus.Labels {
				expirer.SeenMetric("other_metric", prometheus.Labels{"label2": "val2"})
				expirer.SeenMetric("and_other_metric", prometheus.Labels{})
				return deletedMetrics
			},
			500*time.Millisecond,
			50*time.Millisecond,
		).Should(
			And(
				HaveLen(2),
				HaveKeyWithValue("a_metric", prometheus.Labels{"label1": "val1"}),
				HaveKeyWithValue("other_metric", prometheus.Labels{"label1": "val1"}),
			),
		)
	})
})
