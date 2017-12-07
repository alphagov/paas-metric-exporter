package app

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alphagov/paas-cf-apps-statsd/events"
	"github.com/alphagov/paas-cf-apps-statsd/metrics"
	"github.com/alphagov/paas-cf-apps-statsd/processors"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
)

// Config is the application configuration
type Config struct {
	CFClientConfig       *cfclient.Config
	CFAppUpdateFrequency time.Duration
}

// Application is the main application logic
type Application struct {
	config       *Config
	processors   map[sonde_events.Envelope_EventType]processors.Processor
	eventFetcher events.FetcherProcess
	sender       metrics.StatsdClient
	appEventChan chan *events.AppEvent
	errorChan    chan error
	exitChan     chan bool
	logger       io.Writer
}

// NewApplication creates a new application instance
func NewApplication(
	config *Config,
	processors map[sonde_events.Envelope_EventType]processors.Processor,
	sender metrics.StatsdClient,
	logger io.Writer,
) *Application {
	eventTypes := make([]sonde_events.Envelope_EventType, 0, len(processors))
	for eventType := range processors {
		eventTypes = append(eventTypes, eventType)
	}
	fetcherConfig := &events.FetcherConfig{
		CFClientConfig:  config.CFClientConfig,
		EventTypes:      eventTypes,
		UpdateFrequency: config.CFAppUpdateFrequency,
	}
	appEventChan := make(chan *events.AppEvent)
	errorChan := make(chan error)
	eventFetcher := events.NewFetcher(fetcherConfig, appEventChan, errorChan)

	return &Application{
		config:       config,
		processors:   processors,
		sender:       sender,
		eventFetcher: eventFetcher,
		appEventChan: appEventChan,
		errorChan:    errorChan,
		exitChan:     make(chan bool),
		logger:       logger,
	}
}

// Run starts the application
func (a *Application) Run() {
	fmt.Fprintln(a.logger, "Starting")
	go a.runEventFetcher()

	for {
		select {
		case appEvent := <-a.appEventChan:
			processor, ok := a.processors[appEvent.Envelope.GetEventType()]
			if !ok {
				continue
			}

			processedMetrics, procErr := processor.Process(appEvent)
			if procErr != nil {
				fmt.Fprintf(a.logger, "processing metrics failed: %v\n", procErr)
				continue
			}

			for _, metric := range processedMetrics {
				if err := metric.Send(a.sender); err != nil {
					fmt.Fprintf(a.logger, "sending metrics failed: %v\n", err)
				}
			}
		case err := <-a.errorChan:
			fmt.Fprintf(a.logger, "fetching events failed: %v\n", err)
		case <-a.exitChan:
			return
		}
	}
}

// Stop stops the application
func (a *Application) Stop() {
	a.exitChan <- true
}

func (a *Application) runEventFetcher() {
	err := a.eventFetcher.Run()
	if err != nil {
		fmt.Fprintf(a.logger, "fetching events failed: %v\n", err)
		os.Exit(-1)
	}
}
