package senders_test

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_model/go"

	. "github.com/alphagov/paas-metric-exporter/metrics"
	. "github.com/alphagov/paas-metric-exporter/senders"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PrometheusSender", func() {
	var (
		registerer prometheus.Registerer
		gatherer   prometheus.Gatherer
		sender     Sender
	)
	BeforeEach(func() {
		registry := prometheus.NewRegistry()
		gatherer = registry
		registerer = registry
		sender = NewPrometheusSender(
			registerer,
			100*time.Millisecond,
		)
	})

	Describe("#Incr", func() {
		It("sends a counter metric to prometheus", func() {
			families := captureMetrics(gatherer, func() {
				sender.Incr(CounterMetric{
					Metric: "counter_incremented_once",
					Value:  1,
					App:    "some_value",
				})
			})

			Expect(len(families)).To(Equal(1))
			family := families[0]
			metrics := family.GetMetric()

			Expect(len(metrics)).To(Equal(1))
			metric := metrics[0].Counter
			labels := metrics[0].GetLabel()

			Expect(family.GetName()).To(Equal("counter_incremented_once"))
			Expect(metric.GetValue()).To(Equal(float64(1)))

			Expect(labels[0].GetName()).To(Equal("app"))
			Expect(labels[0].GetValue()).To(Equal("some_value"))
		})

		It("forgets metrics from an app instance after a while", func() {
			families := captureMetrics(gatherer, func() {
				sender.Incr(CounterMetric{
					Metric:   "counter_incremented_once",
					Value:    1,
					App:      "some_value",
					GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
					Instance: "0",
				})
				sender.Incr(CounterMetric{
					Metric:   "counter_incremented_once",
					Value:    1,
					App:      "some_value",
					GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
					Instance: "1",
				})
			})

			Expect(len(families)).To(Equal(1))
			family := families[0]
			metrics := family.GetMetric()

			Expect(len(metrics)).To(Equal(2))

			Eventually(
				func() interface{} {
					sender.Incr(CounterMetric{
						Metric:   "counter_incremented_once",
						Value:    1,
						App:      "some_value",
						GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
						Instance: "0",
					})
					families, _ := gatherer.Gather()
					Expect(len(families)).To(Equal(1))
					family := families[0]
					metrics := family.GetMetric()
					return metrics
				},
				500*time.Millisecond,
				200*time.Millisecond,
			).Should(
				HaveLen(1),
			)
		})

		It("presents metric names and label names as snake case", func() {
			families := captureMetrics(gatherer, func() {
				sender.Incr(CounterMetric{
					Metric: "fooBarBaz",
					Value:  1,
					App:    "shouldNotBeChanged",
					CellId: "cell_id_value",
					GUID:   "guid_value",
				})
			})

			Expect(len(families)).To(Equal(1))
			family := families[0]

			metrics := family.GetMetric()
			Expect(len(metrics)).To(Equal(1))
			labels := metrics[0].GetLabel()

			Expect(family.GetName()).To(Equal("foo_bar_baz"))

			Expect(labels[0].GetName()).To(Equal("app"))
			Expect(labels[1].GetName()).To(Equal("cell_id"))
			Expect(labels[2].GetName()).To(Equal("guid"))

			Expect(labels[0].GetValue()).To(Equal("shouldNotBeChanged"))
		})

		It("does not error when called multiple times", func() {
			counterMetric := CounterMetric{
				Metric: "counter_incremented_multiple_times",
				Value:  1,
				App:    "some_value",
			}

			families := captureMetrics(gatherer, func() {
				sender.Incr(counterMetric)
				sender.Incr(counterMetric)
				sender.Incr(counterMetric)
			})

			Expect(len(families)).To(Equal(1))
			metrics := families[0].GetMetric()

			Expect(len(metrics)).To(Equal(1))
			metric := metrics[0].Counter

			Expect(metric.GetValue()).To(Equal(float64(3)))
		})

		It("includes Metadata as additional labels", func() {
			families := captureMetrics(gatherer, func() {
				sender.Incr(CounterMetric{
					Metric:   "response",
					Metadata: map[string]string{"statusRange": "2xx"},
					Value:    1,
				})
			})

			Expect(len(families)).To(Equal(1))
			metrics := families[0].GetMetric()

			Expect(len(metrics)).To(Equal(1))
			labels := metrics[0].GetLabel()
			metadata := labels[len(labels)-1]

			Expect(metadata.GetName()).To(Equal("status_range"))
			Expect(metadata.GetValue()).To(Equal("2xx"))
		})

	})

	Describe("#Gauge", func() {
		It("sends a floating point gauge metric to prometheus", func() {
			families := captureMetrics(gatherer, func() {
				sender.Gauge(GaugeMetric{
					Metric: "my_gauge",
					Value:  3,
				})
			})

			Expect(len(families)).To(Equal(1))
			metrics := families[0].GetMetric()

			Expect(len(metrics)).To(Equal(1))
			family := families[0]
			metric := metrics[0].Gauge

			Expect(family.GetName()).To(Equal("my_gauge"))
			Expect(metric.GetValue()).To(Equal(3.0))
		})

		It("forgets metrics from an app instance after a while", func() {
			families := captureMetrics(gatherer, func() {
				sender.Gauge(GaugeMetric{
					Metric:   "my_gauge",
					Value:    3,
					App:      "some_value",
					GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
					Instance: "0",
				})
				sender.Gauge(GaugeMetric{
					Metric:   "my_gauge",
					Value:    3,
					App:      "some_value",
					GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
					Instance: "1",
				})
			})

			Expect(len(families)).To(Equal(1))
			family := families[0]
			metrics := family.GetMetric()
			Expect(len(metrics)).To(Equal(2))

			Eventually(
				func() interface{} {
					sender.Gauge(GaugeMetric{
						Metric:   "my_gauge",
						Value:    3,
						App:      "some_value",
						GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
						Instance: "0",
					})
					families, _ := gatherer.Gather()
					Expect(len(families)).To(Equal(1))
					family := families[0]
					metrics := family.GetMetric()
					return metrics
				},
				500*time.Millisecond,
				200*time.Millisecond,
			).Should(
				HaveLen(1),
			)
		})
	})

	Describe("#PrecisionTiming", func() {
		It("sends a histogram metric into a sensible bucket in prometheus", func() {
			families := captureMetrics(gatherer, func() {
				sender.PrecisionTiming(PrecisionTimingMetric{
					Metric: "my_precise_time",
					Value:  time.Duration(3142) * time.Millisecond,
				})
			})

			Expect(len(families)).To(Equal(1))
			family := families[0]
			metrics := family.GetMetric()

			Expect(len(metrics)).To(Equal(1))
			metric := metrics[0].Histogram
			buckets := metric.GetBucket()

			Expect(family.GetName()).To(Equal("my_precise_time"))
			Expect(metric.GetSampleCount()).To(Equal(uint64(1)))
			Expect(metric.GetSampleSum()).To(Equal(3.142))

			last_buckets := buckets[len(buckets)-3:]

			Expect(last_buckets[0].GetUpperBound()).To(Equal(2.5))
			Expect(last_buckets[0].GetCumulativeCount()).To(Equal(uint64(0)))

			Expect(last_buckets[1].GetUpperBound()).To(Equal(5.0))
			Expect(last_buckets[1].GetCumulativeCount()).To(Equal(uint64(1)))

			Expect(last_buckets[2].GetUpperBound()).To(Equal(10.0))
			Expect(last_buckets[2].GetCumulativeCount()).To(Equal(uint64(1)))
		})

		It("forgets metrics from an app instance after a while", func() {
			families := captureMetrics(gatherer, func() {
				sender.PrecisionTiming(PrecisionTimingMetric{
					Metric:   "my_precise_time",
					Value:    time.Duration(3142) * time.Millisecond,
					App:      "some_value",
					GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
					Instance: "0",
				})
				sender.PrecisionTiming(PrecisionTimingMetric{
					Metric:   "my_precise_time",
					Value:    time.Duration(3142) * time.Millisecond,
					App:      "some_value",
					GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
					Instance: "1",
				})
			})

			Expect(len(families)).To(Equal(1))
			family := families[0]
			metrics := family.GetMetric()
			Expect(len(metrics)).To(Equal(2))

			Eventually(
				func() interface{} {
					sender.PrecisionTiming(PrecisionTimingMetric{
						Metric:   "my_precise_time",
						Value:    time.Duration(3142) * time.Millisecond,
						App:      "some_value",
						GUID:     "e44436d3-b319-4296-96c9-fc142358f965",
						Instance: "0",
					})
					families, _ := gatherer.Gather()
					Expect(len(families)).To(Equal(1))
					family := families[0]
					metrics := family.GetMetric()
					return metrics
				},
				500*time.Millisecond,
				200*time.Millisecond,
			).Should(
				HaveLen(1),
			)
		})
	})
})

type metric []*io_prometheus_client.MetricFamily

func captureMetrics(gatherer prometheus.Gatherer, callback func()) metric {
	before, _ := gatherer.Gather()
	callback()
	after, _ := gatherer.Gather()

	subtracted := subtract(after, before)
	Expect(len(subtracted)).To(BeNumerically(">", 0),
		"expected to capture some new metrics",
	)

	return subtracted
}

func subtract(aSlice metric, bSlice metric) metric {
	var subtracted metric

Outer:
	for _, a := range aSlice {
		for _, b := range bSlice {
			if a.GetName() == b.GetName() {
				continue Outer
			}
		}
		subtracted = append(subtracted, a)
	}

	return subtracted
}
