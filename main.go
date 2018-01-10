package main

import (
	"log"
	"strings"
	"time"

	"github.com/alphagov/paas-metric-exporter/app"
	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/processors"
	"github.com/alphagov/paas-metric-exporter/statsd"
	"github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	quipo_statsd "github.com/quipo/statsd"
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
	metricWhitelist   = kingpin.Flag("metric-whitelist", "Comma separated metric name prefixes to enable.").Default("").OverrideDefaultFromEnvar("METRIC_WHITELIST").String()
)

func normalizePrefix(prefix string) string {
	prefix = strings.TrimRight(strings.TrimSpace(prefix), ".")
	if prefix == "" {
		return prefix
	}
	return prefix + "."
}

func normalizeWhitelist(csv string) []string {
	list := strings.Split(csv, ",")
	whitelist := make([]string, len(list))

	for i, val := range list {
		whitelist[i] = strings.TrimSpace(val)
	}

	return whitelist
}

func main() {
	kingpin.Parse()

	*statsdPrefix = normalizePrefix(*statsdPrefix)

	log.SetFlags(0)

	config := &app.Config{
		CFClientConfig: &cfclient.Config{
			ApiAddress:        *apiEndpoint,
			SkipSslValidation: *skipSSLValidation,
			Username:          *username,
			Password:          *password,
		},
		CFAppUpdateFrequency: time.Duration(*updateFrequency) * time.Second,
		Whitelist:            normalizeWhitelist(*metricWhitelist),
	}

	processors := map[sonde_events.Envelope_EventType]processors.Processor{
		sonde_events.Envelope_ContainerMetric: processors.NewContainerMetricProcessor(*metricTemplate),
		sonde_events.Envelope_LogMessage:      processors.NewLogMessageProcessor(*metricTemplate),
		sonde_events.Envelope_HttpStartStop:   processors.NewHttpStartStopProcessor(*metricTemplate),
	}

	var sender metrics.StatsdClient
	if !*debug {
		statsdSender := quipo_statsd.NewStatsdClient(*statsdEndpoint, *statsdPrefix)
		statsdSender.CreateSocket()
		sender = statsdSender
	} else {
		sender = statsd.DebugClient{Prefix: *statsdPrefix}
	}

	app := app.NewApplication(config, processors, sender)
	app.Run()
}
