package main

import (
	"crypto/tls"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/alphagov/paas-cf-apps-statsd/metrics"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

type metricProcessor struct {
	cfClient       *cfclient.Client
	cfClientConfig *cfclient.Config
	msgChan        chan *metrics.Stream
	errorChan      chan error
	watchedApps    map[string]chan cfclient.App
	sync.RWMutex
}

// RefreshAuthToken satisfies the `consumer.TokenRefresher` interface.
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

func (m *metricProcessor) authenticate() (err error) {
	client, err := cfclient.NewClient(m.cfClientConfig)
	if err != nil {
		return err
	}

	m.cfClient = client
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
