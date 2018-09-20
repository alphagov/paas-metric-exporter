package senders_test

import (
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"github.com/alphagov/paas-metric-exporter/helpers"
	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/senders"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("LoggregatorSender", func() {
	var (
		server *helpers.FakeLoggregatorIngressServer
		sender *senders.LoggregatorSender
		err    error
	)

	BeforeEach(func() {
		server, err = helpers.NewFakeLoggregatorIngressServer(
			"../fixtures/loggregator-server.cert.pem",
			"../fixtures/loggregator-server.key.pem",
			"../fixtures/ca.cert.pem",
		)
		Expect(err).NotTo(HaveOccurred())

		err = server.Start()
		Expect(err).NotTo(HaveOccurred())

		sender, err = senders.NewLoggregatorSender(
			server.Addr,
			"../fixtures/ca.cert.pem",
			"../fixtures/client.cert.pem",
			"../fixtures/client.key.pem",
		)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("#Incr", func() {
		It("should increment a counter metric", func() {
			err = sender.Incr(metrics.CounterMetric{
				Metric:   "crashes",
				Value:    1,
				GUID:     "guid1",
				Instance: "0",
			})
			Expect(err).ToNot(HaveOccurred())

			var envelope *loggregator_v2.Envelope
			Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))
			Expect(envelope.GetTimestamp()).To(
				BeNumerically(">=", time.Now().Add(-1*time.Minute).UnixNano()),
			)
			Expect(envelope.GetSourceId()).To(Equal("guid1"))
			Expect(envelope.GetCounter()).NotTo(BeNil())
			Expect(envelope.GetCounter().GetDelta()).To(Equal(uint64(1)))
		})
	})

	Describe("#Gauge", func() {
		It("should set a gauge", func() {
			err = sender.Gauge(metrics.GaugeMetric{
				Metric:   "cpu",
				Value:    100,
				GUID:     "guid2",
				Instance: "0",
			})
			Expect(err).ToNot(HaveOccurred())

			var envelope *loggregator_v2.Envelope
			Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))
			Expect(envelope.GetTimestamp()).To(
				BeNumerically(">=", time.Now().Add(-1*time.Minute).UnixNano()),
			)
			Expect(envelope.GetSourceId()).To(Equal("guid2"))
			Expect(envelope.GetGauge()).NotTo(BeNil())
			Expect(envelope.GetGauge().GetMetrics()).NotTo(BeNil())
			Expect(envelope.GetGauge().GetMetrics()).To(HaveKey("cpu"))
			Expect(envelope.GetGauge().GetMetrics()["cpu"].Value).To(Equal(float64(100)))
			Expect(envelope.GetGauge().GetMetrics()["cpu"].Unit).To(Equal("gauge"))
		})
	})

	Describe("#PrecisionTiming", func() {
		It("should set a timing", func() {
			startTime := time.Unix(0, 0).UnixNano() // Epoch
			stopTime := time.Unix(10, 0).UnixNano() // 10 seconds later
			err = sender.PrecisionTiming(metrics.PrecisionTimingMetric{
				Metric:   "responseTime",
				GUID:     "guid3",
				Instance: "0",
				Start:    startTime,
				Stop:     stopTime,
			})
			Expect(err).ToNot(HaveOccurred())

			var envelope *loggregator_v2.Envelope
			Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))
			Expect(envelope.GetTimestamp()).To(
				BeNumerically(">=", time.Now().Add(-1*time.Minute).UnixNano()),
			)
			Expect(envelope.GetSourceId()).To(Equal("guid3"))
			Expect(envelope.GetTimer()).NotTo(BeNil())
			Expect(envelope.GetTimer().GetStart()).To(Equal(startTime))
			Expect(envelope.GetTimer().GetStop()).To(Equal(stopTime))
		})
	})
})
