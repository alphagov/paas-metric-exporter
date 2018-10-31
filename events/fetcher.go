package events

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
)

//go:generate counterfeiter -o mocks/fetcher_process.go . FetcherProcess
type FetcherProcess interface {
	Run() error
}

type FetcherConfig struct {
	CFClientConfig  *cfclient.Config
	EventTypes      []sonde_events.Envelope_EventType
	UpdateFrequency time.Duration
}

type Fetcher struct {
	config         *FetcherConfig
	cfClient       *cfclient.Client
	appEventChan   chan *AppEvent
	newAppChan     chan string
	deletedAppChan chan string
	errorChan      chan error
	watchedApps    map[string]chan cfclient.App
	sync.RWMutex
}

func NewFetcher(
	config *FetcherConfig,
	appEventChan chan *AppEvent,
	newAppChan chan string,
	deletedAppChan chan string,
	errorChan chan error,
) *Fetcher {
	return &Fetcher{
		config:         config,
		appEventChan:   appEventChan,
		newAppChan:     newAppChan,
		deletedAppChan: deletedAppChan,
		errorChan:      errorChan,
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

func (m *Fetcher) Run() error {
	for {
		err := m.authenticate()
		if err != nil {
			return err
		}

		for {
			err := m.updateApps()
			if err != nil {
				if strings.Contains(err.Error(), `"error":"invalid_token"`) {
					log.Printf("Authentication error: %v\n", err)
					break
				}
				return err
			}

			time.Sleep(m.config.UpdateFrequency)
		}
	}
}

func (m *Fetcher) authenticate() (err error) {
	client, err := cfclient.NewClient(m.config.CFClientConfig)
	if err != nil {
		return err
	}

	m.cfClient = client
	return nil
}

func (m *Fetcher) startStream(app cfclient.App) chan cfclient.App {
	appChan := make(chan cfclient.App)
	go func() {
		tlsConfig := tls.Config{InsecureSkipVerify: m.config.CFClientConfig.SkipSslValidation}
		conn := consumer.New(m.cfClient.Endpoint.DopplerEndpoint, &tlsConfig, nil)
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

		eventTypesMap := make(map[sonde_events.Envelope_EventType]bool, len(m.config.EventTypes))
		for _, eventType := range m.config.EventTypes {
			eventTypesMap[eventType] = true
		}

		msgs, errs := conn.Stream(app.Guid, authToken)

		for i := 0; i < app.Instances; i++ {
			m.newAppChan <- fmt.Sprintf("%s:%d", app.Guid, i)
		}

		log.Printf("Started reading %s events\n", app.Name)
		for {
			select {
			case message, ok := <-msgs:
				if !ok {
					m.deletedAppChan <- app.Guid

					log.Printf("Stopped reading %s events\n", app.Name)
					return
				}
				stream := AppEvent{Envelope: message, App: app}
				if _, ok := eventTypesMap[*message.EventType]; ok {
					m.appEventChan <- &stream
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

				if updatedApp.Instances > app.Instances {
					for i := app.Instances; i < updatedApp.Instances; i++ {
						m.newAppChan <- fmt.Sprintf("%s:%d", app.Guid, i)
					}
				} else if updatedApp.Instances < app.Instances {
					for i := updatedApp.Instances; i < app.Instances; i++ {
						m.deletedAppChan <- fmt.Sprintf("%s:%d", app.Guid, i)
					}
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
	for i, _ := range apps {
		if err = updateAppSpaceData(&apps[i]); err != nil {
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
