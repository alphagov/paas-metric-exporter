package processors_test

import (
	"time"

	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
	. "github.com/alphagov/paas-metric-exporter/processors"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HttpStartStopProcessor", func() {
	var (
		processor                      *HttpStartStopProcessor
		envelopeHttpStartStopEventType sonde_events.Envelope_EventType
		httpStatusTestCases            = map[string]int32{
			"1xx":   100,
			"2xx":   200,
			"3xx":   301,
			"4xx":   401,
			"5xx":   500,
			"other": 0,
		}
	)

	BeforeEach(func() {
		processor = &HttpStartStopProcessor{}

		envelopeHttpStartStopEventType = sonde_events.Envelope_HttpStartStop
	})

	Describe("#Process", func() {
		for statusRange, statusCode := range httpStatusTestCases {
			Context(statusRange, func() {
				var (
					applicationID      string
					httpStartStopEvent *events.AppEvent
					stopTimestamp      int64
				)

				BeforeEach(func() {
					applicationID = "4630f6ba-8ddc-41f1-afea-1905332d6660"
					startTimestamp := int64(0)
					stopTimestamp = int64(11 * time.Millisecond)
					clientPeerType := sonde_events.PeerType_Client
					getMethod := sonde_events.Method_GET
					instanceIndex := int32(0)

					httpStartStopEvent = &events.AppEvent{
						App: cfclient.App{
							Guid: applicationID,
						},
						Envelope: &sonde_events.Envelope{
							EventType: &envelopeHttpStartStopEventType,
							HttpStartStop: &sonde_events.HttpStartStop{
								StartTimestamp: &startTimestamp,
								StopTimestamp:  &stopTimestamp,
								PeerType:       &clientPeerType,
								Method:         &getMethod,
								Uri:            str("/"),
								StatusCode:     &statusCode,
								InstanceIndex:  &instanceIndex,
							},
						},
					}
				})

				It("returns requests counter metric", func() {
					processedMetrics, err := processor.Process(httpStartStopEvent)
					Expect(err).ToNot(HaveOccurred())
					Expect(processedMetrics).To(ContainElement(metrics.CounterMetric{
						Metric:   "requests",
						Instance: "0",
						GUID:     applicationID,
						Metadata: map[string]string{"statusRange": statusRange},
						Value:    1,
					}))
				})

				It("returns response_time timer metric", func() {
					processedMetrics, err := processor.Process(httpStartStopEvent)
					Expect(err).ToNot(HaveOccurred())
					Expect(processedMetrics).To(ContainElement(metrics.PrecisionTimingMetric{
						Metric:   "responseTime",
						Instance: "0",
						GUID:     applicationID,
						Metadata: map[string]string{"statusRange": statusRange},
						Value:    11 * time.Millisecond,
						Stop:     stopTimestamp,
					}))
				})

				It("ignores metrics with a Server PeerType", func() {
					serverPeerType := sonde_events.PeerType_Server
					httpStartStopEvent.Envelope.HttpStartStop.PeerType = &serverPeerType

					processedMetrics, err := processor.Process(httpStartStopEvent)
					Expect(err).ToNot(HaveOccurred())
					Expect(processedMetrics).To(BeEmpty())
				})
			})
		}
	})
})
