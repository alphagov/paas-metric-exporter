package events

import (
	"crypto/tls"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

type Fetcher struct {
	cfClient       *cfclient.Client
	cfClientConfig *cfclient.Config
	MsgChan        chan *AppEvent
	ErrorChan      chan error
	watchedApps    map[string]chan cfclient.App
	sync.RWMutex
}

func NewFetcher(cfClientConfig *cfclient.Config) *Fetcher {
	return &Fetcher{
		cfClientConfig: cfClientConfig,
		MsgChan:        make(chan *AppEvent),
		ErrorChan:      make(chan error),
		watchedApps:    make(map[string]chan cfclient.App),
	}
}

// RefreshAuthToken satisfies the `consumer.TokenRefresher` interface.
func (m *Fetcher) RefreshAuthToken() (token string, authError error) {
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

func (m *Fetcher) Run(updateFrequency int64) error {
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

func (m *Fetcher) authenticate() (err error) {
	client, err := cfclient.NewClient(m.cfClientConfig)
	if err != nil {
		return err
	}

	m.cfClient = client
	return nil
}

func (m *Fetcher) startStream(app cfclient.App) chan cfclient.App {
	appChan := make(chan cfclient.App)
	go func() {
		tlsConfig := tls.Config{InsecureSkipVerify: m.cfClientConfig.SkipSslValidation}
		conn := consumer.New(m.cfClient.Endpoint.DopplerEndpoint, &tlsConfig, nil)
		conn.RefreshTokenFrom(m)
		defer func() {
			m.Lock()
			defer m.Unlock()
			delete(m.watchedApps, app.Guid)
		}()

		authToken, err := m.cfClient.GetToken()
		if err != nil {
			m.ErrorChan <- err
			return
		}
		msgs, errs := conn.Stream(app.Guid, authToken)
		for {
			select {
			case message, ok := <-msgs:
				if !ok {
					return
				}
				stream := AppEvent{Msg: message, App: app}
				if *message.EventType == events.Envelope_ContainerMetric {
					m.MsgChan <- &stream
				}
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				if err == nil {
					continue
				}
				m.ErrorChan <- err
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

func (m *Fetcher) getApps() ([]cfclient.App, error) {
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

func (m *Fetcher) isWatched(guid string) bool {
	m.RLock()
	defer m.RUnlock()
	_, exists := m.watchedApps[guid]
	return exists
}

func (m *Fetcher) updateApps() error {

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
