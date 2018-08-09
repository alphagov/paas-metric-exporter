package senders

import (
	"time"

	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/presenters"
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusSender struct {
	presenter     presenters.SnakeCasePresenter
	counterVecs   map[string]prometheus.CounterVec
	gaugeVecs     map[string]prometheus.GaugeVec
	histogramVecs map[string]prometheus.HistogramVec
	registerer    prometheus.Registerer
}

var _ metrics.Sender = &PrometheusSender{}

func NewPrometheusSender(registerer prometheus.Registerer) *PrometheusSender {
	presenter := presenters.NewSnakeCasePresenter()

	counterVecs := make(map[string]prometheus.CounterVec)
	gaugeVecs := make(map[string]prometheus.GaugeVec)
	histogramVecs := make(map[string]prometheus.HistogramVec)

	return &PrometheusSender{
		presenter,
		counterVecs,
		gaugeVecs,
		histogramVecs,
		registerer,
	}
}

func (s *PrometheusSender) Gauge(metric metrics.GaugeMetric) error {
	name := s.presenter.Present(metric.Name())

	gaugeVec, present := s.gaugeVecs[name]
	labelNames := s.buildLabelsFromMetric(metric)

	if !present {
		options := prometheus.GaugeOpts{Name: name, Help: " "}
		gaugeVec = *prometheus.NewGaugeVec(options, labelNames)

		s.registerer.MustRegister(gaugeVec)
		s.gaugeVecs[name] = gaugeVec
	}

	labels := s.labels(metric, labelNames)
	value := float64(metric.Value)

	gaugeVec.With(labels).Set(value)

	return nil
}

func (s *PrometheusSender) Incr(metric metrics.CounterMetric) error {
	name := s.presenter.Present(metric.Name())

	counterVec, present := s.counterVecs[name]
	labelNames := s.buildLabelsFromMetric(metric)

	if !present {
		options := prometheus.CounterOpts{Name: name, Help: " "}
		counterVec = *prometheus.NewCounterVec(options, labelNames)

		s.registerer.MustRegister(counterVec)
		s.counterVecs[name] = counterVec
	}

	labels := s.labels(metric, labelNames)
	value := float64(metric.Value)

	counterVec.With(labels).Add(value)

	return nil
}

func (s *PrometheusSender) PrecisionTiming(metric metrics.PrecisionTimingMetric) error {
	name := s.presenter.Present(metric.Name())

	histogramVec, present := s.histogramVecs[name]
	labelNames := s.buildLabelsFromMetric(metric)

	if !present {
		options := prometheus.HistogramOpts{Name: name, Help: " "}
		histogramVec = *prometheus.NewHistogramVec(options, labelNames)

		s.registerer.MustRegister(histogramVec)
		s.histogramVecs[name] = histogramVec
	}

	labels := s.labels(metric, labelNames)
	value := float64(metric.Value) / float64(time.Second)

	histogramVec.With(labels).Observe(value)

	return nil
}

func (s *PrometheusSender) labels(metric metrics.Metric, labelNames []string) prometheus.Labels {
	labels := make(prometheus.Labels)
	fields := map[string]string{}

	for mk, mv := range metric.GetLabels() {
		presented := s.presenter.Present(mk)
		fields[presented] = mv
	}

	for k, v := range fields {
		presented := s.presenter.Present(k)

		for _, n := range labelNames {
			if presented == n {
				labels[presented] = v
			}
		}
	}

	return labels
}
func (s *PrometheusSender) buildLabelsFromMetric(metric metrics.Metric) (labelNames []string) {
	for k := range metric.GetLabels() {
		presented := s.presenter.Present(k)
		labelNames = append(labelNames, presented)
	}

	return labelNames
}
