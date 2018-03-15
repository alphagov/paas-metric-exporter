package app

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/processors"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config is the application configuration
type Config struct {
	CFClientConfig       *cfclient.Config
	CFAppUpdateFrequency time.Duration
	Whitelist            []string
	Template             string
	EnablePrometheus     bool
	PrometheusPort       int
}

// Application is the main application logic
type Application struct {
	config       *Config
	processors   map[sonde_events.Envelope_EventType]processors.Processor
	eventFetcher events.FetcherProcess
	senders      []metrics.Sender
	appEventChan chan *events.AppEvent
	errorChan    chan error
	exitChan     chan bool
}

// NewApplication creates a new application instance
func NewApplication(
	config *Config,
	processors map[sonde_events.Envelope_EventType]processors.Processor,
	senders []metrics.Sender,
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
		senders:      senders,
		eventFetcher: eventFetcher,
		appEventChan: appEventChan,
		errorChan:    errorChan,
		exitChan:     make(chan bool),
	}
}

func (a *Application) enabled(name string) bool {
	if len(a.config.Whitelist) == 0 {
		return true
	}
	for _, prefix := range a.config.Whitelist {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// Run starts the application
func (a *Application) Run() {
	log.Println("Starting")
	go a.runEventFetcher()

	if a.config.EnablePrometheus {
		go a.runPrometheusServer()
	}

	for {
		select {
		case appEvent := <-a.appEventChan:
			processor, ok := a.processors[appEvent.Envelope.GetEventType()]
			if !ok {
				continue
			}

			processedMetrics, procErr := processor.Process(appEvent)
			if procErr != nil {
				log.Printf("processing metrics failed: %v\n", procErr)
				continue
			}

			for _, metric := range processedMetrics {
				if !a.enabled(metric.Name()) {
					continue
				}

				for _, sender := range a.senders {
					err := metric.Send(sender)
					if err != nil {
						log.Printf("sending metrics failed %v\n", err)
					}
				}
			}
		case err := <-a.errorChan:
			log.Printf("fetching events failed: %v\n", err)
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
		log.Fatalf("fetching events failed: %v\n", err)
	}
}

func (a *Application) runPrometheusServer() {
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Starting prometheus server on port %d\n", a.config.PrometheusPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", a.config.PrometheusPort), nil))
}
