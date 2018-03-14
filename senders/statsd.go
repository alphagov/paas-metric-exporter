package senders

import (
	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/presenters"
	quipo_statsd "github.com/quipo/statsd"
)

type StatsdSender struct {
	Client    *quipo_statsd.StatsdClient
	presenter presenters.PathPresenter
}

var _ metrics.Sender = StatsdSender{}

func NewStatsdSender(statsdEndpoint string, statsdPrefix string, template string) (StatsdSender, error) {
	presenter, err := presenters.NewPathPresenter(template)

	client := quipo_statsd.NewStatsdClient(statsdEndpoint, statsdPrefix)
	client.CreateSocket()

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

func (s StatsdSender) FGauge(metric metrics.FGaugeMetric) error {
	stat, err := s.presenter.Present(metric)
	if err != nil {
		return err
	}

	return s.Client.FGauge(stat, metric.Value)
}

func (s StatsdSender) Incr(metric metrics.CounterMetric) error {
	stat, err := s.presenter.Present(metric)
	if err != nil {
		return err
	}

	return s.Client.Incr(stat, metric.Value)
}

func (s StatsdSender) Timing(metric metrics.TimingMetric) error {
	stat, err := s.presenter.Present(metric)
	if err != nil {
		return err
	}

	return s.Client.Timing(stat, metric.Value)
}

func (s StatsdSender) PrecisionTiming(metric metrics.PrecisionTimingMetric) error {
	stat, err := s.presenter.Present(metric)
	if err != nil {
		return err
	}

	return s.Client.PrecisionTiming(stat, metric.Value)
}
