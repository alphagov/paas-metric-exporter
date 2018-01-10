package processors_test

import (
	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
	. "github.com/alphagov/paas-metric-exporter/processors"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogMessageProcessor", func() {
	var (
		processor         *LogMessageProcessor
		tmpl              string
		appCrashEvent     *events.AppEvent
		appNonCrashEvents []*events.AppEvent
	)

	BeforeEach(func() {
		tmpl = "apps.{{.GUID}}.{{.Instance}}.{{.Metric}}"
		processor = NewLogMessageProcessor(tmpl)

		envelopeLogMessageEventType := sonde_events.Envelope_LogMessage
		logMessageOutMessageType := sonde_events.LogMessage_OUT
		logMessageErrMessageType := sonde_events.LogMessage_ERR

		appCrashEvent = &events.AppEvent{
			App: cfclient.App{
				Guid: "4630f6ba-8ddc-41f1-afea-1905332d6660",
			},
			Envelope: &sonde_events.Envelope{
				Origin:    str("cloud_controller"),
				EventType: &envelopeLogMessageEventType,
				LogMessage: &sonde_events.LogMessage{
					Message:        []byte("App instance exited with guid 4630f6ba-8ddc-41f1-afea-1905332d6660 payload: {\"instance\"=>\"bc932892-f191-4fe2-60c3-7090\", \"index\"=>0, \"reason\"=>\"CRASHED\", \"exit_description\"=>\"APP/PROC/WEB: Exited with status 137\", \"crash_count\"=>1, \"crash_timestamp\"=>1512569260335558205, \"version\"=>\"d24b0422-0c88-4692-bf52-505091890e7d\"}"),
					MessageType:    &logMessageOutMessageType,
					AppId:          str("4630f6ba-8ddc-41f1-afea-1905332d6660"),
					SourceType:     str("API"),
					SourceInstance: str("1"),
				},
			},
		}
		appNonCrashEvents = []*events.AppEvent{
			&events.AppEvent{
				App: cfclient.App{
					Guid: "4630f6ba-8ddc-41f1-afea-1905332d6660",
				},
				Envelope: &sonde_events.Envelope{
					Origin:    str("gorouter"),
					EventType: &envelopeLogMessageEventType,
					LogMessage: &sonde_events.LogMessage{
						Message:        []byte("dora.dcarley.dev.cloudpipelineapps.digital - [2017-12-06T14:05:45.897+0000] \"GET / HTTP/1.1\" 200 0 13 \"-\" \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.94 Safari/537.36\" \"127.0.0.1:48966\" \"10.0.34.4:61223\" x_forwarded_for:\"213.86.153.212, 127.0.0.1\" x_forwarded_proto:\"https\" vcap_request_id:\"cd809903-c35d-4c98-6f62-1f22862cc82c\" response_time:0.018321645 app_id:\"4630f6ba-8ddc-41f1-afea-1905332d6660\" app_index:\"0\"\n"),
						MessageType:    &logMessageOutMessageType,
						AppId:          str("4630f6ba-8ddc-41f1-afea-1905332d6660"),
						SourceType:     str("RTR"),
						SourceInstance: str("1"),
					},
				},
			},
			&events.AppEvent{
				App: cfclient.App{
					Guid: "4630f6ba-8ddc-41f1-afea-1905332d6660",
				},
				Envelope: &sonde_events.Envelope{
					Origin:    str("rep"),
					EventType: &envelopeLogMessageEventType,
					LogMessage: &sonde_events.LogMessage{
						Message:        []byte("[2017-12-06 14:06:41] INFO  WEBrick 1.3.1"),
						MessageType:    &logMessageErrMessageType,
						AppId:          str("4630f6ba-8ddc-41f1-afea-1905332d6660"),
						SourceType:     str("APP/PROC/WEB"),
						SourceInstance: str("0"),
					},
				},
			},
			&events.AppEvent{
				App: cfclient.App{
					Guid: "4630f6ba-8ddc-41f1-afea-1905332d6660",
				},
				Envelope: &sonde_events.Envelope{
					Origin:    str("cloud_controller"),
					EventType: &envelopeLogMessageEventType,
					LogMessage: &sonde_events.LogMessage{
						Message:        []byte("Process has crashed with type: \"web\""),
						MessageType:    &logMessageOutMessageType,
						AppId:          str("4630f6ba-8ddc-41f1-afea-1905332d6660"),
						SourceType:     str("API"),
						SourceInstance: str("1"),
					},
				},
			},
			&events.AppEvent{
				App: cfclient.App{
					Guid: "4630f6ba-8ddc-41f1-afea-1905332d6660",
				},
				Envelope: &sonde_events.Envelope{
					Origin:    str("cloud_controller"),
					EventType: &envelopeLogMessageEventType,
					LogMessage: &sonde_events.LogMessage{
						Message:        []byte("Updated app with guid 4630f6ba-8ddc-41f1-afea-1905332d6660 ({\"state\"=>\"STOPPED\"})"),
						MessageType:    &logMessageOutMessageType,
						AppId:          str("4630f6ba-8ddc-41f1-afea-1905332d6660"),
						SourceType:     str("API"),
						SourceInstance: str("1"),
					},
				},
			},
		}
	})

	Describe("#Process", func() {
		It("returns a metric if the event is an app crash", func() {
			processedMetrics, err := processor.Process(appCrashEvent)
			Expect(err).ToNot(HaveOccurred())
			Expect(processedMetrics).To(Equal([]metrics.Metric{
				metrics.CounterMetric{
					GUID:     "4630f6ba-8ddc-41f1-afea-1905332d6660",
					Instance: "0",
					Metric:   "crash",
					Template: tmpl,
					Value:    1,
				},
			}))
		})

		It("returns no metrics if the event is not an app crash", func() {
			for _, appNonCrashEvent := range appNonCrashEvents {
				processedMetrics, err := processor.Process(appNonCrashEvent)
				Expect(err).ToNot(HaveOccurred())
				Expect(processedMetrics).To(BeEmpty())
			}
		})
	})
})

func str(s string) *string {
	return &s
}
