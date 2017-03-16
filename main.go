package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alphagov/paas-cf-apps-statsd/metrics"
	"github.com/alphagov/paas-cf-apps-statsd/processors"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
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
	metricTemplate    = kingpin.Flag("metric-template", "The template that will form a new metric namespace.").Default("{{.Space}}.{{.App}}.{{Instance}}.{{.Metric}}").OverrideDefaultFromEnvar("METRIC_TEMPLATE").String()
)

func main() {
	kingpin.Parse()

	c := &cfclient.Config{
		ApiAddress:        *apiEndpoint,
		SkipSslValidation: *skipSSLValidation,
		Username:          *username,
		Password:          *password,
	}

	containerMetricProcessor := processors.NewContainerMetricProcessor()

	sender := statsd.NewStatsdClient(*statsdEndpoint, *statsdPrefix)
	sender.CreateSocket()

	var processedMetrics []metrics.Metric
	var proc_err error

	msgChan := make(chan *metrics.Stream)
	errorChan := make(chan error)

	go func() {
		for err := range errorChan {
			fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		}
	}()
	go func() {
		for {
			err := metricProcessor(c, msgChan, errorChan)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
		}

	}()

	for wrapper := range msgChan {
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

func updateApps(client *cfclient.Client, watchedApps map[string]*consumer.Consumer, msgChan chan *metrics.Stream, errorChan chan error, newClient bool) error {
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
		if _, ok := watchedApps[app.Guid]; !ok {
			conn := consumer.New(client.Endpoint.DopplerEndpoint, &tls.Config{InsecureSkipVerify: *skipSSLValidation}, nil)
			msg, err := conn.Stream(app.Guid, authToken)

			go func(currentApp cfclient.App) {
				for m := range msg {
					stream := metrics.Stream{Msg: m, App: currentApp, Tmpl: *metricTemplate}

					msgChan <- &stream
				}
			}(app)

			go func() {
				for e := range err {
					if e != nil {
						errorChan <- e
					}
				}
			}()

			watchedApps[app.Guid] = conn
		} else if newClient {
			watchedApps[app.Guid].Stream(app.Guid, authToken)
		}
	}

	for appGuid, _ := range watchedApps {
		if _, ok := runningApps[appGuid]; !ok {
			watchedApps[appGuid].Close()
			delete(watchedApps, appGuid)
		}
	}

	return nil
}

func metricProcessor(c *cfclient.Config, msgChan chan *metrics.Stream, errorChan chan error) error {
	var newClient bool
	apps := make(map[string]*consumer.Consumer)

	for {
		client, err := cfclient.NewClient(c)
		newClient = true
		if err != nil {
			return err
		}

		for {
			err := updateApps(client, apps, msgChan, errorChan, newClient)
			newClient = false
			if err != nil {
				if strings.Contains(err.Error(), `"error":"invalid_token"`) {
					break
				}
				return err
			}

			time.Sleep(time.Duration(*updateFrequency) * time.Second)
		}
	}
}
