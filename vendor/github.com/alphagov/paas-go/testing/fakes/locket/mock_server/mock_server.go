package mock_server

import (
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket/grpcserver"
	"code.cloudfoundry.org/locket/models"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/tedsuo/ifrit"
	gcontext "golang.org/x/net/context"
	"os"
	"path"
)

func NewMockLocketServer(fixturesPath, lockingMode string) *MockLocketServer {
	return &MockLocketServer{
		fixturesPath: fixturesPath,
		handler: &testHandler{
			mode: lockingMode,
		},
	}
}

type MockLocketServer struct {
	ListenAddress string
	fixturesPath  string
	handler       *testHandler
	process       ifrit.Process
}

type testHandler struct {
	mode      string
	lockCount int
}

func (h *testHandler) Lock(ctx gcontext.Context, req *models.LockRequest) (*models.LockResponse, error) {
	h.lockCount++
	switch h.mode {
	case "alwaysGrantLock":
		return &models.LockResponse{}, nil
	case "neverGrantLock":
		return nil, models.ErrLockCollision
	case "grantLockAfterFiveAttempts":
		if h.lockCount <= 5 {
			return nil, models.ErrLockCollision
		} else {
			return &models.LockResponse{}, nil
		}
	case "grantLockOnceThenFail":
		if h.lockCount <= 1 {
			return &models.LockResponse{}, nil
		} else {
			return nil, models.ErrLockCollision
		}
	default:
		return nil, errors.New(fmt.Sprintf("Unexpected mode %s", h.mode))
	}
}
func (h *testHandler) Release(ctx gcontext.Context, req *models.ReleaseRequest) (*models.ReleaseResponse, error) {
	return &models.ReleaseResponse{}, nil
}
func (h *testHandler) Fetch(ctx gcontext.Context, req *models.FetchRequest) (*models.FetchResponse, error) {
	return &models.FetchResponse{}, nil
}
func (h *testHandler) FetchAll(ctx gcontext.Context, req *models.FetchAllRequest) (*models.FetchAllResponse, error) {
	return &models.FetchAllResponse{}, nil
}

func (server *MockLocketServer) Run() error {
	port, err := freeport.GetFreePort()
	if err != nil {
		return err
	}

	server.ListenAddress = fmt.Sprintf("127.0.0.1:%d", port)

	logger := lager.NewLogger("grpc")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	logger.Debug("LockingMode: " + server.handler.mode)
	certificate, err := tls.LoadX509KeyPair(
		path.Join(server.fixturesPath, "locket-server.cert.pem"),
		path.Join(server.fixturesPath, "locket-server.key.pem"),
	)
	if err != nil {
		return err
	}

	grpcServer := grpcserver.NewGRPCServer(logger, server.ListenAddress, &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}, server.handler)
	server.process = ifrit.Invoke(grpcServer)

	go func() {
		err = <-server.process.Wait()
		logger.Error("grpc server process exited", err)
	}()
	return nil
}

func (server *MockLocketServer) Kill() {
	server.process.Signal(os.Interrupt)
}
