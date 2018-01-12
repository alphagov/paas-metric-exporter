package processors_test

import (
	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
	. "github.com/alphagov/paas-metric-exporter/processors"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContainerMetricProcessor", func() {
	var (
		processor            *ContainerMetricProcessor
		event                *sonde_events.Envelope
		appEvent             *events.AppEvent
		containerMetricEvent *sonde_events.ContainerMetric
	)

	BeforeEach(func() {
		processor = &ContainerMetricProcessor{}

		applicationId := "60a13b0f-fce7-4c02-b92a-d43d583877ed"

		containerMetricEvent = &sonde_events.ContainerMetric{
			ApplicationId:    proto.String(applicationId),
			InstanceIndex:    proto.Int32(1),
			CpuPercentage:    proto.Float64(70.75),
			MemoryBytes:      proto.Uint64(16 * 1024 * 1024),
			MemoryBytesQuota: proto.Uint64(32 * 1024 * 1024),
			DiskBytes:        proto.Uint64(25 * 1024 * 1024),
			DiskBytesQuota:   proto.Uint64(100 * 1024 * 1024),
		}

		event = &sonde_events.Envelope{
			ContainerMetric: containerMetricEvent,
			Index:           proto.String("cell-id"),
			Job:             proto.String("job-name"),
		}

		appEvent = &events.AppEvent{
			App: cfclient.App{
				Name: "app-name",
				Guid: applicationId,
				SpaceData: cfclient.SpaceResource{
					Entity: cfclient.Space{
						Name: "space-name",
						OrgData: cfclient.OrgResource{
							Entity: cfclient.Org{
								Name: "org-name",
							},
						},
					},
				},
			},
			Envelope: event,
		}
	})

	Describe("Process", func() {
		It("returns a Metric for each of the ProcessContainerMetric* methods", func() {
			processedMetrics, err := processor.Process(appEvent)

			Expect(err).To(BeNil())
			Expect(processedMetrics).To(HaveLen(5))

			expectedCPUMetric := metrics.GaugeMetric{
				Instance:     "1",
				App:          "app-name",
				GUID:         "60a13b0f-fce7-4c02-b92a-d43d583877ed",
				CellId:       "cell-id",
				Job:          "job-name",
				Organisation: "org-name",
				Space:        "space-name",
				Metric:       "cpu",
				Value:        70,
			}
			Expect(processedMetrics).To(ContainElement(expectedCPUMetric))

			expectedMemoryMetric := metrics.GaugeMetric{
				Instance:     "1",
				App:          "app-name",
				GUID:         "60a13b0f-fce7-4c02-b92a-d43d583877ed",
				CellId:       "cell-id",
				Job:          "job-name",
				Organisation: "org-name",
				Space:        "space-name",
				Metric:       "memoryBytes",
				Value:        16 * 1024 * 1024,
			}
			Expect(processedMetrics).To(ContainElement(expectedMemoryMetric))

			expectedMemoryUtilisationMetric := metrics.GaugeMetric{
				Instance:     "1",
				App:          "app-name",
				GUID:         "60a13b0f-fce7-4c02-b92a-d43d583877ed",
				CellId:       "cell-id",
				Job:          "job-name",
				Organisation: "org-name",
				Space:        "space-name",
				Metric:       "memoryUtilization",
				Value:        50,
			}
			Expect(processedMetrics).To(ContainElement(expectedMemoryUtilisationMetric))

			expectedDiskMetric := metrics.GaugeMetric{
				Instance:     "1",
				App:          "app-name",
				GUID:         "60a13b0f-fce7-4c02-b92a-d43d583877ed",
				CellId:       "cell-id",
				Job:          "job-name",
				Organisation: "org-name",
				Space:        "space-name",
				Metric:       "diskBytes",
				Value:        25 * 1024 * 1024,
			}
			Expect(processedMetrics).To(ContainElement(expectedDiskMetric))

			expectedDiskUtilisationMetric := metrics.GaugeMetric{
				Instance:     "1",
				App:          "app-name",
				GUID:         "60a13b0f-fce7-4c02-b92a-d43d583877ed",
				CellId:       "cell-id",
				Job:          "job-name",
				Organisation: "org-name",
				Space:        "space-name",
				Metric:       "diskUtilization",
				Value:        25,
			}
			Expect(processedMetrics).To(ContainElement(expectedDiskUtilisationMetric))
		})

		It("returns an error if memory quota is zero", func() {
			containerMetricEvent.MemoryBytesQuota = proto.Uint64(0)
			_, err := processor.Process(appEvent)
			Expect(err).To(HaveOccurred())
		})

		It("returns an error if disk quota is zero", func() {
			containerMetricEvent.DiskBytesQuota = proto.Uint64(0)
			_, err := processor.Process(appEvent)
			Expect(err).To(HaveOccurred())
		})
	})

})
