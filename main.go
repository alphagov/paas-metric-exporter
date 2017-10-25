package main

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
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
	watchedApps    map[string]chan cfclient.App
	sync.RWMutex
}

func (m *metricProcessor) RefreshAuthToken() (token string, authError error) {
	token, err := m.cfClient.GetToken()
	if err != nil {
		err := m.authenticate()

		if err != nil {
			return "", err
		}

		return m.cfClient.GetToken()
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
		watchedApps: make(map[string]chan cfclient.App),
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

func updateAppSpaceData(app *cfclient.App) error {
	if app.SpaceData == (cfclient.SpaceResource{}) {
		space, err := app.Space()
		if err != nil {
			return err
		}
		org, err := space.Org()
		if err != nil {
			return err
		}
		space.OrgData.Entity = org
		app.SpaceData.Entity = space
	}
	return nil
}

func (m *metricProcessor) startStream(app cfclient.App) chan cfclient.App {
	appChan := make(chan cfclient.App)
	go func() {
		conn := consumer.New(m.cfClient.Endpoint.DopplerEndpoint, &tls.Config{InsecureSkipVerify: *skipSSLValidation}, nil)
		conn.RefreshTokenFrom(m)
		defer func() {
			m.Lock()
			defer m.Unlock()
			delete(m.watchedApps, app.Guid)
		}()

		authToken, err := m.cfClient.GetToken()
		if err != nil {
			m.errorChan <- err
			return
		}
		msgs, errs := conn.Stream(app.Guid, authToken)
		for {
			select {
			case message, ok := <-msgs:
				if !ok {
					return
				}
				stream := metrics.Stream{Msg: message, App: app, Tmpl: *metricTemplate}
				if *message.EventType == events.Envelope_ContainerMetric {
					m.msgChan <- &stream
				}
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				if err == nil {
					continue
				}
				m.errorChan <- err
			case updatedApp, ok := <-appChan:
				if !ok {
					appChan = nil
					conn.Close()
					continue
				}
				app = updatedApp
			}
		}
	}()
	return appChan
}

func (m *metricProcessor) getApps() ([]cfclient.App, error) {
	q := url.Values{}
	apps, err := m.cfClient.ListAppsByQuery(q)
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		if err = updateAppSpaceData(&app); err != nil {
			return nil, err
		}
	}
	return apps, nil
}

func (m *metricProcessor) isWatched(guid string) bool {
	m.RLock()
	defer m.RUnlock()
	_, exists := m.watchedApps[guid]
	return exists
}

func (m *metricProcessor) updateApps() error {

	m.Lock()
	defer m.Unlock()

	apps, err := m.getApps()
	if err != nil {
		return err
	}

	running := map[string]bool{}
	for _, app := range apps {
		running[app.Guid] = true
		if appChan, ok := m.watchedApps[app.Guid]; ok {
			appChan <- app
		} else {
			appChan = m.startStream(app)
			m.watchedApps[app.Guid] = appChan
		}
	}

	for appGuid, appChan := range m.watchedApps {
		if ok := running[appGuid]; !ok {
			close(appChan)
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
