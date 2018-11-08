package app

import (
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	"code.cloudfoundry.org/locket/lock"
	locketmodels "code.cloudfoundry.org/locket/models"
	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/processors"
	"github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/satori/go.uuid"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
	"os"
)

// Config is the application configuration
type Config struct {
	CFClientConfig       *cfclient.Config
	CFAppUpdateFrequency time.Duration
	Whitelist            []string
	Template             string
	EnablePrometheus     bool
	PrometheusPort       int
	Loggregator          LoggregatorConfig
	locket.ClientLocketConfig
}

type LoggregatorConfig struct {
	MetronURL      string
	CACertPath     string
	ClientCertPath string
	ClientKeyPath  string
}

var DefaultLoggregatorConfig = LoggregatorConfig{
	MetronURL:      "localhost:3458",
	CACertPath:     "/var/vcap/jobs/metric-exporter/config/loggregator.ca_cert.crt",
	ClientCertPath: "/var/vcap/jobs/metric-exporter/config/loggregator.client_cert.crt",
	ClientKeyPath:  "/var/vcap/jobs/metric-exporter/config/loggregator.client_key.key",
}

var defaultLocketConfig = locket.ClientLocketConfig{
	LocketAddress:        "127.0.0.1:8891",
	LocketCACertFile:     "/var/vcap/jobs/metric-exporter/config/locket.ca_cert.crt",
	LocketClientCertFile: "/var/vcap/jobs/metric-exporter/config/locket.client_cert.crt",
	LocketClientKeyFile:  "/var/vcap/jobs/metric-exporter/config/locket.client_key.key",
}

func NewLocketConfig(addr, caCert, clientCert, clientKey *string) locket.ClientLocketConfig {
	locketConfig := defaultLocketConfig
	if *addr != "" {
		locketConfig.LocketAddress = *addr
	}
	if *caCert != "" {
		locketConfig.LocketCACertFile = *caCert
	}
	if *clientCert != "" {
		locketConfig.LocketClientCertFile = *clientCert
	}
	if *clientKey != "" {
		locketConfig.LocketClientKeyFile = *clientKey
	}
	return locketConfig
}

// Application is the main application logic
type Application struct {
	config         *Config
	processors     map[sonde_events.Envelope_EventType]processors.Processor
	eventFetcher   events.FetcherProcess
	senders        []metrics.Sender
	appEventChan   chan *events.AppEvent
	serviceEventChan chan *events.ServiceEvent
	newAppChan     chan string
	deletedAppChan chan string
	newServiceChan  chan string
	errorChan      chan error
	exitChan       chan bool
	logger         lager.Logger
}

// NewApplication creates a new application instance
func NewApplication(
	config *Config,
	logCacheAPI string,
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
	serviceEventChan := make(chan *events.ServiceEvent)
	newAppChan := make(chan string)
	newServiceChan := make(chan string)
	deletedAppChan := make(chan string)
	errorChan := make(chan error)
	eventFetcher := events.NewFetcher(fetcherConfig, appEventChan, logCacheAPI, newAppChan, newServiceChan, serviceEventChan, deletedAppChan, errorChan)

	logger := lager.NewLogger("metric-exporter")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	return &Application{
		config:         config,
		processors:     processors,
		senders:        senders,
		eventFetcher:   eventFetcher,
		appEventChan:   appEventChan,
		serviceEventChan: serviceEventChan,
		newAppChan:     newAppChan,
		newServiceChan: newServiceChan,
		deletedAppChan: deletedAppChan,
		errorChan:      errorChan,
		exitChan:       make(chan bool),
		logger:         logger,
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

func (a *Application) createLocketRunner() ifrit.Runner {
	var (
		err          error
		locketClient locketmodels.LocketClient
	)
	locketClient, err = locket.NewClient(a.logger, a.config.ClientLocketConfig)
	if err != nil {
		a.logger.Fatal("Failed to initialize locket client", err)
	}
	a.logger.Debug("connected-to-locket")
	id := uuid.NewV4()

	lockIdentifier := &locketmodels.Resource{
		Key:   "metric-exporter",
		Owner: id.String(),
		Type:  locketmodels.LockType,
	}

	return lock.NewLockRunner(
		a.logger,
		locketClient,
		lockIdentifier,
		locket.DefaultSessionTTLInSeconds,
		clock.NewClock(),
		locket.SQLRetryInterval,
	)
}

func (a *Application) Start(withLock bool) {
	if withLock {
		members := []grouper.Member{}
		locketRunner := a.createLocketRunner()

		members = append(members, grouper.Member{"locketRunner", locketRunner})
		members = append(members, grouper.Member{"app", a})
		group := grouper.NewOrdered(os.Interrupt, members)

		monitor := ifrit.Invoke(sigmon.New(group))
		err := <-monitor.Wait()
		if err != nil {
			a.logger.Error("process-group-stopped-with-error", err)
			os.Exit(1)
		}
	} else {
		a.run()
	}
}

func (a *Application) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	go a.run()
	a.logger.Info("started")
	sig := <-signals
	a.logger.Debug("received-signal", lager.Data{"signal": sig})
	a.Stop()
	return nil
}

func (a *Application) run() {
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
		case newApp := <-a.newAppChan:
			for _, sender := range a.senders {
				err := sender.AppCreated(newApp)
				if err != nil {
					log.Printf("registering app failed %v\n", err)
				}
			}
		case deletedApp := <-a.deletedAppChan:
			for _, sender := range a.senders {
				err := sender.AppDeleted(deletedApp)
				if err != nil {
					log.Printf("unregistering app failed %v\n", err)
				}
			}
		case newService := <-a.newServiceChan:
			log.Printf("%s is on the chan!!!", newService)
			for _, sender := range a.senders {
				err := sender.ServiceCreated(newService)
				if err != nil {
					log.Printf("registering service failed %v\n", err)
				}
			}
		case serviceEvent := <- a.serviceEventChan:
			log.Printf("TADA! Got event %v on the serviceEventChan", serviceEvent)
			log.Printf("Got metrics %v", serviceEvent.Envelope.GetGauge().Metrics)
			var metric *loggregator_v2.GaugeValue
			var metricName string
			for key, theMetric := range serviceEvent.Envelope.GetGauge().Metrics {
				metricName = key
				metric = theMetric
			}
			for _, sender := range a.senders {
				err := sender.Gauge(metrics.GaugeMetric{
					App: serviceEvent.Service.Name,
					Space: serviceEvent.Service.SpaceGuid,
					GUID: serviceEvent.Service.Guid,
					CellId: "TODO",
					Metric: metricName,
					Instance: "TODO",
					Job: "TODO",
					Metadata: map[string]string{},
					Organisation: "TODO",
					Unit: metric.Unit,
					Value: int64(metric.Value),
				})
				if err != nil {
					log.Printf("registering service failed %v\n", err)
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
		log.Fatalf("error running event fetcher: %v\n", err)
	}
}

func (a *Application) runPrometheusServer() {
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Starting prometheus server on port %d\n", a.config.PrometheusPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", a.config.PrometheusPort), nil))
}
