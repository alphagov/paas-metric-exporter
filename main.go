package main

import (
	"crypto/tls"
	"net/url"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alphagov/paas-cf-apps-statsd/metrics"
	"github.com/alphagov/paas-cf-apps-statsd/processors"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
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

type metricProcessor struct {
	cfClient       *cfclient.Client
	cfClientConfig *cfclient.Config
	msgChan        chan *metrics.Stream
	errorChan      chan error
	watchedApps    map[string]*consumer.Consumer
}

func (tr metricProcessor) RefreshAuthToken() (token string, authError error) {
	token, err := tr.cfClient.GetToken()
	if err != nil {
		err := tr.authenticate()

		if err != nil {
			return "", err
		}

		return tr.cfClient.GetToken()
	}

	return token, nil
}

func main() {
	kingpin.Parse()

	metricProc := &metricProcessor{
		cfClientConfig: &cfclient.Config{
			ApiAddress:        *apiEndpoint,
			SkipSslValidation: *skipSSLValidation,
			Username:          *username,
			Password:          *password,
		},

		msgChan:     make(chan *metrics.Stream),
		errorChan:   make(chan error),
		watchedApps: make(map[string]*consumer.Consumer),
	}

	containerMetricProcessor := processors.NewContainerMetricProcessor()

	sender := statsd.NewStatsdClient(*statsdEndpoint, *statsdPrefix)
	sender.CreateSocket()

	var processedMetrics []metrics.Metric
	var proc_err error

	go func() {
		for err := range metricProc.errorChan {
			fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		}
	}()

	go func() {
		for {
			err := metricProc.process(*updateFrequency)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
		}
	}()

	for wrapper := range metricProc.msgChan {
		processedMetrics, proc_err = containerMetricProcessor.Process(wrapper)

		if proc_err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", proc_err.Error())
		} else {
			if len(processedMetrics) > 0 {
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
		}
		processedMetrics = nil
	}
}

func (m *metricProcessor) authenticate() (err error) {
	client, err := cfclient.NewClient(m.cfClientConfig)
	if err != nil {
		return err
	}

	m.cfClient = client
	return nil
}

func (m *metricProcessor) updateApps() error {

	authToken, err := m.cfClient.GetToken()
	if err != nil {
		return err
	}

	q := url.Values{}
	apps, err := m.cfClient.ListAppsByQuery(q)
	if err != nil {
		return err
	}

	runningApps := map[string]bool{}
	for _, app := range apps {
		runningApps[app.Guid] = true
		if _, ok := m.watchedApps[app.Guid]; !ok {
			conn := consumer.New(m.cfClient.Endpoint.DopplerEndpoint, &tls.Config{InsecureSkipVerify: *skipSSLValidation}, nil)
			msg, err := conn.Stream(app.Guid, authToken)

			go func(currentApp cfclient.App) {
				for message := range msg {
					stream := metrics.Stream{Msg: message, App: currentApp, Tmpl: *metricTemplate}
					if (*message.EventType == events.Envelope_ContainerMetric) {
						m.msgChan <- &stream
					}
				}
			}(app)

			go func() {
				for e := range err {
					if e != nil {
						m.errorChan <- e
					}
				}
			}()

			conn.RefreshTokenFrom(m)

			m.watchedApps[app.Guid] = conn
		}
	}

	for appGuid, _ := range m.watchedApps {
		if _, ok := runningApps[appGuid]; !ok {
			m.watchedApps[appGuid].Close()
			delete(m.watchedApps, appGuid)
		}
	}

	return nil
}

func (m *metricProcessor) process(updateFrequency int64) error {

	for {
		err := m.authenticate()
		if err != nil {
			return err
		}

		for {
			err := m.updateApps()
			if err != nil {
				if strings.Contains(err.Error(), `"error":"invalid_token"`) {
					break
				}
				return err
			}

			time.Sleep(time.Duration(updateFrequency) * time.Second)
		}
	}
}
