package senders

import (
	"time"

	"code.cloudfoundry.org/go-loggregator"
	"fmt"
	"github.com/alphagov/paas-metric-exporter/metrics"
)

type LoggregatorSender struct {
	IngressClient *loggregator.IngressClient
}

type IngressClientLogger struct{}

func (icl *IngressClientLogger) Printf(str string, args ...interface{}) {
	fmt.Println(str, args)
}

func (icl *IngressClientLogger) Panicf(str string, args ...interface{}) {
	fmt.Println("panic", str, args)
	panic(args)
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
		loggregator.WithLogger(&IngressClientLogger{}),
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
		loggregator.WithGaugeValue(metric.Name(), float64(metric.Value), metric.Unit),
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

func (ls *LoggregatorSender) AppCreated(guid string) error {
	return nil
}

func (ls *LoggregatorSender) AppDeleted(guid string) error {
	return nil
}

func (ls *LoggregatorSender) ServiceCreated(guid string) error {
	return nil
}
