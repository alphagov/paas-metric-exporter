package events

import (
	"crypto/tls"
	"sync"
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
	sync.RWMutex       // TODO: what's this?
}

type InstanceMetrics struct {
	cpu prometheus.Gauge
}

func NewAppWatcher(
	config *cfclient.Config,
	app cfclient.App,
) *AppWatcher {
	return &AppWatcher{
		metricsForInstance: make([]InstanceMetrics, 2),
		config:             config,
		app:                app,
	}
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

	instanceNumber := m.app.Instances

	for i:=0; i<instanceNumber; i++ {
		m.metricsForInstance[i].cpu = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "cpu",
				Help: " ",
				ConstLabels: prometheus.Labels{
					"instance": fmt.Sprintf("%d", i),
				},
			},
		)

		prometheus.MustRegister(m.metricsForInstance[i].cpu)
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
				metric := message.GetContainerMetric()
				instance := m.metricsForInstance[metric.GetInstanceIndex()]
				// TODO: case where metrics recieved before cf tells us a new instance has been created
				instance.cpu.Set(metric.GetCpuPercentage())
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err == nil {
				continue
			}
			// m.errorChan <- err
			// case updatedApp, ok := <-appChan:
			// 	if !ok {
			// 		appChan = nil
			// 		conn.Close()
			// 		continue
			// 	}

			// 	if updatedApp.Instances > app.Instances {
			// 		for i := app.Instances; i < updatedApp.Instances; i++ {
			// 			m.newAppInstanceChan <- fmt.Sprintf("%s:%d", app.Guid, i)
			// 		}
			// 	} else if updatedApp.Instances < app.Instances {
			// 		for i := updatedApp.Instances; i < app.Instances; i++ {
			// 			m.deletedAppInstanceChan <- fmt.Sprintf("%s:%d", app.Guid, i)
			// 		}
			// 	}
			// 	app = updatedApp
		}
	}
}
