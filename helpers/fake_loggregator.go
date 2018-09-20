// Note: Copied from https://github.com/cloudfoundry/go-loggregator/blob/8ebcfd3c7377510fe5a45ded5110d7749b562606/servers_test.go

package helpers

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
)

type FakeLoggregatorIngressServer struct {
	ReceivedEnvelopes chan *loggregator_v2.Envelope
	Addr              string
	tlsConfig         *tls.Config
	grpcServer        *grpc.Server
	grpc.Stream
}

func NewFakeLoggregatorIngressServer(serverCert, serverKey, caCert string) (*FakeLoggregatorIngressServer, error) {
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tls.RequestClientCert,
		InsecureSkipVerify: false,
	}
	caCertBytes, err := ioutil.ReadFile(caCert)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertBytes)
	tlsConfig.RootCAs = caCertPool

	return &FakeLoggregatorIngressServer{
		tlsConfig:         tlsConfig,
		ReceivedEnvelopes: make(chan *loggregator_v2.Envelope, 1000),
		Addr:              "localhost:0",
	}, nil
}

func (t *FakeLoggregatorIngressServer) Sender(srv loggregator_v2.Ingress_SenderServer) error {
	return grpc.Errorf(codes.Unimplemented, "this function is not implemented")
}

func (t *FakeLoggregatorIngressServer) BatchSender(srv loggregator_v2.Ingress_BatchSenderServer) error {
	for {
		envBatch, err := srv.Recv()
		if err != nil {
			return err
		}

		for _, e := range envBatch.Batch {
			t.ReceivedEnvelopes <- e
		}
	}
	return nil
}

func (t *FakeLoggregatorIngressServer) Send(_ context.Context, b *loggregator_v2.EnvelopeBatch) (*loggregator_v2.SendResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "this endpoint is not yet implemented")
}

func (t *FakeLoggregatorIngressServer) Start() error {
	listener, err := net.Listen("tcp4", t.Addr)
	if err != nil {
		return err
	}
	t.Addr = listener.Addr().String()

	var opts []grpc.ServerOption
	if t.tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(t.tlsConfig)))
	}
	t.grpcServer = grpc.NewServer(opts...)
	loggregator_v2.RegisterIngressServer(t.grpcServer, t)

	go func() {
		if err := t.grpcServer.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	return nil
}

func (t *FakeLoggregatorIngressServer) Stop() {
	t.grpcServer.Stop()
}
