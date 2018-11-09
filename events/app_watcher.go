package events

import (
	"crypto/tls"
	"fmt"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	"github.com/prometheus/client_golang/prometheus"
)

//go:generate counterfeiter -o mocks/appWatcher_process.go . AppWatcherProcess
type AppWatcherProcess interface {
	Run() error
}

type AppWatcher struct {
	config             *cfclient.Config
	cfClient           *cfclient.Client
	metricsForInstance []InstanceMetrics
	app                cfclient.App
	appUpdateChan      chan cfclient.App
	registerer         prometheus.Registerer
}

type InstanceMetrics struct {
	cpu prometheus.Gauge
}

func NewInstanceMetrics(instanceIndex int, registerer prometheus.Registerer) InstanceMetrics {
	im := InstanceMetrics{
		cpu: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "cpu",
				Help: " ",
				ConstLabels: prometheus.Labels{
					"instance": fmt.Sprintf("%d", instanceIndex),
				},
			},
		),
	}
	registerer.MustRegister(im.cpu)
	return im
}

func NewAppWatcher(
	config *cfclient.Config,
	app cfclient.App,
	registerer prometheus.Registerer,
) *AppWatcher {
	appWatcher := &AppWatcher{
		metricsForInstance: make([]InstanceMetrics, 0),
		config:             config,
		app:                app,
		registerer:         registerer,
		appUpdateChan:      make(chan cfclient.App),
	}
	appWatcher.scaleTo(app.Instances)
	return appWatcher
}

// RefreshAuthToken satisfies the `consumer.TokenRefresher` interface.
func (m *AppWatcher) RefreshAuthToken() (token string, authError error) {
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

func (m *AppWatcher) authenticate() (err error) {
	client, err := cfclient.NewClient(m.config)
	if err != nil {
		return err
	}

	m.cfClient = client
	return nil
}

func (m *AppWatcher) Run() error {
	err := m.authenticate()
	if err != nil {
		return err
	}
	tlsConfig := tls.Config{InsecureSkipVerify: false} // TODO: is this needed?
	conn := consumer.New(m.cfClient.Endpoint.DopplerEndpoint, &tlsConfig, nil)
	conn.RefreshTokenFrom(m)

	authToken, err := m.cfClient.GetToken()
	if err != nil {
		return err
	}

	msgs, errs := conn.Stream(m.app.Guid, authToken)

	// log.Printf("Started reading %s events\n", app.Name)
	for {
		select {
		case message, ok := <-msgs:
			if !ok {
				// delete all instances

				return nil
			}
			switch message.GetEventType() {
			case sonde_events.Envelope_ContainerMetric:
				m.processContainerMetric(message.GetContainerMetric())
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err == nil {
				continue
			}
			// TODO: do something with errors
			break
			// m.errorChan <- err
		case updatedApp, ok := <-m.appUpdateChan:
			if !ok {
				conn.Close()
				m.scaleTo(0)
				break
			}

			if updatedApp.Instances != m.app.Instances {
				m.scaleTo(updatedApp.Instances)
			}
			m.app = updatedApp
		}
	}
}

func (m *AppWatcher) processContainerMetric(metric *sonde_events.ContainerMetric) {
	index := metric.GetInstanceIndex()
	if int(index) < len(m.metricsForInstance) {
		instance := m.metricsForInstance[index]
		instance.cpu.Set(metric.GetCpuPercentage())
	}
}

func (m *AppWatcher) AppName() string {
	return m.app.Name
}

func (m *AppWatcher) UpdateApp(app cfclient.App) {
	m.appUpdateChan <- app
}

func (m *AppWatcher) Close() {
	close(m.appUpdateChan)
}

func (m *AppWatcher) scaleTo(newInstanceCount int) {
	currentInstanceCount := len(m.metricsForInstance)

	if currentInstanceCount < newInstanceCount {
		for i := currentInstanceCount; i < newInstanceCount; i++ {
			m.metricsForInstance = append(m.metricsForInstance, NewInstanceMetrics(i, m.registerer))
		}
	} else {
		for i := currentInstanceCount; i > newInstanceCount; i-- {
			m.unregisterInstanceMetrics(i - 1)
		}
		m.metricsForInstance = m.metricsForInstance[0:newInstanceCount]
	}
}

func (m *AppWatcher) unregisterInstanceMetrics(instanceIndex int) {
	m.registerer.Unregister(m.metricsForInstance[instanceIndex].cpu)
}
