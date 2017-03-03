package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"sync"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/onsi/gomega/ghttp"
	"golang.org/x/oauth2"
)

var _ = Describe("Main", func() {
	var (
		client  *cfclient.Client
		server  *ghttp.Server
		msgChan chan *events.Envelope
		errChan chan error
	)

	BeforeEach(func() {
		server = ghttp.NewServer()

		config := &cfclient.Config{
			ApiAddress: server.URL(),
			Username:   "user",
			Password:   "pass",
		}

		endpoint := cfclient.Endpoint{
			DopplerEndpoint: server.URL(),
			LoggingEndpoint: server.URL(),
			AuthEndpoint:    server.URL(),
			TokenEndpoint:   server.URL(),
		}

		token := oauth2.Token{
			AccessToken:  "access",
			TokenType:    "bearer",
			RefreshToken: "refresh",
		}

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v2/info"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, endpoint),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/oauth/token"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, token),
			),
		)

		var err error
		client, err = cfclient.NewClient(config)
		Expect(err).ToNot(HaveOccurred())

		msgChan = make(chan *events.Envelope, 10)
		errChan = make(chan error, 10)
	})

	AfterEach(func() {
		Expect(msgChan).To(BeEmpty())
		close(msgChan)
		Expect(errChan).To(BeEmpty())
		close(errChan)
	})

	Describe("updateApps", func() {
		var watchers AppMutex
		var apps []cfclient.App

		Context("no watchers and no running apps", func() {
			BeforeEach(func() {
				watchers = AppMutex{
					watch: map[string]*consumer.Consumer{},
					mutex: &sync.Mutex{},
				}
				apps = []cfclient.App{}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(apps)),
					),
				)
			})

			It("should not start any watchers", func() {
				Expect(updateApps(client, watchers, msgChan, errChan)).To(Succeed())
				Expect(watchers.watch).To(BeEmpty())
				Consistently(msgChan).Should(BeEmpty())
			})
		})

		Context("no watchers and three running apps", func() {
			BeforeEach(func() {
				watchers = AppMutex{
					watch: map[string]*consumer.Consumer{},
					mutex: &sync.Mutex{},
				}
				apps = []cfclient.App{
					{Guid: "11111111-1111-1111-1111-111111111111"},
					{Guid: "22222222-2222-2222-2222-222222222222"},
					{Guid: "33333333-3333-3333-3333-333333333333"},
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(apps)),
					),
				)
			})

			It("should start three watchers", func() {
				Expect(updateApps(client, watchers, msgChan, errChan)).To(Succeed())
				Expect(watchers.watch).To(HaveLen(len(apps)))
				Consistently(msgChan).Should(BeEmpty())
			})
		})
	})

	Describe("testAppResponse", func() {
		var apps []cfclient.App

		Context("no apps", func() {
			BeforeEach(func() {
				apps = []cfclient.App{}
			})

			It("should return a single page with no apps", func() {
				Expect(
					testAppResponse(apps),
				).To(
					Equal(cfclient.AppResponse{
						Count:     0,
						Pages:     1,
						Resources: []cfclient.AppResource{},
					}),
				)
			})
		})

		Context("three apps", func() {
			BeforeEach(func() {
				apps = []cfclient.App{
					{Guid: "11111111-1111-1111-1111-111111111111"},
					{Guid: "22222222-2222-2222-2222-222222222222"},
					{Guid: "33333333-3333-3333-3333-333333333333"},
				}
			})

			It("should return a single page with three apps", func() {
				Expect(
					testAppResponse(apps),
				).To(
					Equal(cfclient.AppResponse{
						Count: 3,
						Pages: 1,
						Resources: []cfclient.AppResource{
							{
								Meta:   cfclient.Meta{Guid: apps[0].Guid},
								Entity: apps[0],
							}, {
								Meta:   cfclient.Meta{Guid: apps[1].Guid},
								Entity: apps[1],
							}, {
								Meta:   cfclient.Meta{Guid: apps[2].Guid},
								Entity: apps[2],
							},
						},
					}),
				)
			})
		})
	})
})

func testAppResponse(apps []cfclient.App) cfclient.AppResponse {
	resp := cfclient.AppResponse{
		Count:     len(apps),
		Pages:     1,
		Resources: make([]cfclient.AppResource, len(apps)),
	}

	for i, app := range apps {
		resp.Resources[i] = cfclient.AppResource{
			Meta:   cfclient.Meta{Guid: app.Guid},
			Entity: app,
		}
	}

	return resp
}
