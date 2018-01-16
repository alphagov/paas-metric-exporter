package metrics

import (
	"time"
)

var _ Metric = CounterMetric{}
var _ Metric = GaugeMetric{}
var _ Metric = FGaugeMetric{}
var _ Metric = TimingMetric{}
var _ Metric = PrecisionTimingMetric{}

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
	tmplName, err := render(tmpl, m)
	if err != nil {
		return err
	}

	return statsdClient.Incr(tmplName, m.Value)
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

func (m GaugeMetric) Send(statsdClient StatsdClient, tmpl string) error {
	tmplName, err := render(tmpl, m)
	if err != nil {
		return err
	}

	return statsdClient.Gauge(tmplName, m.Value)
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

func (m FGaugeMetric) Send(statsdClient StatsdClient, tmpl string) error {
	tmplName, err := render(tmpl, m)
	if err != nil {
		return err
	}

	return statsdClient.FGauge(tmplName, m.Value)
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

func (m TimingMetric) Send(statsdClient StatsdClient, tmpl string) error {
	tmplName, err := render(tmpl, m)
	if err != nil {
		return err
	}

	return statsdClient.Timing(tmplName, m.Value)
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

func (m PrecisionTimingMetric) Send(statsdClient StatsdClient, tmpl string) error {
	tmplName, err := render(tmpl, m)
	if err != nil {
		return err
	}

	return statsdClient.PrecisionTiming(tmplName, m.Value)
}
