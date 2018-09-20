package senders

import (
	"time"

	"code.cloudfoundry.org/go-loggregator"
	"github.com/alphagov/paas-metric-exporter/metrics"
)

type LoggregatorSender struct {
	IngressClient *loggregator.IngressClient
}

func NewLoggregatorSender(url, caPath, certPath, keyPath string) (*LoggregatorSender, error) {
	ingressTLSConfig, err := loggregator.NewIngressTLSConfig(caPath, certPath, keyPath)
	if err != nil {
		return nil, err
	}
	client, err := loggregator.NewIngressClient(
		ingressTLSConfig,
		loggregator.WithAddr(url),
		loggregator.WithTag("origin", "paas-metric-exporter"),
	)
	if err != nil {
		return nil, err
	}
	return &LoggregatorSender{
		IngressClient: client,
	}, nil
}

func (ls *LoggregatorSender) Gauge(metric metrics.GaugeMetric) error {
	ls.IngressClient.EmitGauge(
		//TODO: The metrics package doesn't yet have a way to set the unit, so
		// we hardcode it here to "gauge"
		loggregator.WithGaugeValue(metric.Name(), float64(metric.Value), "gauge"),
		loggregator.WithGaugeSourceInfo(metric.GUID, metric.Instance),
		loggregator.WithEnvelopeTags(metric.GetLabels()),
	)
	return nil
}

func (ls *LoggregatorSender) Incr(metric metrics.CounterMetric) error {
	ls.IngressClient.EmitCounter(
		metric.Name(),
		loggregator.WithDelta(1),
		loggregator.WithEnvelopeTags(metric.GetLabels()),
		loggregator.WithCounterSourceInfo(metric.GUID, metric.Instance),
	)
	return nil
}

func (ls *LoggregatorSender) PrecisionTiming(metric metrics.PrecisionTimingMetric) error {
	ls.IngressClient.EmitTimer(
		metric.Name(),
		time.Unix(0, metric.Start),
		time.Unix(0, metric.Stop),
		loggregator.WithTimerSourceInfo(metric.GUID, metric.Instance),
		loggregator.WithEnvelopeTags(metric.GetLabels()),
	)
	return nil
}
