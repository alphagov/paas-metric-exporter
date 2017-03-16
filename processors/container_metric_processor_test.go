package processors_test

import (
	"github.com/alphagov/paas-cf-apps-statsd/metrics"
	. "github.com/alphagov/paas-cf-apps-statsd/processors"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/sonde-go/events"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContainerMetricProcessor", func() {
	var (
		processor            *ContainerMetricProcessor
		event                *events.Envelope
		stream               *metrics.Stream
		containerMetricEvent *events.ContainerMetric
	)

	BeforeEach(func() {
		processor = NewContainerMetricProcessor()

		applicationId := "60a13b0f-fce7-4c02-b92a-d43d583877ed"
		instanceIndex := int32(0)
		cpuPercentage := float64(70.75)
		memoryBytes := uint64(1024)
		diskBytes := uint64(2048)

		containerMetricEvent = &events.ContainerMetric{
			ApplicationId: &applicationId,
			InstanceIndex: &instanceIndex,
			CpuPercentage: &cpuPercentage,
			MemoryBytes:   &memoryBytes,
			DiskBytes:     &diskBytes,
		}

		event = &events.Envelope{
			ContainerMetric: containerMetricEvent,
		}

		stream = &metrics.Stream{
			App: cfclient.App{
				Guid: applicationId,
			},
			Msg:  event,
			Tmpl: "apps.{{.GUID}}.{{.Metric}}.{{.Instance}}",
		}
	})

	Describe("#Process", func() {
		It("returns a Metric for each of the ProcessContainerMetric* methods", func() {
			processedMetrics, err := processor.Process(stream)

			Expect(err).To(BeNil())
			Expect(processedMetrics).To(HaveLen(3))
		})

		It("should fail", func() {
			_, err := processor.Process(&metrics.Stream{Tmpl: "{{Error}}"})

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("#ProcessContainerMetricCPU", func() {
		It("formats the Stat string to include the ContainerMetric's app ID and instance index", func() {
			metric, err := processor.ProcessContainerMetric("cpu", stream)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Stat).To(Equal("apps.60a13b0f-fce7-4c02-b92a-d43d583877ed.cpu.0"))
		})

		It("sets the Metric Value to the value of the ContainerMetric cpuPercentage", func() {
			metric, err := processor.ProcessContainerMetric("cpu", stream)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Value).To(Equal(int64(70)))
		})
	})

	Describe("#ProcessContainerMetricMemory", func() {
		It("formats the Stat string to include the ContainerMetric's app ID and instance index", func() {
			metric, err := processor.ProcessContainerMetric("mem", stream)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Stat).To(Equal("apps.60a13b0f-fce7-4c02-b92a-d43d583877ed.memoryBytes.0"))
		})

		It("sets the Metric Value to the value of the ContainerMetric memoryBytes", func() {
			metric, err := processor.ProcessContainerMetric("mem", stream)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Value).To(Equal(int64(1024)))
		})
	})

	Describe("#ProcessContainerMetricDisk", func() {
		It("formats the Stat string to include the ContainerMetric's app ID and instance index", func() {
			metric, err := processor.ProcessContainerMetric("dsk", stream)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Stat).To(Equal("apps.60a13b0f-fce7-4c02-b92a-d43d583877ed.diskBytes.0"))
		})

		It("sets the Metric Value to the value of the ContainerMetric diskBytes", func() {
			metric, err := processor.ProcessContainerMetric("dsk", stream)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Value).To(Equal(int64(2048)))
		})
	})

	Describe("#ProcessContainerMetricUnknown", func() {
		It("should fail", func() {
			_, err := processor.ProcessContainerMetric("unknown", stream)

			Expect(err).To(HaveOccurred())
		})
	})

})
