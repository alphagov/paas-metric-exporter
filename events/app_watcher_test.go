// given that I've made a new app watcher, When I call appName, I get the app name
package events

import (
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("AppWatcher", func() {
	var (
		appWatcher *AppWatcher
		// Apps        []cfclient.App
		registerer *prometheus.Registry
	)

	BeforeEach(func() {

		config := &cfclient.Config{
			ApiAddress:        "some/endpoint",
			SkipSslValidation: true,
			Username:          "barry",
			Password:          "password",
			ClientID:          "dummy_client_id",
			ClientSecret:      "dummy_client_secret",
		}

		apps := []cfclient.App{
			{Guid: "33333333-3333-3333-3333-333333333333", Instances: 1, Name: "foo", SpaceURL: "/v2/spaces/123"},
		}

		log.Printf("app: %v", apps)
		log.Printf("app: %v", config)
		registerer = prometheus.NewRegistry()
		appWatcher = NewAppWatcher(config, apps[0], registerer)

		log.Printf("app: %v", appWatcher)

	})
	AfterEach(func() {})

	Describe("AppName", func() {
		It("knows the name of its application", func() {
			Expect(appWatcher.AppName()).To(Equal("foo"))
		})
	})

	Describe("processContainerMetrics", func() {
		It("sets a CPU metric on an instance", func() {
			cpuPercentage := 10.0
			var instanceIndex int32 = 0
			container_metric := sonde_events.ContainerMetric{
				CpuPercentage: &cpuPercentage,
				InstanceIndex: &instanceIndex,
			}
			appWatcher.processContainerMetric(&container_metric)
			metrics, _ := registerer.Gather()
			// log.Printf("%+v", metrics)
			log.Printf("%v", metrics[0].Metric[0].Gauge)
			// This prints a number that's far greater than 10...
			log.Printf("%d", metrics[0].Metric[0].Gauge.Value)
			// Expect(appWatcher.processContainerMetric(&container_metric).To(Equal(metrics[0].Name))
		})

	})
})
