package metrics

import (
	"time"
)

var _ Metric = CounterMetric{}
var _ Metric = GaugeMetric{}
var _ Metric = PrecisionTimingMetric{}

//go:generate counterfeiter -o mocks/sender.go . Sender
type Sender interface {
	Gauge(metric GaugeMetric) error
	Incr(metric CounterMetric) error
	PrecisionTiming(metric PrecisionTimingMetric) error
}

//go:generate counterfeiter -o mocks/metric.go . Metric
type Metric interface {
	Send(sender Sender) error
	Name() string
	GetLabels() map[string]string
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
	Metadata     map[string]string

	Value int64
}

func (m CounterMetric) Name() string {
	return m.Metric
}

func (m CounterMetric) GetLabels() map[string]string {
	labels := map[string]string{
		"App":          m.App,
		"CellId":       m.CellId,
		"GUID":         m.GUID,
		"Instance":     m.Instance,
		"Job":          m.Job,
		"Organisation": m.Organisation,
		"Space":        m.Space,
	}
	for k, v := range m.Metadata {
		labels[k] = v
	}
	return labels
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
	Metadata     map[string]string

	Value int64
}

func (m GaugeMetric) Name() string {
	return m.Metric
}

func (m GaugeMetric) GetLabels() map[string]string {
	labels := map[string]string{
		"App":          m.App,
		"CellId":       m.CellId,
		"GUID":         m.GUID,
		"Instance":     m.Instance,
		"Job":          m.Job,
		"Organisation": m.Organisation,
		"Space":        m.Space,
	}
	for k, v := range m.Metadata {
		labels[k] = v
	}
	return labels
}

func (m GaugeMetric) Send(sender Sender) error {
	return sender.Gauge(m)
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
	Metadata     map[string]string

	Value time.Duration
}

func (m PrecisionTimingMetric) Name() string {
	return m.Metric
}

func (m PrecisionTimingMetric) GetLabels() map[string]string {
	labels := map[string]string{
		"App":          m.App,
		"CellId":       m.CellId,
		"GUID":         m.GUID,
		"Instance":     m.Instance,
		"Job":          m.Job,
		"Organisation": m.Organisation,
		"Space":        m.Space,
	}
	for k, v := range m.Metadata {
		labels[k] = v
	}
	return labels
}

func (m PrecisionTimingMetric) Send(sender Sender) error {
	return sender.PrecisionTiming(m)
}
