package logcache_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"code.cloudfoundry.org/go-log-cache"
	rpc "code.cloudfoundry.org/go-log-cache/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"google.golang.org/grpc"
)

// Assert that logcache.Reader is fulfilled by Client.Read
var _ logcache.Reader = logcache.Reader(logcache.NewClient("").Read)

func TestClientRead(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	client := logcache.NewClient(logCache.addr())

	envelopes, err := client.Read(context.Background(), "some-id", time.Unix(0, 99))

	if err != nil {
		t.Fatal(err.Error())
	}

	if len(envelopes) != 2 {
		t.Fatalf("expected to receive 2 envelopes, got %d", len(envelopes))
	}

	if envelopes[0].Timestamp != 99 || envelopes[1].Timestamp != 100 {
		t.Fatal("wrong envelopes")
	}

	if len(logCache.reqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.reqs))
	}

	if logCache.reqs[0].URL.Path != "/v1/read/some-id" {
		t.Fatalf("expected Path '/v1/read/some-id' but got '%s'", logCache.reqs[0].URL.Path)
	}

	assertQueryParam(t, logCache.reqs[0].URL, "start_time", "99")

	if len(logCache.reqs[0].URL.Query()) != 1 {
		t.Fatalf("expected only a single query parameter, but got %d", len(logCache.reqs[0].URL.Query()))
	}
}

func TestGrpcClientRead(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcLogCache()
	client := logcache.NewClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	endTime := time.Now()

	envelopes, err := client.Read(context.Background(), "some-id", time.Unix(0, 99),
		logcache.WithLimit(10),
		logcache.WithEndTime(endTime),
		logcache.WithEnvelopeTypes(rpc.EnvelopeType_LOG),
		logcache.WithDescending(),
	)

	if err != nil {
		t.Fatal(err.Error())
	}

	if len(envelopes) != 2 {
		t.Fatalf("expected to receive 2 envelopes, got %d", len(envelopes))
	}

	if envelopes[0].Timestamp != 99 || envelopes[1].Timestamp != 100 {
		t.Fatal("wrong envelopes")
	}

	if len(logCache.reqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.reqs))
	}

	if logCache.reqs[0].SourceId != "some-id" {
		t.Fatalf("expected SourceId (%s) to equal %s", logCache.reqs[0].SourceId, "some-id")
	}

	if logCache.reqs[0].StartTime != 99 {
		t.Fatalf("expected StartTime (%d) to equal %d", logCache.reqs[0].StartTime, 99)
	}

	if logCache.reqs[0].EndTime != endTime.UnixNano() {
		t.Fatalf("expected EndTime (%d) to equal %d", logCache.reqs[0].EndTime, endTime.UnixNano())
	}

	if logCache.reqs[0].Limit != 10 {
		t.Fatalf("expected Limit (%d) to equal %d", logCache.reqs[0].Limit, 10)
	}

	if len(logCache.reqs[0].EnvelopeTypes) == 0 {
		t.Fatalf("expected to have EnvelopeTypes")
	}

	if logCache.reqs[0].EnvelopeTypes[0] != rpc.EnvelopeType_LOG {
		t.Fatalf("expected EnvelopeType (%v) to equal %v", logCache.reqs[0].EnvelopeTypes[0], rpc.EnvelopeType_LOG)
	}

	if !logCache.reqs[0].Descending {
		t.Fatalf("expected Descending to be set")
	}
}

func TestClientReadWithOptions(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	client := logcache.NewClient(logCache.addr())

	_, err := client.Read(
		context.Background(),
		"some-id",
		time.Unix(0, 99),
		logcache.WithEndTime(time.Unix(0, 101)),
		logcache.WithLimit(103),
		logcache.WithEnvelopeTypes(rpc.EnvelopeType_LOG, rpc.EnvelopeType_GAUGE),
		logcache.WithDescending(),
	)

	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logCache.reqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.reqs))
	}

	if logCache.reqs[0].URL.Path != "/v1/read/some-id" {
		t.Fatalf("expected Path '/v1/read/some-id' but got '%s'", logCache.reqs[0].URL.Path)
	}

	assertQueryParam(t, logCache.reqs[0].URL, "start_time", "99")
	assertQueryParam(t, logCache.reqs[0].URL, "end_time", "101")
	assertQueryParam(t, logCache.reqs[0].URL, "limit", "103")
	assertQueryParam(t, logCache.reqs[0].URL, "envelope_types", "LOG", "GAUGE")
	assertQueryParam(t, logCache.reqs[0].URL, "descending", "true")

	if len(logCache.reqs[0].URL.Query()) != 5 {
		t.Fatalf("expected 5 query parameters, but got %d", len(logCache.reqs[0].URL.Query()))
	}
}

func TestClientReadNon200(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.statusCode = 500
	client := logcache.NewClient(logCache.addr())

	_, err := client.Read(context.Background(), "some-id", time.Unix(0, 99))

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientReadInvalidResponse(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.result["GET/v1/read/some-id"] = []byte("invalid")
	client := logcache.NewClient(logCache.addr())

	_, err := client.Read(context.Background(), "some-id", time.Unix(0, 99))

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientReadUnknownAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewClient("http://invalid.url")

	_, err := client.Read(context.Background(), "some-id", time.Unix(0, 99))

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientReadInvalidAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewClient("-:-invalid")

	_, err := client.Read(context.Background(), "some-id", time.Unix(0, 99))

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientReadCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.block = true
	client := logcache.NewClient(logCache.addr())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Read(
		ctx,
		"some-id",
		time.Unix(0, 99),
		logcache.WithEndTime(time.Unix(0, 101)),
		logcache.WithLimit(103),
		logcache.WithEnvelopeTypes(rpc.EnvelopeType_LOG),
	)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGrpcClientReadCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcLogCache()
	logCache.block = true
	client := logcache.NewClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Read(
		ctx,
		"some-id",
		time.Unix(0, 99),
		logcache.WithEndTime(time.Unix(0, 101)),
		logcache.WithLimit(103),
		logcache.WithEnvelopeTypes(rpc.EnvelopeType_LOG),
	)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientMeta(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	client := logcache.NewClient(logCache.addr())

	logCache.result["GET/v1/meta"] = []byte(`{
		"meta": {
			"source-0": {},
			"source-1": {}
		}
	}`)

	meta, err := client.Meta(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(meta) != 2 {
		t.Fatalf("expected 2 sourceIDs: %d", len(meta))
	}

	if _, ok := meta["source-0"]; !ok {
		t.Fatal("did not find source-0")
	}

	if _, ok := meta["source-1"]; !ok {
		t.Fatal("did not find source-1")
	}
}

func TestClientMetaReturnsErrorWhenRequestFails(t *testing.T) {
	t.Parallel()

	client := logcache.NewClient("https://some-bad-addr")
	if _, err := client.Meta(context.Background()); err == nil {
		t.Fatal("did not error out on bad address")
	}
}

func TestClientMetaFailsOnNon200(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.statusCode = http.StatusNotFound
	client := logcache.NewClient(logCache.addr())

	_, err := client.Meta(context.Background())
	if err == nil {
		t.Fatal("did not error out on bad status code")
	}
}

func TestClientMetaFailsOnInvalidResponseBody(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.result["GET/v1/meta"] = []byte("not-real-result")
	client := logcache.NewClient(logCache.addr())

	_, err := client.Meta(context.Background())
	if err == nil {
		t.Fatal("did not error out on bad response body")
	}
}

func TestClientMetaCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.block = true
	client := logcache.NewClient(logCache.addr())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Meta(ctx)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGrpcClientMeta(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcLogCache()
	client := logcache.NewClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	meta, err := client.Meta(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(meta) != 2 {
		t.Fatalf("expected 2 sourceIDs: %d", len(meta))
	}

	if _, ok := meta["source-0"]; !ok {
		t.Fatal("did not find source-0")
	}

	if _, ok := meta["source-1"]; !ok {
		t.Fatal("did not find source-1")
	}
}

func TestGrpcClientMetaCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcLogCache()
	client := logcache.NewClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := client.Meta(ctx); err == nil {
		t.Fatal("expected an error")
	}
}

type stubLogCache struct {
	statusCode int
	server     *httptest.Server
	reqs       []*http.Request
	result     map[string][]byte
	block      bool
}

func newStubLogCache() *stubLogCache {
	s := &stubLogCache{
		statusCode: http.StatusOK,
		result: map[string][]byte{
			"GET/v1/read/some-id": []byte(`{
		"envelopes": {
			"batch": [
			    {
					"timestamp": 99,
					"sourceId": "some-id"
				},
			    {
					"timestamp": 100,
					"sourceId": "some-id"
				}
			]
		}
	}`),
		},
	}
	s.server = httptest.NewServer(s)
	return s
}

func (s *stubLogCache) addr() string {
	return s.server.URL
}

func (s *stubLogCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.block {
		var block chan struct{}
		<-block
	}

	s.reqs = append(s.reqs, r)
	w.WriteHeader(s.statusCode)
	w.Write(s.result[r.Method+r.URL.Path])
}

func assertQueryParam(t *testing.T, u *url.URL, name string, values ...string) {
	t.Helper()
	for _, value := range values {
		var found bool
		for _, actual := range u.Query()[name] {
			if actual == value {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("expected query parameter '%s' to contain '%s', but got '%v'", name, value, u.Query()[name])
		}
	}
}

type stubGrpcLogCache struct {
	mu    sync.Mutex
	reqs  []*rpc.ReadRequest
	lis   net.Listener
	block bool
}

func newStubGrpcLogCache() *stubGrpcLogCache {
	s := &stubGrpcLogCache{}
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	s.lis = lis
	srv := grpc.NewServer()
	rpc.RegisterEgressServer(srv, s)
	go srv.Serve(lis)

	return s
}

func (s *stubGrpcLogCache) addr() string {
	return s.lis.Addr().String()
}

func (s *stubGrpcLogCache) Read(c context.Context, r *rpc.ReadRequest) (*rpc.ReadResponse, error) {
	if s.block {
		var block chan struct{}
		<-block
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs = append(s.reqs, r)

	return &rpc.ReadResponse{
		Envelopes: &loggregator_v2.EnvelopeBatch{
			Batch: []*loggregator_v2.Envelope{
				{Timestamp: 99, SourceId: "some-id"},
				{Timestamp: 100, SourceId: "some-id"},
			},
		},
	}, nil
}

func (s *stubGrpcLogCache) Meta(context.Context, *rpc.MetaRequest) (*rpc.MetaResponse, error) {
	return &rpc.MetaResponse{
		Meta: map[string]*rpc.MetaInfo{
			"source-0": {},
			"source-1": {},
		},
	}, nil
}

func (s *stubGrpcLogCache) requests() []*rpc.ReadRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := make([]*rpc.ReadRequest, len(s.reqs))
	copy(r, s.reqs)
	return r
}
