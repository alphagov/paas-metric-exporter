package logcache_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	logcache "code.cloudfoundry.org/go-log-cache"
	rpc "code.cloudfoundry.org/go-log-cache/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"google.golang.org/grpc"
)

// Assert that logcache.Reader is fulfilled by GroupReaderClient.BuildReader
var _ logcache.Reader = logcache.Reader(logcache.NewGroupReaderClient("").BuildReader(999))

func TestClientGroupRead(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.result["GET/v1/group/some-name"] = []byte(`{
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
	}`)
	client := logcache.NewGroupReaderClient(logCache.addr())

	reader := client.BuildReader(999)

	envelopes, err := reader(context.Background(), "some-name", time.Unix(0, 99))

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

	if logCache.reqs[0].URL.Path != "/v1/group/some-name" {
		t.Fatalf("expected Path '/v1/group/some-name' but got '%s'", logCache.reqs[0].URL.Path)
	}

	assertQueryParam(t, logCache.reqs[0].URL, "start_time", "99")
	assertQueryParam(t, logCache.reqs[0].URL, "requester_id", "999")

	if len(logCache.reqs[0].URL.Query()) != 2 {
		t.Fatalf("expected only two query parameters, but got %d", len(logCache.reqs[0].URL.Query()))
	}
}

func TestClientGroupReadWithOptions(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.result["GET/v1/group/some-name"] = []byte(`{
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
	}`)
	client := logcache.NewGroupReaderClient(logCache.addr())

	_, err := client.Read(
		context.Background(),
		"some-name",
		time.Unix(0, 99),
		999,
		logcache.WithEndTime(time.Unix(0, 101)),
		logcache.WithLimit(103),
		logcache.WithEnvelopeTypes(rpc.EnvelopeType_LOG),
	)

	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logCache.reqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.reqs))
	}

	if logCache.reqs[0].URL.Path != "/v1/group/some-name" {
		t.Fatalf("expected Path '/v1/group/some-name' but got '%s'", logCache.reqs[0].URL.Path)
	}

	assertQueryParam(t, logCache.reqs[0].URL, "start_time", "99")
	assertQueryParam(t, logCache.reqs[0].URL, "end_time", "101")
	assertQueryParam(t, logCache.reqs[0].URL, "limit", "103")
	assertQueryParam(t, logCache.reqs[0].URL, "envelope_types", "LOG")
	assertQueryParam(t, logCache.reqs[0].URL, "requester_id", "999")

	if len(logCache.reqs[0].URL.Query()) != 5 {
		t.Fatalf("expected only 5 query parameters, but got %d", len(logCache.reqs[0].URL.Query()))
	}
}

func TestClientGroupReadNon200(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.statusCode = 500
	client := logcache.NewGroupReaderClient(logCache.addr())

	_, err := client.Read(context.Background(), "some-id", time.Unix(0, 99), 999)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientGroupReadInvalidResponse(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.result["GET/v1/group/some-name"] = []byte("invalid")
	client := logcache.NewGroupReaderClient(logCache.addr())

	_, err := client.Read(context.Background(), "some-name", time.Unix(0, 99), 999)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientGroupReadUnknownAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewGroupReaderClient("http://invalid.url")

	_, err := client.Read(context.Background(), "some-id", time.Unix(0, 99), 999)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientGroupReadInvalidAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewGroupReaderClient("-:-invalid")

	_, err := client.Read(context.Background(), "some-id", time.Unix(0, 99), 999)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientGroupReadCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.block = true
	client := logcache.NewGroupReaderClient(logCache.addr())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Read(
		ctx,
		"some-id",
		time.Unix(0, 99),
		999,
		logcache.WithEndTime(time.Unix(0, 101)),
		logcache.WithLimit(103),
		logcache.WithEnvelopeTypes(rpc.EnvelopeType_LOG),
	)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGrpcClientGroupRead(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcGroupReader()
	client := logcache.NewGroupReaderClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	endTime := time.Now()

	envelopes, err := client.Read(context.Background(), "some-id", time.Unix(0, 99), 999,
		logcache.WithLimit(10),
		logcache.WithEndTime(endTime),
		logcache.WithEnvelopeTypes(rpc.EnvelopeType_LOG),
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

	if len(logCache.readReqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.readReqs))
	}

	if logCache.readReqs[0].RequesterId != 999 {
		t.Fatalf("expected RequesterId (%d) to equal %d", logCache.readReqs[0].RequesterId, 999)
	}

	if logCache.readReqs[0].StartTime != 99 {
		t.Fatalf("expected StartTime (%d) to equal %d", logCache.readReqs[0].StartTime, 99)
	}

	if logCache.readReqs[0].EndTime != endTime.UnixNano() {
		t.Fatalf("expected EndTime (%d) to equal %d", logCache.readReqs[0].EndTime, endTime.UnixNano())
	}

	if logCache.readReqs[0].Limit != 10 {
		t.Fatalf("expected Limit (%d) to equal %d", logCache.readReqs[0].Limit, 10)
	}

	if len(logCache.readReqs[0].EnvelopeTypes) == 0 {
		t.Fatalf("expected EnvelopeTypes to not be empty")
	}

	if logCache.readReqs[0].EnvelopeTypes[0] != rpc.EnvelopeType_LOG {
		t.Fatalf("expected EnvelopeTypes (%v) to equal %v", logCache.readReqs[0].EnvelopeTypes, rpc.EnvelopeType_LOG)
	}
}

func TestGrpcClientGroupReadCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcGroupReader()
	logCache.block = true
	client := logcache.NewGroupReaderClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Read(
		ctx,
		"some-id",
		time.Unix(0, 99),
		999,
		logcache.WithEndTime(time.Unix(0, 101)),
		logcache.WithLimit(103),
		logcache.WithEnvelopeTypes(rpc.EnvelopeType_LOG),
	)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientAddToGroup(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.result["PUT/v1/group/some-name/some-id"] = []byte("{}")
	client := logcache.NewGroupReaderClient(logCache.addr())

	err := client.AddToGroup(context.Background(), "some-name", "some-id")
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logCache.reqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.reqs))
	}

	if logCache.reqs[0].URL.Path != "/v1/group/some-name/some-id" {
		t.Fatalf("expected Path '/v1/group/some-name/some-id' but got '%s'", logCache.reqs[0].URL.Path)
	}

	if logCache.reqs[0].Method != "PUT" {
		t.Fatalf("expected Method to be PUT: %s", logCache.reqs[0].Method)
	}
}

func TestClientAddToGroupUnknownAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewGroupReaderClient("http://invalid.url")

	err := client.AddToGroup(context.Background(), "some-name", "some-id")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientAddToGroupInvalidAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewGroupReaderClient("-:-invalid")

	err := client.AddToGroup(context.Background(), "some-name", "some-id")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientAddToGroupNon200(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.statusCode = 500
	client := logcache.NewGroupReaderClient(logCache.addr())

	err := client.AddToGroup(context.Background(), "some-name", "some-id")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientAddToGroupCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.block = true
	client := logcache.NewGroupReaderClient(logCache.addr())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.AddToGroup(ctx, "some-name", "some-id")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGrpcClientAddToGroup(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcGroupReader()
	client := logcache.NewGroupReaderClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	err := client.AddToGroup(context.Background(), "some-name", "some-id")
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logCache.addReqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.addReqs))
	}

	if logCache.addReqs[0].Name != "some-name" {
		t.Fatalf("expected Name 'some-name' but got '%s'", logCache.addReqs[0].Name)
	}

	if logCache.addReqs[0].SourceId != "some-id" {
		t.Fatalf("expected SourceId 'some-id' but got '%s'", logCache.addReqs[0].SourceId)
	}

	logCache.addErr = errors.New("some-error")
	err = client.AddToGroup(context.Background(), "some-name", "some-id")
	if err == nil {
		t.Fatal("expected err")
	}
}

func TestClientRemoveFromGroup(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.result["DELETE/v1/group/some-name/some-id"] = []byte("{}")
	client := logcache.NewGroupReaderClient(logCache.addr())

	err := client.RemoveFromGroup(context.Background(), "some-name", "some-id")
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logCache.reqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.reqs))
	}

	if logCache.reqs[0].URL.Path != "/v1/group/some-name/some-id" {
		t.Fatalf("expected Path '/v1/group/some-name/some-id' but got '%s'", logCache.reqs[0].URL.Path)
	}

	if logCache.reqs[0].Method != "DELETE" {
		t.Fatalf("expected Method to be DELETE: %s", logCache.reqs[0].Method)
	}
}

func TestClientRemoveFromGroupUnknownAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewGroupReaderClient("http://invalid.url")

	err := client.RemoveFromGroup(context.Background(), "some-name", "some-id")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientRemoveFromGroupInvalidAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewGroupReaderClient("-:-invalid")

	err := client.RemoveFromGroup(context.Background(), "some-name", "some-id")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientRemoveFromGroupNon200(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.statusCode = 500
	client := logcache.NewGroupReaderClient(logCache.addr())

	err := client.RemoveFromGroup(context.Background(), "some-name", "some-id")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientRemoveFromGroupCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.block = true
	client := logcache.NewGroupReaderClient(logCache.addr())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.RemoveFromGroup(ctx, "some-name", "some-id")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGrpcClientRemoveFromGroup(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcGroupReader()
	client := logcache.NewGroupReaderClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	err := client.RemoveFromGroup(context.Background(), "some-name", "some-id")
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logCache.removeReqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.removeReqs))
	}

	if logCache.removeReqs[0].Name != "some-name" {
		t.Fatalf("expected Name 'some-name' but got '%s'", logCache.removeReqs[0].Name)
	}

	if logCache.removeReqs[0].SourceId != "some-id" {
		t.Fatalf("expected SourceId 'some-id' but got '%s'", logCache.removeReqs[0].SourceId)
	}

	logCache.removeErr = errors.New("some-error")
	err = client.RemoveFromGroup(context.Background(), "some-name", "some-id")
	if err == nil {
		t.Fatal("expected err")
	}
}

func TestClientGroupMeta(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()

	expectedResp := &rpc.GroupResponse{
		SourceIds:    []string{"a", "b"},
		RequesterIds: []uint64{1, 2},
	}

	data, err := json.Marshal(expectedResp)
	if err != nil {
		t.Fatal(err)
	}

	logCache.result["GET/v1/group/some-name/meta"] = data
	client := logcache.NewGroupReaderClient(logCache.addr())

	resp, err := client.Group(context.Background(), "some-name")
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logCache.reqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.reqs))
	}

	if logCache.reqs[0].URL.Path != "/v1/group/some-name/meta" {
		t.Fatalf("expected Path '/v1/group/some-name/meta' but got '%s'", logCache.reqs[0].URL.Path)
	}

	if logCache.reqs[0].Method != "GET" {
		t.Fatalf("expected Method to be GET: %s", logCache.reqs[0].Method)
	}

	if !reflect.DeepEqual(resp.SourceIDs, []string{"a", "b"}) {
		t.Fatalf(`expected SourceIds to equal: ["a", "b"]: %s`, resp.SourceIDs)
	}

	if !reflect.DeepEqual(resp.RequesterIDs, []uint64{1, 2}) {
		t.Fatalf(`expected RequesterIds to equal: [1, 2]: %s`, resp.RequesterIDs)
	}
}

func TestClientGroupsUnknownAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewGroupReaderClient("http://invalid.url")

	_, err := client.Group(context.Background(), "some-name")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientGroupInvalidAddr(t *testing.T) {
	t.Parallel()
	client := logcache.NewGroupReaderClient("-:-invalid")

	_, err := client.Group(context.Background(), "some-name")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientGroupNon200(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.statusCode = 500
	client := logcache.NewGroupReaderClient(logCache.addr())

	_, err := client.Group(context.Background(), "some-name")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientGroupInvalidResponse(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.result["GET/v1/group/some-name/meta"] = []byte("invalid")
	client := logcache.NewGroupReaderClient(logCache.addr())

	_, err := client.Group(context.Background(), "some-name")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestClientGroupCancelling(t *testing.T) {
	t.Parallel()
	logCache := newStubLogCache()
	logCache.block = true
	client := logcache.NewGroupReaderClient(logCache.addr())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Group(ctx, "some-name")

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGrpcClientGroup(t *testing.T) {
	t.Parallel()
	logCache := newStubGrpcGroupReader()
	client := logcache.NewGroupReaderClient(logCache.addr(), logcache.WithViaGRPC(grpc.WithInsecure()))

	resp, err := client.Group(context.Background(), "some-name")
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logCache.groupReqs) != 1 {
		t.Fatalf("expected have 1 request, have %d", len(logCache.groupReqs))
	}

	if logCache.groupReqs[0].Name != "some-name" {
		t.Fatalf("expected Name 'some-name' but got '%s'", logCache.groupReqs[0].Name)
	}

	if !reflect.DeepEqual(resp.SourceIDs, []string{"a", "b"}) {
		t.Fatalf(`expected SourceIds to equal: ["a", "b"]: %s`, resp.SourceIDs)
	}

	if !reflect.DeepEqual(resp.RequesterIDs, []uint64{1, 2}) {
		t.Fatalf(`expected RequesterIds to equal: [1, 2]: %s`, resp.RequesterIDs)
	}

	logCache.groupErr = errors.New("some-error")
	_, err = client.Group(context.Background(), "some-name")
	if err == nil {
		t.Fatal("expected err")
	}
}

type stubGrpcGroupReader struct {
	mu         sync.Mutex
	addReqs    []*rpc.AddToGroupRequest
	addErr     error
	removeReqs []*rpc.RemoveFromGroupRequest
	removeErr  error
	groupReqs  []*rpc.GroupRequest
	groupErr   error
	lis        net.Listener
	block      bool

	readReqs []*rpc.GroupReadRequest
	readErr  error
}

func newStubGrpcGroupReader() *stubGrpcGroupReader {
	s := &stubGrpcGroupReader{}
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	s.lis = lis
	srv := grpc.NewServer()
	rpc.RegisterGroupReaderServer(srv, s)
	go srv.Serve(lis)

	return s
}

func (s *stubGrpcGroupReader) addr() string {
	return s.lis.Addr().String()
}

func (s *stubGrpcGroupReader) AddToGroup(ctx context.Context, r *rpc.AddToGroupRequest) (*rpc.AddToGroupResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addReqs = append(s.addReqs, r)
	return &rpc.AddToGroupResponse{}, s.addErr
}

func (s *stubGrpcGroupReader) RemoveFromGroup(ctx context.Context, r *rpc.RemoveFromGroupRequest) (*rpc.RemoveFromGroupResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeReqs = append(s.removeReqs, r)
	return &rpc.RemoveFromGroupResponse{}, s.removeErr
}

func (s *stubGrpcGroupReader) Read(ctx context.Context, r *rpc.GroupReadRequest) (*rpc.GroupReadResponse, error) {
	if s.block {
		var block chan struct{}
		<-block
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.readReqs = append(s.readReqs, r)

	return &rpc.GroupReadResponse{
		Envelopes: &loggregator_v2.EnvelopeBatch{
			Batch: []*loggregator_v2.Envelope{
				{Timestamp: 99, SourceId: "some-id"},
				{Timestamp: 100, SourceId: "some-id"},
			},
		},
	}, nil
}

func (s *stubGrpcGroupReader) Group(ctx context.Context, r *rpc.GroupRequest) (*rpc.GroupResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groupReqs = append(s.groupReqs, r)
	return &rpc.GroupResponse{
		SourceIds:    []string{"a", "b"},
		RequesterIds: []uint64{1, 2},
	}, s.groupErr
}
