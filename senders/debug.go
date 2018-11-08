package senders

import (
	"log"

	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/presenters"
)

type DebugSender struct {
	Prefix    string
	presenter presenters.PathPresenter
}

var _ metrics.Sender = DebugSender{}

func NewDebugSender(statsdPrefix string, template string) (DebugSender, error) {
	presenter, err := presenters.NewPathPresenter(template)
	sender := DebugSender{Prefix: statsdPrefix, presenter: presenter}

	return sender, err
}

func (d DebugSender) Gauge(metric metrics.GaugeMetric) error {
	stat, err := d.presenter.Present(metric)
	if err != nil {
		return err
	}

	log.Printf("gauge %s %d\n", d.Prefix+stat, metric.Value)
	return nil
}

func (d DebugSender) Incr(metric metrics.CounterMetric) error {
	stat, err := d.presenter.Present(metric)
	if err != nil {
		return err
	}

	log.Printf("incr %s %d\n", d.Prefix+stat, metric.Value)
	return nil
}

func (d DebugSender) PrecisionTiming(metric metrics.PrecisionTimingMetric) error {
	stat, err := d.presenter.Present(metric)
	if err != nil {
		return err
	}

	log.Printf("timing %s %d\n", d.Prefix+stat, metric.Value)
	return nil
}

func (d DebugSender) AppCreated(guid string) error {
	log.Printf("app %s created\n", guid)
	return nil
}

func (d DebugSender) AppDeleted(guid string) error {
	log.Printf("app %s deleted\n", guid)
	return nil
}

func (d DebugSender) ServiceCreated(guid string) error {
	return nil
}