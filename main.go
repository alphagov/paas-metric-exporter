package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/pivotal-cf/graphite-nozzle/metrics"
	"github.com/pivotal-cf/graphite-nozzle/processors"
	"github.com/quipo/statsd"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	apiEndpoint       = kingpin.Flag("api-endpoint", "API endpoint").Default("https://api.10.244.0.34.xip.io").OverrideDefaultFromEnvar("API_ENDPOINT").String()
	statsdEndpoint    = kingpin.Flag("statsd-endpoint", "Statsd endpoint").Default("10.244.11.2:8125").OverrideDefaultFromEnvar("STATSD_ENDPOINT").String()
	statsdPrefix      = kingpin.Flag("statsd-prefix", "Statsd prefix").Default("mycf.").OverrideDefaultFromEnvar("STATSD_PREFIX").String()
	prefixJob         = kingpin.Flag("prefix-job", "Prefix metric names with job.index").Default("false").OverrideDefaultFromEnvar("PREFIX_JOB").Bool()
	username          = kingpin.Flag("username", "UAA username.").Default("").OverrideDefaultFromEnvar("USERNAME").String()
	password          = kingpin.Flag("password", "UAA password.").Default("").OverrideDefaultFromEnvar("PASSWORD").String()
	skipSSLValidation = kingpin.Flag("skip-ssl-validation", "Please don't").Default("false").OverrideDefaultFromEnvar("SKIP_SSL_VALIDATION").Bool()
	debug             = kingpin.Flag("debug", "Enable debug mode. This disables forwarding to statsd and prints to stdout").Default("false").OverrideDefaultFromEnvar("DEBUG").Bool()
	updateFrequency   = kingpin.Flag("update-frequency", "The time in seconds, that takes between each apps update call.").Default("300").OverrideDefaultFromEnvar("UPDATE_FREQUENCY").Int64()
	// metricTemplate    = kingpin.Flag("metric-template", "The template that will form a new metric namespace.").Default("").OverrideDefaultFromEnvar("METRIC_TEMPLATE").String()
)

func main() {
	kingpin.Parse()

	c := &cfclient.Config{
		ApiAddress:        *apiEndpoint,
		SkipSslValidation: *skipSSLValidation,
		Username:          *username,
		Password:          *password,
	}

	client, err := cfclient.NewClient(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	httpStartStopProcessor := processors.NewHttpStartStopProcessor()
	valueMetricProcessor := processors.NewValueMetricProcessor()
	containerMetricProcessor := processors.NewContainerMetricProcessor()
	counterProcessor := processors.NewCounterProcessor()

	sender := statsd.NewStatsdClient(*statsdEndpoint, *statsdPrefix)
	sender.CreateSocket()

	var processedMetrics []metrics.Metric
	var proc_err error

	msgChan := make(chan *events.Envelope)
	errorChan := make(chan error)

	go func() {
		for err := range errorChan {
			fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		}
	}()

	go func() {
		applications := AppMutex{}
		applications.watch = make(map[string]*consumer.Consumer)
		applications.mutex = &sync.Mutex{}

		for {
			err := updateApps(client, applications, msgChan, errorChan)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}

			time.Sleep(time.Duration(*updateFrequency) * time.Second)
		}
	}()

	for msg := range msgChan {
		eventType := msg.GetEventType()

		// graphite-nozzle can handle CounterEvent, ContainerMetric,
		// HttpStartStop and ValueMetric events
		switch eventType {
		case events.Envelope_ContainerMetric:
			processedMetrics, proc_err = containerMetricProcessor.Process(msg)
		case events.Envelope_CounterEvent:
			processedMetrics, proc_err = counterProcessor.Process(msg)
		case events.Envelope_HttpStartStop:
			processedMetrics, proc_err = httpStartStopProcessor.Process(msg)
		case events.Envelope_ValueMetric:
			processedMetrics, proc_err = valueMetricProcessor.Process(msg)
		default:
			// do nothing
		}

		if proc_err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", proc_err.Error())
		} else {
			// TODO We'd like to implement the metric template somewhere about here.
			if !*debug {
				if len(processedMetrics) > 0 {
					for _, metric := range processedMetrics {
						var prefix string
						if *prefixJob {
							prefix = msg.GetJob() + "." + msg.GetIndex()
						}
						metric.Send(sender, prefix)
					}
				}
			} else {
				for _, msg := range processedMetrics {
					fmt.Println(msg)
				}
			}
		}
		processedMetrics = nil
	}
}

// AppMutex should consit of a lock and the map of applications.
type AppMutex struct {
	watch map[string]*consumer.Consumer
	mutex *sync.Mutex
}

func updateApps(client *cfclient.Client, applications AppMutex, msgChan chan *events.Envelope, errorChan chan error) error {
	applications.mutex.Lock()
	defer applications.mutex.Unlock()

	authToken, err := client.GetToken()
	if err != nil {
		return err
	}

	apps, err := client.ListApps()
	if err != nil {
		return err
	}

	runningApps := map[string]bool{}
	for _, app := range apps {
		runningApps[app.Guid] = true
		if _, ok := applications.watch[app.Guid]; !ok {
			conn := consumer.New(client.Endpoint.DopplerEndpoint, &tls.Config{InsecureSkipVerify: *skipSSLValidation}, nil)
			msg, err := conn.Stream(app.Guid, authToken)

			go func() {
				for m := range msg {
					msgChan <- m
				}
			}()
			go func() {
				for e := range err {
					if e != nil {
						errorChan <- e
					}
				}
			}()

			applications.watch[app.Guid] = conn
		}
	}

	for appGuid, _ := range applications.watch {
		if _, ok := runningApps[appGuid]; !ok {
			applications.watch[appGuid].Close()
			delete(applications.watch, appGuid)
		}
	}

	return nil
}
