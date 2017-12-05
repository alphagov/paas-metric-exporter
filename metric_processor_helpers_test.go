package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"regexp"
	"sync"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
)

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

		DiskBytes: proto.Uint64(4),
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
