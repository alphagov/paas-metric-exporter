package main

import (
	"fmt"
	"os"

	"github.com/alphagov/paas-cf-apps-statsd/events"
	"github.com/alphagov/paas-cf-apps-statsd/metrics"
	"github.com/alphagov/paas-cf-apps-statsd/processors"
	"github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	"github.com/quipo/statsd"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	apiEndpoint       = kingpin.Flag("api-endpoint", "API endpoint").Default("https://api.10.244.0.34.xip.io").OverrideDefaultFromEnvar("API_ENDPOINT").String()
	statsdEndpoint    = kingpin.Flag("statsd-endpoint", "Statsd endpoint").Default("10.244.11.2:8125").OverrideDefaultFromEnvar("STATSD_ENDPOINT").String()
	statsdPrefix      = kingpin.Flag("statsd-prefix", "Statsd prefix").Default("mycf.").OverrideDefaultFromEnvar("STATSD_PREFIX").String()
	username          = kingpin.Flag("username", "UAA username.").Default("").OverrideDefaultFromEnvar("USERNAME").String()
	password          = kingpin.Flag("password", "UAA password.").Default("").OverrideDefaultFromEnvar("PASSWORD").String()
	skipSSLValidation = kingpin.Flag("skip-ssl-validation", "Please don't").Default("false").OverrideDefaultFromEnvar("SKIP_SSL_VALIDATION").Bool()
	debug             = kingpin.Flag("debug", "Enable debug mode. This disables forwarding to statsd and prints to stdout").Default("false").OverrideDefaultFromEnvar("DEBUG").Bool()
	updateFrequency   = kingpin.Flag("update-frequency", "The time in seconds, that takes between each apps update call.").Default("300").OverrideDefaultFromEnvar("UPDATE_FREQUENCY").Int64()
	metricTemplate    = kingpin.Flag("metric-template", "The template that will form a new metric namespace.").Default("{{.Space}}.{{.App}}.{{.Instance}}.{{.Metric}}").OverrideDefaultFromEnvar("METRIC_TEMPLATE").String()
)

func main() {
	kingpin.Parse()

	cfClientConfig := &cfclient.Config{
		ApiAddress:        *apiEndpoint,
		SkipSslValidation: *skipSSLValidation,
		Username:          *username,
		Password:          *password,
	}
	eventFetcher := events.NewFetcher(cfClientConfig)

	go func() {
		for err := range eventFetcher.ErrorChan {
			fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		}
	}()

	go func() {
		for {
			err := eventFetcher.Run(*updateFrequency)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
		}
	}()

	sender := statsd.NewStatsdClient(*statsdEndpoint, *statsdPrefix)
	sender.CreateSocket()

	sendMetrics(eventFetcher.AppEventChan, sender)
}

func sendMetrics(msgChan chan (*events.AppEvent), sender *statsd.StatsdClient) {
	var processedMetrics []metrics.Metric
	var proc_err error

	processors := map[sonde_events.Envelope_EventType]processors.Processor{
		sonde_events.Envelope_ContainerMetric: processors.NewContainerMetricProcessor(*metricTemplate),
		sonde_events.Envelope_LogMessage:      processors.NewLogMessageProcessor(*metricTemplate),
	}

	for appEvent := range msgChan {
		processor, ok := processors[appEvent.Envelope.GetEventType()]
		if !ok {
			continue
		}
		processedMetrics, proc_err = processor.Process(appEvent)

		if proc_err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", proc_err.Error())
		} else {
			for _, metric := range processedMetrics {
				if *debug {
					fmt.Println(metric)
				} else {
					err := metric.Send(sender)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
					}
				}
			}
		}
		processedMetrics = nil
	}
}
