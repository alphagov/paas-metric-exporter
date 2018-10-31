package senders

import (
	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/presenters"
	quipo_statsd "github.com/quipo/statsd"
)

type StatsdSender struct {
	Client    quipo_statsd.Statsd
	presenter presenters.PathPresenter
}

var _ metrics.Sender = StatsdSender{}

const DefaultTemplate = "{{.Space}}.{{.App}}.{{.Instance}}.{{.Metric}}"

func NewStatsdSender(client quipo_statsd.Statsd, template string) (StatsdSender, error) {
	presenter, err := presenters.NewPathPresenter(template)
	sender := StatsdSender{Client: client, presenter: presenter}

	return sender, err
}

func (s StatsdSender) Gauge(metric metrics.GaugeMetric) error {
	stat, err := s.presenter.Present(metric)
	if err != nil {
		return err
	}

	return s.Client.Gauge(stat, metric.Value)
}

func (s StatsdSender) Incr(metric metrics.CounterMetric) error {
	stat, err := s.presenter.Present(metric)
	if err != nil {
		return err
	}

	return s.Client.Incr(stat, metric.Value)
}

func (s StatsdSender) PrecisionTiming(metric metrics.PrecisionTimingMetric) error {
	stat, err := s.presenter.Present(metric)
	if err != nil {
		return err
	}

	return s.Client.PrecisionTiming(stat, metric.Value)
}

func (d StatsdSender) AppInstanceCreated(guidInstance string) error {
	return nil
}

func (d StatsdSender) AppInstanceDeleted(guidInstance string) error {
	return nil
}
