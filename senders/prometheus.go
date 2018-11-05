package senders

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/presenters"
	"github.com/prometheus/client_golang/prometheus"
)

type appMetrics struct {
	counterVecs   map[string]prometheus.CounterVec
	gaugeVecs     map[string]prometheus.GaugeVec
	histogramVecs map[string]prometheus.HistogramVec
}

type PrometheusSender struct {
	presenter  presenters.SnakeCasePresenter
	appMetrics map[string]appMetrics
}

var _ metrics.Sender = &PrometheusSender{}

func NewPrometheusSender() *PrometheusSender {
	presenter := presenters.NewSnakeCasePresenter()

	return &PrometheusSender{
		presenter,
		make(map[string]appMetrics),
	}
}

func (s *PrometheusSender) Gauge(metric metrics.GaugeMetric) error {
	name := s.presenter.Present(metric.Name())

	appMetrics := s.getOrCreateAppMetrics(metric.GUID, metric.Instance)

	gaugeVec, present := appMetrics.gaugeVecs[name]
	labelNames := s.buildLabelsFromMetric(metric)

	if !present {
		options := prometheus.GaugeOpts{
			Name: name,
			Help: " ",
			ConstLabels: prometheus.Labels{
				"guid": metric.GUID,
				"instance": metric.Instance,
			},
		}
		gaugeVec = *prometheus.NewGaugeVec(options, labelNames)

		prometheus.MustRegister(gaugeVec)
		appMetrics.gaugeVecs[name] = gaugeVec
	}

	labels := s.labels(metric, labelNames)
	value := float64(metric.Value)

	gaugeVec.With(labels).Set(value)

	return nil
}

func (s *PrometheusSender) Incr(metric metrics.CounterMetric) error {
	name := s.presenter.Present(metric.Name())

	appMetrics := s.getOrCreateAppMetrics(metric.GUID, metric.Instance)

	counterVec, present := appMetrics.counterVecs[name]
	labelNames := s.buildLabelsFromMetric(metric)

	if !present {
		options := prometheus.CounterOpts{
			Name: name,
			Help: " ",
			ConstLabels: prometheus.Labels{
				"guid": metric.GUID,
				"instance": metric.Instance,
			},
		}
		counterVec = *prometheus.NewCounterVec(options, labelNames)

		prometheus.MustRegister(counterVec)
		appMetrics.counterVecs[name] = counterVec
	}

	labels := s.labels(metric, labelNames)
	value := float64(metric.Value)

	counterVec.With(labels).Add(value)

	return nil
}

func (s *PrometheusSender) PrecisionTiming(metric metrics.PrecisionTimingMetric) error {
	name := s.presenter.Present(metric.Name())

	appMetrics := s.getOrCreateAppMetrics(metric.GUID, metric.Instance)

	histogramVec, present := appMetrics.histogramVecs[name]
	labelNames := s.buildLabelsFromMetric(metric)

	if !present {
		options := prometheus.HistogramOpts{
			Name: name,
			Help: " ",
			ConstLabels: prometheus.Labels{
				"guid": metric.GUID,
				"instance": metric.Instance,
			},
		}
		histogramVec = *prometheus.NewHistogramVec(options, labelNames)

		prometheus.MustRegister(histogramVec)
		appMetrics.histogramVecs[name] = histogramVec
	}

	labels := s.labels(metric, labelNames)
	value := float64(metric.Value) / float64(time.Second)

	histogramVec.With(labels).Observe(value)

	return nil
}

func (s *PrometheusSender) getOrCreateAppMetrics(guid string, instance string) appMetrics {
	guidInstance := fmt.Sprintf("%s:%s", guid, instance)

	m, present := s.appMetrics[guidInstance]
	if !present {
		newM := appMetrics{
			counterVecs:   make(map[string]prometheus.CounterVec),
			gaugeVecs:     make(map[string]prometheus.GaugeVec),
			histogramVecs: make(map[string]prometheus.HistogramVec),
		}
		s.appMetrics[guidInstance] = newM
		return newM
	}
	return m
}

func (s *PrometheusSender) labels(metric metrics.Metric, labelNames []string) prometheus.Labels {
	labels := make(prometheus.Labels)
	fields := map[string]string{}

	for mk, mv := range metric.GetLabels() {
		switch mk {
		case "GUID", "CellId", "Job", "Instance":
			continue
		}
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
		switch k {
		case "GUID", "CellId", "Job", "Instance":
			continue
		}
		presented := s.presenter.Present(k)
		labelNames = append(labelNames, presented)
	}

	return labelNames
}

func (s PrometheusSender) AppInstanceCreated(guidInstance string) error {
	return nil
}

func (s PrometheusSender) AppInstanceDeleted(guidInstance string) error {
	appMetrics := s.appMetrics[guidInstance]
	for _, v := range appMetrics.counterVecs {
		v.Reset()
	}
	for _, v := range appMetrics.gaugeVecs {
		v.Reset()
	}
	for _, v := range appMetrics.histogramVecs {
		v.Reset()
	}
	delete(s.appMetrics, guidInstance)
	return nil
}
