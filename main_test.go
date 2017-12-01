package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/alphagov/paas-cf-apps-statsd/metrics"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/onsi/gomega/ghttp"
	"golang.org/x/oauth2"
)

type TokenErr struct {
	Type        string `json:"error"`
	Description string `json:"error_description"`
}

var _ = Describe("Main", func() {
	var (
		apiServer  *ghttp.Server
		tcServer   *ghttp.Server
		tcHandler  testWebsocketHandler
		endpoint   cfclient.Endpoint
		token      oauth2.Token
		metricProc *metricProcessor
	)

	BeforeEach(func() {
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

		tcHandler = testWebsocketHandler{
			conns: map[string]*websocket.Conn{},
		}
		tcServer.RouteToHandler("GET", regexp.MustCompile(`/.*`), tcHandler.ServeHTTP)

		metricProc = &metricProcessor{
			cfClientConfig: &cfclient.Config{
				ApiAddress: apiServer.URL(),
				Username:   "user",
				Password:   "pass",
			},

			msgChan:     make(chan *metrics.Stream, 10),
			errorChan:   make(chan error, 10),
			watchedApps: make(map[string]chan cfclient.App),
		}

		metricProc.authenticate()
	})

	AfterEach(func() {
		Expect(metricProc.msgChan).To(BeEmpty())
		close(metricProc.msgChan)
		Expect(metricProc.errorChan).To(BeEmpty())
		close(metricProc.errorChan)
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
					{Guid: "33333333-3333-3333-3333-333333333333", Name: "foo", SpaceURL: "/v2/spaces/" + spaceGuid},
				}
				appsAfterRename = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", Name: "bar", SpaceURL: "/v2/spaces/" + spaceGuid},
				}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(appsBeforeRename)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(appsAfterRename)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
				)
			})

			It("update the metrics name", func() {
				Expect(metricProc.updateApps()).To(Succeed())

				var eventBeforeRename *metrics.Stream
				Eventually(metricProc.msgChan).Should(Receive(&eventBeforeRename))
				Expect(eventBeforeRename.App.Name).To(Equal("foo"))

				retrieveNewName := func() string {
					tcHandler.WriteMessage(appsBeforeRename[0].Guid)
					var eventAfterRename *metrics.Stream
					Eventually(metricProc.msgChan).Should(Receive(&eventAfterRename))
					return eventAfterRename.App.Name
				}

				Expect(metricProc.updateApps()).To(Succeed())
				Eventually(retrieveNewName).Should(Equal("bar"))
			})
		})

		Context("no watchers and no running apps", func() {
			BeforeEach(func() {
				apps = []cfclient.App{}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(apps)),
					),
				)
			})

			It("should not start any watchers", func() {
				Expect(metricProc.updateApps()).To(Succeed())
				Consistently(metricProc.msgChan).Should(BeEmpty())
				Expect(tcServer.ReceivedRequests()).To(HaveLen(0))
			})
		})

		Context("no watchers and three running apps", func() {
			BeforeEach(func() {
				apps = []cfclient.App{
					{Guid: "11111111-1111-1111-1111-111111111111", SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "22222222-2222-2222-2222-222222222222", SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "33333333-3333-3333-3333-333333333333", SpaceURL: "/v2/spaces/" + spaceGuid},
				}

				afterApps := []cfclient.App{}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(apps)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(afterApps)),
					),
				)
			})

			It("should start three watchers and disconnect when requested", func() {

				Expect(metricProc.updateApps()).To(Succeed())

				for _, app := range apps {
					guid := app.Guid
					inMap := func() bool {
						return metricProc.isWatched(guid)
					}
					Eventually(inMap).Should(BeTrue())
					Eventually(metricProc.msgChan).Should(Receive())
				}

				Expect(metricProc.updateApps()).To(Succeed())

				for _, app := range apps {
					guid := app.Guid
					inMap := func() bool {
						return metricProc.isWatched(guid)
					}
					Eventually(inMap).Should(BeFalse())
				}
			})
		})

		Context("three watchers and two apps replaced", func() {
			var appsBefore []cfclient.App

			BeforeEach(func() {
				appsBefore = []cfclient.App{
					{Guid: "11111111-1111-1111-1111-111111111111", SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "22222222-2222-2222-2222-222222222222", SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "33333333-3333-3333-3333-333333333333", SpaceURL: "/v2/spaces/" + spaceGuid},
				}
				apps = []cfclient.App{
					{Guid: "33333333-3333-3333-3333-333333333333", SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "44444444-4444-4444-4444-444444444444", SpaceURL: "/v2/spaces/" + spaceGuid},
					{Guid: "55555555-5555-5555-5555-555555555555", SpaceURL: "/v2/spaces/" + spaceGuid},
				}

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(appsBefore)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(apps)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/spaces/"+spaceGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testSpaceResource(spaceGuid, orgGuid)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/organizations/"+orgGuid),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testOrgResource(orgGuid)),
					),
				)
			})

			It("should stop two old watchers and start two new watchers", func() {
				Expect(metricProc.updateApps()).To(Succeed())
				for range appsBefore {
					Eventually(metricProc.msgChan).Should(Receive())
				}

				Expect(metricProc.updateApps()).To(Succeed())

				stoppedApps := appsBefore[:2]
				for _, app := range stoppedApps {
					guid := app.Guid
					inMap := func() bool {
						return metricProc.isWatched(guid)
					}
					Eventually(inMap).Should(BeFalse())
				}

				newApps := apps[1:]
				for _, app := range newApps {
					Eventually(metricProc.msgChan).Should(Receive())
					guid := app.Guid
					inMap := func() bool {
						return metricProc.isWatched(guid)
					}
					Eventually(inMap).Should(BeTrue())
				}
			})
		})
	})

	Describe("metricProcessor", func() {
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
					ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(apps)),
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
				var updateFrequency int64 = 1

				err := metricProc.process(updateFrequency)
				Eventually(err, 5*time.Second).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`"error":"invalid_token"`))
			})
		})
	})
})

var _ = Describe("test helpers", func() {
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

func testSpaceResource(spaceGuid, orgGuid string) cfclient.SpaceResource {
	return cfclient.SpaceResource{
		Meta:   cfclient.Meta{Guid: spaceGuid},
		Entity: cfclient.Space{OrgURL: "/v2/organizations/" + orgGuid},
	}
}

func testOrgResource(orgGuid string) cfclient.OrgResource {
	return cfclient.OrgResource{
		Meta:   cfclient.Meta{Guid: orgGuid},
		Entity: cfclient.Org{},
	}
}

func testNewEvent(appGuid string) *events.Envelope {
	metric := &events.ContainerMetric{
		ApplicationId: proto.String(appGuid),
		InstanceIndex: proto.Int32(1),
		CpuPercentage: proto.Float64(2),
		MemoryBytes:   proto.Uint64(3),
		DiskBytes:     proto.Uint64(4),
	}
	event := &events.Envelope{
		ContainerMetric: metric,
		EventType:       events.Envelope_ContainerMetric.Enum(),
		Origin:          proto.String("fake-origin-1"),
		Timestamp:       proto.Int64(time.Now().UnixNano()),
	}

	return event
}

type testWebsocketHandler struct {
	conns map[string]*websocket.Conn
	sync.RWMutex
}

func (t *testWebsocketHandler) WriteMessage(appGuid string) error {
	t.Lock()
	defer t.Unlock()
	Expect(t.conns).To(HaveKey(appGuid))
	conn := t.conns[appGuid]
	buf, _ := proto.Marshal(testNewEvent(appGuid))
	return conn.WriteMessage(websocket.BinaryMessage, buf)
}

func (t *testWebsocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	re := regexp.MustCompile(`/apps/([^/]+)/stream`)
	match := re.FindStringSubmatch(r.URL.Path)
	Expect(match).To(HaveLen(2), "unable to extract app GUID from request path")
	appGuid := match[1]

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	t.RLock()
	Expect(t.conns).ToNot(HaveKey(appGuid))
	t.RUnlock()

	conn, err := upgrader.Upgrade(w, r, nil)
	Expect(err).NotTo(HaveOccurred())
	t.Lock()
	t.conns[appGuid] = conn
	t.Unlock()
	defer conn.Close()

	cm := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	defer conn.WriteControl(websocket.CloseMessage, cm, time.Time{})

	err = t.WriteMessage(appGuid)
	Expect(err).ToNot(HaveOccurred())

	for {
		if _, _, err := conn.NextReader(); err != nil {
			break
		}
	}
}
