package metrics

import (
	"time"
)

var _ Metric = CounterMetric{}
var _ Metric = GaugeMetric{}
var _ Metric = FGaugeMetric{}
var _ Metric = TimingMetric{}
var _ Metric = PrecisionTimingMetric{}

//go:generate counterfeiter -o mocks/sender.go . Sender
type Sender interface {
	Gauge(metric GaugeMetric) error
	FGauge(metric FGaugeMetric) error
	Incr(metric CounterMetric) error
	Timing(metric TimingMetric) error
	PrecisionTiming(metric PrecisionTimingMetric) error
}

//go:generate counterfeiter -o mocks/metric.go . Metric
type Metric interface {
	Send(sender Sender) error
	Name() string
}

type CounterMetric struct {
	App          string
	CellId       string
	GUID         string
	Instance     string
	Job          string
	Metric       string
	Organisation string
	Space        string

	Value int64
}

func (m CounterMetric) Name() string {
	return m.Metric
}

func (m CounterMetric) Send(sender Sender) error {
	return sender.Incr(m)
}

type GaugeMetric struct {
	App          string
	CellId       string
	GUID         string
	Instance     string
	Job          string
	Metric       string
	Organisation string
	Space        string

	Value int64
}

func (m GaugeMetric) Name() string {
	return m.Metric
}

func (m GaugeMetric) Send(sender Sender) error {
	return sender.Gauge(m)
}

type FGaugeMetric struct {
	App          string
	CellId       string
	GUID         string
	Instance     string
	Job          string
	Metric       string
	Organisation string
	Space        string

	Value float64
}

func (m FGaugeMetric) Name() string {
	return m.Metric
}

func (m FGaugeMetric) Send(sender Sender) error {
	return sender.FGauge(m)
}

type TimingMetric struct {
	App          string
	CellId       string
	GUID         string
	Instance     string
	Job          string
	Metric       string
	Organisation string
	Space        string

	Value int64
}

func (m TimingMetric) Name() string {
	return m.Metric
}

func (m TimingMetric) Send(sender Sender) error {
	return sender.Timing(m)
}

type PrecisionTimingMetric struct {
	App          string
	CellId       string
	GUID         string
	Instance     string
	Job          string
	Metric       string
	Organisation string
	Space        string

	Value time.Duration
}

func (m PrecisionTimingMetric) Name() string {
	return m.Metric
}

func (m PrecisionTimingMetric) Send(sender Sender) error {
	return sender.PrecisionTiming(m)
}
