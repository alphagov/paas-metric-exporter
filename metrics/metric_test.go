package metrics_test

import (
	"errors"

	. "github.com/alphagov/paas-metric-exporter/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"time"
)

type FakeStatsdClient struct {
	timingCalled          bool
	precisionTimingCalled bool
	incrCalled            bool
	gaugeCalled           bool
	fGaugeCalled          bool
	stat                  string
	value                 int64
	fValue                float64
	precisionTimingValue  time.Duration
}

func (f *FakeStatsdClient) Timing(stat string, delta int64) error {
	f.timingCalled = true
	f.stat = stat
	f.value = delta
	return nil
}

func (f *FakeStatsdClient) PrecisionTiming(stat string, delta time.Duration) error {
	f.precisionTimingCalled = true
	f.stat = stat
	f.precisionTimingValue = delta
	return nil
}

func (f *FakeStatsdClient) Incr(stat string, count int64) error {
	f.incrCalled = true
	f.stat = stat
	f.value = count
	return nil
}

func (f *FakeStatsdClient) Gauge(stat string, value int64) error {
	f.gaugeCalled = true
	f.stat = stat
	f.value = value
	return nil
}

func (f *FakeStatsdClient) FGauge(stat string, value float64) error {
	f.fGaugeCalled = true
	f.stat = stat
	f.fValue = value

	return errors.New("StatsdClientSendError")
}

var _ = Describe("Metric", func() {
	var (
		fakeStatsdClient *FakeStatsdClient
	)

	Describe("#NewCounterMetric", func() {
		It("creates a new CounterMetric", func() {
			metric := CounterMetric{Metric: "my.counter.metric", Value: 1}

			Expect(metric.Name()).To(Equal("my.counter.metric"))
			Expect(metric.Value).To(Equal(int64(1)))
		})
	})

	Describe("#NewGaugeMetric", func() {
		It("creates a new GaugeMetric", func() {
			metric := GaugeMetric{Metric: "my.gauge.metric", Value: 20}

			Expect(metric.Name()).To(Equal("my.gauge.metric"))
			Expect(metric.Value).To(Equal(int64(20)))
		})
	})

	Describe("#NewFGaugeMetric", func() {
		It("creates a new FGaugeMetric", func() {
			metric := FGaugeMetric{Metric: "my.fgauge.metric", Value: 20.25}

			Expect(metric.Name()).To(Equal("my.fgauge.metric"))
			Expect(metric.Value).To(Equal(float64(20.25)))
		})
	})

	Describe("#NewTimingMetric", func() {
		It("creates a new TimingMetric", func() {
			metric := TimingMetric{Metric: "my.timing.metric", Value: 100}

			Expect(metric.Name()).To(Equal("my.timing.metric"))
			Expect(metric.Value).To(Equal(int64(100)))
		})
	})

	Describe("#NewPrecisionTimingMetric", func() {
		It("creates a new PrecisionTimingMetric", func() {
			metric := PrecisionTimingMetric{Metric: "my.precision.timing.metric", Value: 100 * time.Millisecond}

			Expect(metric.Name()).To(Equal("my.precision.timing.metric"))
			Expect(metric.Value).To(Equal(100 * time.Millisecond))
		})
	})

	Describe("#Send", func() {
		BeforeEach(func() {
			fakeStatsdClient = new(FakeStatsdClient)
		})

		Context("with a PrecisionTimingMetric", func() {
			Context("without prefix", func() {
				It("sends the Metric to StatsD with time.Duration precision", func() {
					metric := PrecisionTimingMetric{Metric: "http.responsetimes.api_10_244_0_34_xip_io", Value: 50 * time.Millisecond}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.precisionTimingCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("http.responsetimes.api_10_244_0_34_xip_io"))
					Expect(fakeStatsdClient.precisionTimingValue).To(Equal(50 * time.Millisecond))
				})
			})

			Context("with prefix", func() {
				It("sends the Metric to StatsD with time.Duration precision", func() {
					metric := PrecisionTimingMetric{Metric: "http.responsetimes.api_10_244_0_34_xip_io", Value: 50 * time.Millisecond}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.precisionTimingCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("http.responsetimes.api_10_244_0_34_xip_io"))
					Expect(fakeStatsdClient.precisionTimingValue).To(Equal(50 * time.Millisecond))
				})
			})
		})

		Context("with a CounterMetric", func() {
			Context("without prefix", func() {
				It("sends the Metric to StatsD with int64 precision", func() {
					metric := CounterMetric{Metric: "http.statuscodes.api_10_244_0_34_xip_io.200", Value: 1}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.incrCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("http.statuscodes.api_10_244_0_34_xip_io.200"))
					Expect(fakeStatsdClient.value).To(Equal(int64(1)))
				})
			})

			Context("with prefix", func() {
				It("sends the Metric to StatsD with int64 precision", func() {
					metric := CounterMetric{Metric: "http.statuscodes.api_10_244_0_34_xip_io.200", Value: 1}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.incrCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("http.statuscodes.api_10_244_0_34_xip_io.200"))
					Expect(fakeStatsdClient.value).To(Equal(int64(1)))
				})
			})
		})

		Context("with a GaugeMetric", func() {
			Context("without prefix", func() {
				It("sends the Metric to StatsD with int64 precision", func() {
					metric := GaugeMetric{Metric: "router__0.numCPUS", Value: 4}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.gaugeCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("router__0.numCPUS"))
					Expect(fakeStatsdClient.value).To(Equal(int64(4)))
				})
			})

			Context("with prefix", func() {
				It("sends the Metric to StatsD with int64 precision", func() {
					metric := GaugeMetric{Metric: "router__0.numCPUS", Value: 4}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.gaugeCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("router__0.numCPUS"))
					Expect(fakeStatsdClient.value).To(Equal(int64(4)))
				})
			})
		})

		Context("with an FGaugeMetric", func() {
			Context("without prefix", func() {
				It("sends the Metric to StatsD with float64 precision", func() {
					metric := FGaugeMetric{Metric: "router__0.numCPUS", Value: 4}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.fGaugeCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("router__0.numCPUS"))
					Expect(fakeStatsdClient.fValue).To(Equal(float64(4)))
				})
			})

			Context("with prefix", func() {
				It("sends the Metric to StatsD with float64 precision", func() {
					metric := FGaugeMetric{Metric: "router__0.numCPUS", Value: 4}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.fGaugeCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("router__0.numCPUS"))
					Expect(fakeStatsdClient.fValue).To(Equal(float64(4)))
				})
			})
		})

		Context("with an TimingMetric", func() {
			Context("without prefix", func() {
				It("sends the Metric to StatsD with float64 precision", func() {
					metric := TimingMetric{Metric: "my.timing.metric", Value: 100}
					metric.Send(fakeStatsdClient)

					Expect(fakeStatsdClient.timingCalled).To(BeTrue())
					Expect(fakeStatsdClient.stat).To(Equal("my.timing.metric"))
					Expect(fakeStatsdClient.value).To(Equal(int64(100)))
				})
			})
		})

		Context("when the StatsdClient doesn't return an error", func() {
			It("doesn't return an error", func() {
				metric := GaugeMetric{Metric: "router__0.numCPUS", Value: 4}
				err := metric.Send(fakeStatsdClient)

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the StatsdClient returns an error", func() {
			It("returns the error", func() {
				metric := FGaugeMetric{Metric: "router__0.numCPUS", Value: 4}
				err := metric.Send(fakeStatsdClient)

				Expect(err).To(MatchError(errors.New("StatsdClientSendError")))
			})
		})
	})
})
