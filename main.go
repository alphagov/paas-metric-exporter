package main

import (
	"log"
	"strings"
	"time"

	"github.com/alphagov/paas-metric-exporter/app"
	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/processors"
	"github.com/alphagov/paas-metric-exporter/senders"
	"github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	"github.com/prometheus/client_golang/prometheus"
	quipo_statsd "github.com/quipo/statsd"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
)

var (
	apiEndpoint         = kingpin.Flag("api-endpoint", "API endpoint").Default("https://api.10.244.0.34.xip.io").OverrideDefaultFromEnvar("API_ENDPOINT").String()
	statsdEndpoint      = kingpin.Flag("statsd-endpoint", "Statsd endpoint").Default("10.244.11.2:8125").OverrideDefaultFromEnvar("STATSD_ENDPOINT").String()
	statsdPrefix        = kingpin.Flag("statsd-prefix", "Statsd prefix").Default("mycf.").OverrideDefaultFromEnvar("STATSD_PREFIX").String()
	username            = kingpin.Flag("username", "UAA username.").Default("").OverrideDefaultFromEnvar("USERNAME").String()
	password            = kingpin.Flag("password", "UAA password.").Default("").OverrideDefaultFromEnvar("PASSWORD").String()
	clientID            = kingpin.Flag("client-id", "UAA client ID.").Default("").OverrideDefaultFromEnvar("CLIENT_ID").String()
	clientSecret        = kingpin.Flag("client-secret", "UAA client secret.").Default("").OverrideDefaultFromEnvar("CLIENT_SECRET").String()
	skipSSLValidation   = kingpin.Flag("skip-ssl-validation", "Please don't").Default("false").OverrideDefaultFromEnvar("SKIP_SSL_VALIDATION").Bool()
	debug               = kingpin.Flag("debug", "Enable debug mode. This disables forwarding to statsd and prometheus and prints to stdout").Default("false").OverrideDefaultFromEnvar("DEBUG").Bool()
	updateFrequency     = kingpin.Flag("update-frequency", "The time in seconds, that takes between each apps update call.").Default("300").OverrideDefaultFromEnvar("UPDATE_FREQUENCY").Int64()
	metricTemplate      = kingpin.Flag("metric-template", "The template that will form a new metric namespace.").Default(senders.DefaultTemplate).OverrideDefaultFromEnvar("METRIC_TEMPLATE").String()
	metricWhitelist     = kingpin.Flag("metric-whitelist", "Comma separated metric name prefixes to enable.").Default("").OverrideDefaultFromEnvar("METRIC_WHITELIST").String()
	prometheusBindPort  = kingpin.Flag("prometheus-bind-port", "The port to bind to for prometheus metrics.").Default("8080").OverrideDefaultFromEnvar("PORT").Int()
	prometheusMetricTTL = kingpin.Flag("prometheus-metric-ttl", "Time that a metric is kept in the prometheus exporter.").Default("60").OverrideDefaultFromEnvar("PROMETHEUS_METRIC_TTL").Int()
	enableStatsd        = kingpin.Flag("enable-statsd", "Enable the statsd sender.").Default("true").OverrideDefaultFromEnvar("ENABLE_STATSD").Bool()
	enablePrometheus    = kingpin.Flag("enable-prometheus", "Enable the prometheus sender.").Default("false").OverrideDefaultFromEnvar("ENABLE_PROMETHEUS").Bool()
	enableLoggregator   = kingpin.Flag("enable-loggregator", "Enable the Loggregator sender.").Default("false").OverrideDefaultFromEnvar("ENABLE_LOGGREGATOR").Bool()
	enableLocking       = kingpin.Flag("enable-locking", "Enable locking via Locket.").Default("false").OverrideDefaultFromEnvar("ENABLE_LOCKING").Bool()
	locketAddress       = kingpin.Flag("locket-address", "address:port of Locket server.").Default("127.0.0.1:8891").OverrideDefaultFromEnvar("LOCKET_API_LOCATION").String()
	locketCACert        = kingpin.Flag("locket-ca-cert", "File path to Locket CA certificate.").Default("").OverrideDefaultFromEnvar("LOCKET_CA_CERT").String()
	locketClientCert    = kingpin.Flag("locket-client-cert", "File path to Locket client certificate.").Default("").OverrideDefaultFromEnvar("LOCKET_CLIENT_CERT").String()
	locketClientKey     = kingpin.Flag("locket-client-key", "File path to Locket client key.").Default("").OverrideDefaultFromEnvar("LOCKET_CLIENT_KEY").String()
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
			ClientID:          *clientID,
			ClientSecret:      *clientSecret,
		},
		CFAppUpdateFrequency: time.Duration(*updateFrequency) * time.Second,
		Whitelist:            normalizeWhitelist(*metricWhitelist),
		Template:             *metricTemplate,
		EnablePrometheus:     *enablePrometheus,
		PrometheusPort:       *prometheusBindPort,
	}

	locketConfig := app.NewLocketConfig(locketAddress, locketCACert, locketClientCert, locketClientKey)
	config.ClientLocketConfig = locketConfig

	processors := map[sonde_events.Envelope_EventType]processors.Processor{
		sonde_events.Envelope_ContainerMetric: &processors.ContainerMetricProcessor{},
		sonde_events.Envelope_LogMessage:      &processors.LogMessageProcessor{},
		sonde_events.Envelope_HttpStartStop:   &processors.HttpStartStopProcessor{},
	}

	var metricSenders []metrics.Sender
	if *debug {
		debugSender, err := senders.NewDebugSender(*statsdPrefix, config.Template)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}
		metricSenders = append(metricSenders, debugSender)
	} else {
		if *enableStatsd {
			client := quipo_statsd.NewStatsdClient(*statsdEndpoint, *statsdPrefix)
			client.CreateSocket()

			statsDSender, err := senders.NewStatsdSender(client, config.Template)
			if err != nil {
				os.Stderr.WriteString(err.Error() + "\n")
				os.Exit(1)
			}

			metricSenders = append(metricSenders, statsDSender)
		}

		if *enablePrometheus {
			metricSenders = append(
				metricSenders,
				senders.NewPrometheusSender(
					prometheus.DefaultRegisterer,
					time.Duration(*prometheusMetricTTL)*time.Second,
				),
			)
		}

		if *enableLoggregator {
			loggregatorSender, err := senders.NewLoggregatorSender(
				app.DefaultLoggregatorConfig.MetronURL,
				app.DefaultLoggregatorConfig.CACertPath,
				app.DefaultLoggregatorConfig.ClientCertPath,
				app.DefaultLoggregatorConfig.ClientKeyPath,
			)
			if err != nil {
				os.Stderr.WriteString(err.Error() + "\n")
				os.Exit(1)
			}

			metricSenders = append(metricSenders, loggregatorSender)
		}
	}

	app := app.NewApplication(config, processors, metricSenders)
	app.Start(*enableLocking)
}
