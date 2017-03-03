package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/onsi/gomega/ghttp"
	"golang.org/x/oauth2"
)

var _ = Describe("Main", func() {
	var (
		client    *cfclient.Client
		apiServer *ghttp.Server
		tcServer  *ghttp.Server
		msgChan   chan *events.Envelope
		errChan   chan error
	)

	BeforeEach(func() {
		apiServer = ghttp.NewServer()
		tcServer = ghttp.NewServer()

		tcURL, err := url.Parse(tcServer.URL())
		Expect(err).ToNot(HaveOccurred())
		tcURL.Scheme = "ws"

		config := &cfclient.Config{
			ApiAddress: apiServer.URL(),
			Username:   "user",
			Password:   "pass",
		}

		endpoint := cfclient.Endpoint{
			DopplerEndpoint: tcURL.String(),
			LoggingEndpoint: tcURL.String(),
			AuthEndpoint:    apiServer.URL(),
			TokenEndpoint:   apiServer.URL(),
		}

		token := oauth2.Token{
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

				apiServer.AppendHandlers(
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

				apiServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v2/apps"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, testAppResponse(apps)),
					),
				)

				tcServer.RouteToHandler(
					"GET", "/apps/11111111-1111-1111-1111-111111111111/stream",
					testWebsocketHandler("11111111-1111-1111-1111-111111111111"),
				)
				tcServer.RouteToHandler(
					"GET", "/apps/22222222-2222-2222-2222-222222222222/stream",
					testWebsocketHandler("22222222-2222-2222-2222-222222222222"),
				)
				tcServer.RouteToHandler(
					"GET", "/apps/33333333-3333-3333-3333-333333333333/stream",
					testWebsocketHandler("33333333-3333-3333-3333-333333333333"),
				)
			})

			It("should start three watchers", func() {
				Expect(updateApps(client, watchers, msgChan, errChan)).To(Succeed())
				Expect(watchers.watch).To(HaveLen(len(apps)))

				for _, app := range apps {
					for i := 0; i < 3; i++ {
						Eventually(msgChan).Should(Receive())
					}
					Expect(watchers.watch[app.Guid].Close()).To(MatchError("websocket: close sent"))
					Eventually(errChan).Should(Receive(MatchError("EOF")))
				}
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

func testWebsocketHandler(appGuid string) func(w http.ResponseWriter, r *http.Request) {
	event := &events.Envelope{
		ContainerMetric: &events.ContainerMetric{
			ApplicationId: proto.String(appGuid),
			InstanceIndex: proto.Int32(1),
			CpuPercentage: proto.Float64(2),
			MemoryBytes:   proto.Uint64(3),
			DiskBytes:     proto.Uint64(4),
		},
		EventType: events.Envelope_ContainerMetric.Enum(),
		Origin:    proto.String("fake-origin-1"),
		Timestamp: proto.Int64(time.Now().UnixNano()),
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		Expect(err).NotTo(HaveOccurred())
		defer conn.Close()

		cm := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		defer conn.WriteControl(websocket.CloseMessage, cm, time.Time{})

		for i := 0; i < 3; i++ {
			buf, _ := proto.Marshal(event)
			err := conn.WriteMessage(websocket.BinaryMessage, buf)
			Expect(err).ToNot(HaveOccurred())
		}
	}
}
