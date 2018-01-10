package processors_test

import (
	"github.com/alphagov/paas-metric-exporter/events"
	. "github.com/alphagov/paas-metric-exporter/processors"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContainerMetricProcessor", func() {
	var (
		processor            *ContainerMetricProcessor
		event                *sonde_events.Envelope
		appEvent             *events.AppEvent
		tmpl                 string
		containerMetricEvent *sonde_events.ContainerMetric
	)

	BeforeEach(func() {
		tmpl = "apps.{{.GUID}}.{{.Metric}}.{{.Instance}}"
		processor = NewContainerMetricProcessor(tmpl)

		applicationId := "60a13b0f-fce7-4c02-b92a-d43d583877ed"
		instanceIndex := int32(0)
		cpuPercentage := float64(70.75)
		memoryBytes := uint64(1024)
		diskBytes := uint64(2048)

		containerMetricEvent = &sonde_events.ContainerMetric{
			ApplicationId: &applicationId,
			InstanceIndex: &instanceIndex,
			CpuPercentage: &cpuPercentage,
			MemoryBytes:   &memoryBytes,
			DiskBytes:     &diskBytes,
		}

		event = &sonde_events.Envelope{
			ContainerMetric: containerMetricEvent,
		}

		appEvent = &events.AppEvent{
			App: cfclient.App{
				Guid: applicationId,
			},
			Envelope: event,
		}
	})

	Describe("#Process", func() {
		It("returns a Metric for each of the ProcessContainerMetric* methods", func() {
			processedMetrics, err := processor.Process(appEvent)

			Expect(err).To(BeNil())
			Expect(processedMetrics).To(HaveLen(3))
		})
	})

	Describe("#ProcessContainerMetricCPU", func() {
		It("sets the Metric Value to the value of the ContainerMetric cpuPercentage", func() {
			metric, err := processor.ProcessContainerMetric("cpu", appEvent)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Value).To(Equal(int64(70)))
		})
	})

	Describe("#ProcessContainerMetricMemory", func() {
		It("sets the Metric Value to the value of the ContainerMetric memoryBytes", func() {
			metric, err := processor.ProcessContainerMetric("mem", appEvent)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Value).To(Equal(int64(1024)))
		})
	})

	Describe("#ProcessContainerMetricDisk", func() {
		It("sets the Metric Value to the value of the ContainerMetric diskBytes", func() {
			metric, err := processor.ProcessContainerMetric("dsk", appEvent)

			Expect(err).NotTo(HaveOccurred())
			Expect(metric.Value).To(Equal(int64(2048)))
		})
	})

	Describe("#ProcessContainerMetricUnknown", func() {
		It("should fail", func() {
			_, err := processor.ProcessContainerMetric("unknown", appEvent)

			Expect(err).To(HaveOccurred())
		})
	})

})
