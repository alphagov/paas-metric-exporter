package metrics_test

import (
	"github.com/alphagov/paas-cf-apps-statsd/events"
	. "github.com/alphagov/paas-cf-apps-statsd/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricTemplate", func() {
	tmpl := "{{.Organisation}}.{{.Space}}.{{.App}}.{{.Metric}}"
	appEvent := events.AppEvent{}

	data := Vars{
		App:          "fakeApp",
		CellId:       "21s7g287s-1s2s12w-12s12-s12w2131",
		GUID:         "j98xh12w-s1r4rf4-s12s23ds-s12sr3",
		Instance:     "0",
		Job:          "cell",
		Metric:       "cpu",
		Organisation: "fakeOrg",
		Space:        "fakeSpace",
	}

	It("should generate a metric name from template", func() {
		metric, err := data.Compose(tmpl)
		Expect(err).NotTo(HaveOccurred())
		Expect(metric).To(Equal("fakeOrg.fakeSpace.fakeApp.cpu"))
	})

	It("should fail to generate a metric name from method", func() {
		_, err := data.Compose("{{Organisation}}")
		Expect(err).To(HaveOccurred())
	})

	It("should fail to generate a metric name from template", func() {
		_, err := data.Compose("{{.NotExistingParameter}}")
		Expect(err).To(HaveOccurred())
	})

	It("should parse the data from the stream", func() {
		mv := Vars{}
		appEvent.App.Guid = "dhbd287bd8-2y3g8j-09197sg-81gs8s"
		mv.Parse(&appEvent)

		metric, err := mv.Compose("{{.GUID}}")
		Expect(err).NotTo(HaveOccurred())
		Expect(metric).To(Equal(appEvent.App.Guid))
	})
})
