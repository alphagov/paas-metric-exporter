package helpers_test

import (
	"time"

	loggregator "code.cloudfoundry.org/go-loggregator"

	"github.com/alphagov/paas-metric-exporter/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IngressClient", func() {
	var (
		server *helpers.FakeLoggregatorIngressServer
		client *loggregator.IngressClient
	)

	BeforeEach(func() {
		var err error
		server, err = helpers.NewFakeLoggregatorIngressServer(
			"../fixtures/loggregator-server.cert.pem",
			"../fixtures/loggregator-server.key.pem",
			"../fixtures/ca.cert.pem",
		)
		Expect(err).NotTo(HaveOccurred())

		err = server.Start()
		Expect(err).NotTo(HaveOccurred())

		tlsConfig, err := loggregator.NewIngressTLSConfig(
			"../fixtures/ca.cert.pem",
			"../fixtures/loggregator-server.cert.pem",
			"../fixtures/loggregator-server.key.pem",
		)
		Expect(err).NotTo(HaveOccurred())

		client, err = loggregator.NewIngressClient(
			tlsConfig,
			loggregator.WithAddr(server.Addr),
			loggregator.WithTag("origin", "test-loggregator-client"),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Stop()
	})

	It("should receive one envelope from one metric", func() {
		client.EmitGauge(
			loggregator.WithGaugeValue("test", 1, "s"),
		)

		Eventually(
			server.ReceivedEnvelopes,
			5*time.Second,
		).Should(Receive())
	})

	It("should receive three metrics envelope", func() {
		client.EmitGauge(
			loggregator.WithGaugeValue("test", 1, "s"),
		)

		Eventually(
			server.ReceivedEnvelopes,
			5*time.Second,
		).Should(Receive())

		client.EmitGauge(
			loggregator.WithGaugeValue("test", 2, "s"),
		)
		time.Sleep(200 * time.Millisecond)

		Eventually(
			server.ReceivedEnvelopes,
			5*time.Second,
		).Should(Receive())

		client.EmitGauge(
			loggregator.WithGaugeValue("test", 3, "s"),
		)
		time.Sleep(200 * time.Millisecond)

		Eventually(
			server.ReceivedEnvelopes,
			5*time.Second,
		).Should(Receive())

	})
})
