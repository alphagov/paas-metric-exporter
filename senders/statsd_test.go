package senders_test

import (
	"time"

	. "github.com/alphagov/paas-metric-exporter/metrics"
	. "github.com/alphagov/paas-metric-exporter/senders"

	. "github.com/quipo/statsd/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StatsdSender", func() {
	client := MockStatsdClient{}

	var capturedMetrics map[string]interface{}
	BeforeEach(func() {
		capturedMetrics = make(map[string]interface{})

		client.IncrFn = func(name string, value int64) error {
			capturedMetrics[name] = value
			return nil
		}

		client.GaugeFn = func(name string, value int64) error {
			capturedMetrics[name] = value
			return nil
		}

		client.PrecisionTimingFn = func(name string, value time.Duration) error {
			capturedMetrics[name] = value
			return nil
		}
	})

	Describe("#Incr", func() {
		It("sends a counter metric to statsd", func() {
			template := "{{.Metric}}"
			sender, _ := NewStatsdSender(&client, template)

			sender.Incr(CounterMetric{
				Metric: "my_counter",
				Value:  3,
			})

			Expect(len(capturedMetrics)).To(Equal(1))
			Expect(capturedMetrics["my_counter"]).To(Equal(int64(3)))
		})

		It("builds statsd paths from template strings", func() {
			template := "{{.Space}}.{{.App}}.{{.Instance}}.{{.Metric}}"
			sender, _ := NewStatsdSender(&client, template)

			sender.Incr(CounterMetric{
				Space:    "my_space",
				App:      "my_app",
				Instance: "my_instance",
				Metric:   "requests",

				Metadata: map[string]string{"statusRange": "2xx"},

				Value: 3,
			})

			Expect(len(capturedMetrics)).To(Equal(1))

			value := capturedMetrics["my_space.my_app.my_instance.requests.2xx"]
			Expect(value).To(Equal(int64(3)))
		})

		It("does not include metadata in the string if absent from metric", func() {
			template := "{{.Space}}.{{.App}}.{{.Instance}}.{{.Metric}}"
			sender, _ := NewStatsdSender(&client, template)

			sender.Incr(CounterMetric{
				Space:    "my_space",
				App:      "my_app",
				Instance: "my_instance",
				Metric:   "requests",

				Value: 3,
			})

			Expect(len(capturedMetrics)).To(Equal(1))

			value := capturedMetrics["my_space.my_app.my_instance.requests"]
			Expect(value).To(Equal(int64(3)))
		})
	})

	Describe("#Gauge", func() {
		It("sends a gauge metric to statsd", func() {
			template := "{{.Metric}}"
			sender, _ := NewStatsdSender(&client, template)

			sender.Gauge(GaugeMetric{
				Metric: "my_gauge",
				Value:  3,
			})

			Expect(len(capturedMetrics)).To(Equal(1))
			Expect(capturedMetrics["my_gauge"]).To(Equal(int64(3)))
		})
	})

	Describe("#PrecisionTiming", func() {
		It("sends a precision timing metric to statsd", func() {
			template := "{{.Metric}}"
			sender, _ := NewStatsdSender(&client, template)

			sender.PrecisionTiming(PrecisionTimingMetric{
				Metric: "my_precise_time",
				Value:  time.Duration(3),
			})

			Expect(len(capturedMetrics)).To(Equal(1))
			Expect(capturedMetrics["my_precise_time"]).To(Equal(time.Duration(3)))
		})
	})
})
