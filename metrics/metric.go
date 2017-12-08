package metrics

import (
	"time"
)

//go:generate counterfeiter -o mocks/statsd_client.go . StatsdClient
type StatsdClient interface {
	Gauge(stat string, value int64) error
	FGauge(stat string, value float64) error
	Incr(stat string, count int64) error
	Timing(string, int64) error
	PrecisionTiming(stat string, delta time.Duration) error
}

//go:generate counterfeiter -o mocks/metric.go . Metric
type Metric interface {
	Send(StatsdClient) error
}

type CounterMetric struct {
	Stat  string
	Value int64
}

type GaugeMetric struct {
	Stat  string
	Value int64
}

type FGaugeMetric struct {
	Stat  string
	Value float64
}

type TimingMetric struct {
	Stat  string
	Value int64
}

type PrecisionTimingMetric struct {
	Stat  string
	Value time.Duration
}

func NewCounterMetric(stat string, value int64) *CounterMetric {
	return &CounterMetric{
		Stat:  stat,
		Value: value,
	}
}

func NewGaugeMetric(stat string, value int64) *GaugeMetric {
	return &GaugeMetric{
		Stat:  stat,
		Value: value,
	}
}

func NewFGaugeMetric(stat string, value float64) *FGaugeMetric {
	return &FGaugeMetric{
		Stat:  stat,
		Value: value,
	}
}

func NewTimingMetric(stat string, value int64) *TimingMetric {
	return &TimingMetric{
		Stat:  stat,
		Value: value,
	}
}

func NewPrecisionTimingMetric(stat string, value time.Duration) *PrecisionTimingMetric {
	return &PrecisionTimingMetric{
		Stat:  stat,
		Value: value,
	}
}

func (m CounterMetric) Send(statsdClient StatsdClient) error {
	return statsdClient.Incr(m.Stat, m.Value)
}

func (m GaugeMetric) Send(statsdClient StatsdClient) error {
	return statsdClient.Gauge(m.Stat, m.Value)
}

func (m FGaugeMetric) Send(statsdClient StatsdClient) error {
	return statsdClient.FGauge(m.Stat, m.Value)
}

func (m TimingMetric) Send(statsdClient StatsdClient) error {
	return statsdClient.Timing(m.Stat, m.Value)
}

func (m PrecisionTimingMetric) Send(statsdClient StatsdClient) error {
	return statsdClient.PrecisionTiming(m.Stat, m.Value)
}
