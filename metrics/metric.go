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
	Send(sender StatsdClient, template string) error
	Name() string
}

type CounterMetric struct {
	App          string
	CellId       string
	GUID         string
	Index        string
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

func (m CounterMetric) Send(statsdClient StatsdClient, tmpl string) error {
	return statsdClient.Incr(render(tmpl, m), m.Value)
}

type GaugeMetric struct {
	App          string
	CellId       string
	GUID         string
	Index        string
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

func (m GaugeMetric) Send(statsdClient StatsdClient, tmpl string) error {
	return statsdClient.Gauge(render(tmpl, m), m.Value)
}

type FGaugeMetric struct {
	App          string
	CellId       string
	GUID         string
	Index        string
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

func (m FGaugeMetric) Send(statsdClient StatsdClient, tmpl string) error {
	return statsdClient.FGauge(render(tmpl, m), m.Value)
}

type TimingMetric struct {
	App          string
	CellId       string
	GUID         string
	Index        string
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

func (m TimingMetric) Send(statsdClient StatsdClient, tmpl string) error {
	return statsdClient.Timing(render(tmpl, m), m.Value)
}

type PrecisionTimingMetric struct {
	App          string
	CellId       string
	GUID         string
	Index        string
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

func (m PrecisionTimingMetric) Send(statsdClient StatsdClient, tmpl string) error {
	return statsdClient.PrecisionTiming(render(tmpl, m), m.Value)
}
