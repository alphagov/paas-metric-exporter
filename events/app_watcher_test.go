// given that I've made a new app watcher, When I call appName, I get the app name
package events

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

  "github.com/cloudfoundry-community/go-cfclient"
  "github.com/prometheus/client_golang/prometheus"

//  sonde_events "github.com/cloudfoundry/sonde-go/events"

)

var _ = Describe("AppWatcher", func() {
  var (
    appWatcher *AppWatcher
    app        cfclient.App
    registerer prometheus.Registerer
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

    appWatcher = NewAppWatcher(config, app,  registerer)

  })
  AfterEach(func() {})

  Describe("AppName", func() {
    It("update the metrics name", func() {
      Expect("baz").To(Equal("bar"))
    })
  })
})
