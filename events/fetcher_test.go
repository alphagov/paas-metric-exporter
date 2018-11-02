package events

import (
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
	"github.com/gorilla/websocket"
	"github.com/onsi/gomega/ghttp"
	"golang.org/x/oauth2"
)

type TokenErr struct {
	Type        string `json:"error"`
	Description string `json:"error_description"`
}

var _ = Describe("Fetcher", func() {
	var (
		apiServer      *ghttp.Server
		tcServer       *ghttp.Server
		tcHandler      mockWebsocketHandler
		endpoint       cfclient.Endpoint
		token          oauth2.Token
		fetcher        *Fetcher
		appEventChan   chan *AppEvent
		newAppChan     chan string
		deletedAppChan chan string
		errorChan      chan error
	)

	BeforeEach(func() {
		log.SetOutput(GinkgoWriter)

		apiServer = ghttp.NewServer()
		tcServer = ghttp.NewServer()

		tcURL, err := url.Parse(tcServer.URL())
		Expect(err).ToNot(HaveOccurred())
		tcURL.Scheme = "ws"

		endpoint = cfclient.Endpoint{
			DopplerEndpoint: tcURL.String(),
			LoggingEndpoint: tcURL.String(),
			AuthEndpoint:    apiServer.URL(),
			TokenEndpoint:   apiServer.URL(),
		}

		token = oauth2.Token{
			AccessToken:  "access",
			TokenType:    "bearer",
			RefreshToken: "refresh",
		}

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v2/info"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, endpoint),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/oauth/token"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, token),
			),
		)

		tcHandler = mockWebsocketHandler{
			conns: map[string]*websocket.Conn{},
		}
		tcServer.RouteToHandler("GET", regexp.MustCompile(`/.*`), tcHandler.ServeHTTP)

		config := &FetcherConfig{
			CFClientConfig: &cfclient.Config{
				ApiAddress: apiServer.URL(),
				Username:   "user",
				Password:   "pass",
			},
			EventTypes: []sonde_events.Envelope_EventType{
				sonde_events.Envelope_ContainerMetric,
				sonde_events.Envelope_LogMessage,
			},
			UpdateFrequency: 1 * time.Second,
		}
		appEventChan = make(chan *AppEvent, 10)
		errorChan = make(chan error, 10)
		newAppChan = make(chan string, 10)
		deletedAppChan = make(chan string, 10)
		fetcher = NewFetcher(config, appEventChan, newAppChan, deletedAppChan, errorChan)
		fetcher.authenticate()
	})

	AfterEach(func() {
		Expect(appEventChan).To(BeEmpty())
		close(appEventChan)
		Expect(errorChan).To(BeEmpty())
		close(errorChan)
		Expect(newAppChan).To(BeEmpty())
		close(newAppChan)
		Expect(deletedAppChan).To(BeEmpty())
		close(deletedAppChan)
		log.SetOutput(os.Stdout)
	})

	Describe("updateApps", func() {
		var apps []cfclient.App

		const (
			spaceGuid = "25F4B52E-E16C-4E1F-A529-E45C2DB9AC07"
			orgGuid   = "98C535C1-A4F8-4066-8384-733E1ADCADEE"
		)

		Context("app is renamed", func() {
			var appsBeforeRename []cfclient.App
			var appsAfterRename []cfclient.App
			BeforeEach(func() {
				appsBeforeRename = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 1, Name: "foo", SpaceURL: "/v2/spaces/" + spaceGuid},
				}
				appsAfterRename = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 1, Name: "bar", SpaceURL: "/v2/spaces/" + spaceGuid},
				}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsBeforeRename)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsAfterRename)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
				)
			})

			It("update the metrics name", func() {
				Expect(fetcher.updateApps()).To(Succeed())

				var eventBeforeRename *AppEvent
				Eventually(appEventChan).Should(Receive(&eventBeforeRename))
				Eventually(newAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:0")))
				Expect(eventBeforeRename.App.Name).To(Equal("foo"))

				retrieveNewName := func() string {
					tcHandler.WriteMessage(appsBeforeRename[0].Guid)
					var eventAfterRename *AppEvent
					Eventually(appEventChan).Should(Receive(&eventAfterRename))
					return eventAfterRename.App.Name
				}

				Expect(fetcher.updateApps()).To(Succeed())
				Eventually(retrieveNewName).Should(Equal("bar"))
			})
		})

		Context("app instances are scaled down", func() {
			var appsBeforeScaleDown []cfclient.App
			var appsAfterScaleDown []cfclient.App
			BeforeEach(func() {
				appsBeforeScaleDown = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 2, Name: "foo", SpaceURL: "/v2/spaces/" + spaceGuid},
				}
				appsAfterScaleDown = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 1, Name: "foo", SpaceURL: "/v2/spaces/" + spaceGuid},
				}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsBeforeScaleDown)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsAfterScaleDown)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
				)
			})

			It("updates the number of instances", func() {
				Expect(fetcher.updateApps()).To(Succeed())
				Eventually(newAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:0")))
				Eventually(newAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:1")))

				Expect(fetcher.updateApps()).To(Succeed())
				Eventually(deletedAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:1")))
				// we are not really expecting appEventChan to have a message
				Eventually(appEventChan).Should(Receive())
			})
		})

		Context("app instances are scaled up", func() {
			var appsBeforeScaleUp []cfclient.App
			var appsAfterScaleUp []cfclient.App
			BeforeEach(func() {
				appsBeforeScaleUp = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 1, Name: "foo", SpaceURL: "/v2/spaces/" + spaceGuid},
				}
				appsAfterScaleUp = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 2, Name: "foo", SpaceURL: "/v2/spaces/" + spaceGuid},
				}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsBeforeScaleUp)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsAfterScaleUp)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
				)
			})

			It("updates the number of instances", func() {
				Expect(fetcher.updateApps()).To(Succeed())
				Eventually(newAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:0")))

				Expect(fetcher.updateApps()).To(Succeed())
				Eventually(newAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:1")))
				// we are not really expecting appEventChan to have a message
				Eventually(appEventChan).Should(Receive())
			})
		})

		Context("all app instances are deleted", func() {
			var appsBeforeDelete []cfclient.App
			var appsAfterDelete []cfclient.App
			BeforeEach(func() {
				appsBeforeDelete = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 2, Name: "foo", SpaceURL: "/v2/spaces/" + spaceGuid},
				}
				appsAfterDelete = []cfclient.App{}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsBeforeDelete)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsAfterDelete)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
				)
			})

			It("deletes all the app's instances", func() {
				Expect(fetcher.updateApps()).To(Succeed())
				Eventually(newAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:0")))
				Eventually(newAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:1")))
				Expect(fetcher.updateApps()).To(Succeed())

				Eventually(appEventChan).Should(Receive())
				Eventually(deletedAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:0")))
				Eventually(deletedAppChan).Should(Receive(Equal("33333333-3333-3333-3333-333333333333:1")))
			})
		})

		Context("no watchers and no running apps", func() {
			BeforeEach(func() {
				apps = []cfclient.App{}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(apps)),
					),
				)
			})

			It("should not start any watchers", func() {
				Expect(fetcher.updateApps()).To(Succeed())
				Consistently(appEventChan).Should(BeEmpty())
				Expect(tcServer.ReceivedRequests()).To(HaveLen(0))
			})
		})

		Context("no watchers and three running apps", func() {
			BeforeEach(func() {
				apps = []cfclient.App{
					{Guid: "11111111-1111-1111-1111-111111111111", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "22222222-2222-2222-2222-222222222222", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
				}

				afterApps := []cfclient.App{}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(apps)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(afterApps)),
					),
				)
			})

			It("should start three watchers and disconnect when requested", func() {

				Expect(fetcher.updateApps()).To(Succeed())

				for _, app := range apps {
					guid := app.Guid
					inMap := func() bool {
						return fetcher.isWatched(guid)
					}
					Eventually(inMap).Should(BeTrue())
					Eventually(appEventChan).Should(Receive())
					Eventually(newAppChan).Should(Receive())
				}

				Expect(fetcher.updateApps()).To(Succeed())

				for _, app := range apps {
					guid := app.Guid
					inMap := func() bool {
						return fetcher.isWatched(guid)
					}
					Eventually(inMap).Should(BeFalse())
					Eventually(deletedAppChan).Should(Receive())
				}
			})
		})

		Context("three watchers and two apps replaced", func() {
			var appsBefore []cfclient.App

			BeforeEach(func() {
				appsBefore = []cfclient.App{
					{Guid: "11111111-1111-1111-1111-111111111111", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "22222222-2222-2222-2222-222222222222", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
				}
				apps = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "44444444-4444-4444-4444-444444444444", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "55555555-5555-5555-5555-555555555555", Instances: 1, SpaceURL: "/v2/spaces/" + spaceGuid},
				}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(appsBefore)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(apps)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, mockOrgResource(orgGuid)),
					),
				)
			})

			It("should stop two old watchers and start two new watchers", func() {
				Expect(fetcher.updateApps()).To(Succeed())
				for range appsBefore {
					Eventually(appEventChan).Should(Receive())
					Eventually(newAppChan).Should(Receive())
				}

				Expect(fetcher.updateApps()).To(Succeed())

				stoppedApps := appsBefore[:2]
				for _, app := range stoppedApps {
					guid := app.Guid
					inMap := func() bool {
						return fetcher.isWatched(guid)
					}
					Eventually(inMap).Should(BeFalse())
					Eventually(deletedAppChan).Should(Receive())
				}

				newApps := apps[1:]
				for _, app := range newApps {
					Eventually(appEventChan).Should(Receive())
					guid := app.Guid
					inMap := func() bool {
						return fetcher.isWatched(guid)
					}
					Eventually(inMap).Should(BeTrue())
					Eventually(newAppChan).Should(Receive())
				}
			})
		})
	})

	Describe("Fetcher", func() {
		type tokenJSON struct {
			AccessToken  string        `json:"access_token"`
			TokenType    string        `json:"token_type"`
			RefreshToken string        `json:"refresh_token"`
			ExpiresIn    time.Duration `json:"expires_in"` // at least PayPal returns string, while most return number
			Expires      time.Duration `json:"expires"`    // broken Facebook spelling of expires_in
		}

		var (
			apps []cfclient.App
			tkj  tokenJSON
		)

		Context("refreshToken has expired", func() {
			BeforeEach(func() {
				tkj = tokenJSON{
					AccessToken:  "access",
					TokenType:    "bearer",
					RefreshToken: "refresh",
					ExpiresIn:    5,
					Expires:      5,
				}

				apps = []cfclient.App{
					{Guid: "55555555-5555-5555-5555-555555555555"},
				}

				tokenErr := TokenErr{
					Type:        "invalid_token",
					Description: "The token expired, was revoked, or the token ID is incorrect: xxxxx-r",
				}

				apiServer.RouteToHandler("GET", "/v2/apps",
					ghttp.RespondWithJSONEncoded(http.StatusOK, mockAppResponse(apps)),
				)
				apiServer.RouteToHandler("GET", "/v2/info",
					ghttp.RespondWithJSONEncoded(http.StatusOK, endpoint),
				)

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/oauth/token"),
						ghttp.VerifyForm(url.Values{"grant_type": []string{"password"}}),
						ghttp.RespondWithJSONEncoded(http.StatusOK, tkj),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/oauth/token"),
						ghttp.VerifyForm(url.Values{"grant_type": []string{"refresh_token"}}),
						ghttp.RespondWithJSONEncoded(http.StatusOK, tkj),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/oauth/token"),
						ghttp.VerifyForm(url.Values{"grant_type": []string{"refresh_token"}}),
						ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, tokenErr),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/oauth/token"),
						ghttp.VerifyForm(url.Values{"grant_type": []string{"password"}}),
						ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, tokenErr),
					),
				)
			})

			It("should try to refresh refreshToken", func() {
				err := fetcher.Run()
				Eventually(err, 5*time.Second).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`"error":"invalid_token"`))
			})
		})
	})
})
