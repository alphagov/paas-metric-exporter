package presenters_test

import (
	. "github.com/alphagov/paas-metric-exporter/metrics"
	. "github.com/alphagov/paas-metric-exporter/presenters"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PathPresenter", func() {
	Describe("#Present", func() {
		It("should pass all data through to the template", func() {
			presenter, _ := NewPathPresenter("{{.Organisation}}.{{.Space}}.{{.App}}.{{.GUID}}.{{.Instance}}.{{.Metric}}")
			data := CounterMetric{
				App:          "app",
				GUID:         "guid",
				Instance:     "instance",
				Metric:       "widgets",
				Space:        "space",
				Organisation: "org",
			}
			output, err := presenter.Present(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal("org.space.app.guid.instance.widgets"))
		})

		It("should present simple metric name", func() {
			presenter, _ := NewPathPresenter("{{.Metric}}")
			data := CounterMetric{Metric: "foo"}
			output, err := presenter.Present(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal("foo"))
		})

		It("should join Metadata.statusRange on to metric name if present", func() {
			presenter, _ := NewPathPresenter("{{.Metric}}")
			data := CounterMetric{Metric: "foo", Metadata: map[string]string{"statusRange": "2xx"}}
			output, err := presenter.Present(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal("foo.2xx"))
		})

		It("should fail to construct the presenter due to lack of dot", func() {
			_, err := NewPathPresenter("{{Foo}}")
			Expect(err).Should(MatchError("template: metric:1: function \"Foo\" not defined"))
		})

		It("should fail to present the data due to unknown property in template", func() {
			presenter, _ := NewPathPresenter("{{.Missing}}")
			data := GaugeMetric{Metric: "foo", Organisation: "bar"}
			_, err := presenter.Present(data)

			Expect(err).Should(MatchError("template: metric:1:2: executing \"metric\" at <.Missing>: can't evaluate field Missing in type presenters.MetricView"))
		})
	})
})
