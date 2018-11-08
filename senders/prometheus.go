package senders

import (
	"fmt"
	"strings"
	//	"time"

	"github.com/alphagov/paas-metric-exporter/metrics"
	"github.com/alphagov/paas-metric-exporter/presenters"
	"github.com/prometheus/client_golang/prometheus"
)

type appInstanceMetrics struct {
	counterVecs   map[string]prometheus.CounterVec
	gaugeVecs     map[string]prometheus.GaugeVec
	histogramVecs map[string]prometheus.HistogramVec
}

type PrometheusSender struct {
	presenter          presenters.SnakeCasePresenter
	appInstanceMetrics map[string]appInstanceMetrics
}

var _ metrics.Sender = &PrometheusSender{}

func NewPrometheusSender() *PrometheusSender {
	presenter := presenters.NewSnakeCasePresenter()

	return &PrometheusSender{
		presenter,
		make(map[string]appInstanceMetrics),
	}
}

func (s *PrometheusSender) Gauge(metric metrics.GaugeMetric) error {
	// name := s.presenter.Present(metric.Name())
	//
	// appInstanceMetrics := s.getOrCreateAppInstanceMetrics(metric.GUID, metric.Instance)
	//
	// gaugeVec, present := appInstanceMetrics.gaugeVecs[name]
	// labelNames := s.buildLabelsFromMetric(metric)
	//
	// if !present {
	// 	options := prometheus.GaugeOpts{
	// 		Name: name,
	// 		Help: " ",
	// 		ConstLabels: prometheus.Labels{
	// 			"guid": metric.GUID,
	// 			"instance": metric.Instance,
	// 		},
	// 	}
	// 	gaugeVec = *prometheus.NewGaugeVec(options, labelNames)
	//
	// 	prometheus.MustRegister(gaugeVec)
	// 	appInstanceMetrics.gaugeVecs[name] = gaugeVec
	// }
	//
	// labels := s.labels(metric, labelNames)
	// value := float64(metric.Value)
	//
	// gaugeVec.With(labels).Set(value)

	return nil
}

func (s *PrometheusSender) Incr(metric metrics.CounterMetric) error {
	name := s.presenter.Present(metric.Name())

	appInstanceMetrics := s.getOrCreateAppInstanceMetrics(metric.GUID, metric.Instance)

	counterVec := appInstanceMetrics.counterVecs[name]
	labelNames := s.buildLabelsFromMetric(metric)

	// if !present {
	// 	options := prometheus.CounterOpts{
	// 		Name: name,
	// 		Help: " ",
	// 		ConstLabels: prometheus.Labels{
	// 			"guid": metric.GUID,
	// 			"instance": metric.Instance,
	// 		},
	// 	}
	// 	counterVec = *prometheus.NewCounterVec(options, labelNames)
	//
	// 	prometheus.MustRegister(counterVec)
	// 	appInstanceMetrics.counterVecs[name] = counterVec
	// }

	labels := s.labels(metric, labelNames)
	value := float64(metric.Value)

	counterVec.With(labels).Add(value)

	return nil
}

func (s *PrometheusSender) PrecisionTiming(metric metrics.PrecisionTimingMetric) error {
	// name := s.presenter.Present(metric.Name())
	//
	// appInstanceMetrics := s.getOrCreateAppInstanceMetrics(metric.GUID, metric.Instance)
	//
	// histogramVec, present := appInstanceMetrics.histogramVecs[name]
	// labelNames := s.buildLabelsFromMetric(metric)
	//
	// if !present {
	// 	options := prometheus.HistogramOpts{
	// 		Name: name,
	// 		Help: " ",
	// 		ConstLabels: prometheus.Labels{
	// 			"guid": metric.GUID,
	// 			"instance": metric.Instance,
	// 		},
	// 	}
	// 	histogramVec = *prometheus.NewHistogramVec(options, labelNames)
	//
	// 	prometheus.MustRegister(histogramVec)
	// 	appInstanceMetrics.histogramVecs[name] = histogramVec
	// }
	//
	// labels := s.labels(metric, labelNames)
	// value := float64(metric.Value) / float64(time.Second)
	//
	// histogramVec.With(labels).Observe(value)

	return nil
}

func (s *PrometheusSender) getOrCreateAppInstanceMetrics(guid string, instance string) appInstanceMetrics {
	guidInstance := fmt.Sprintf("%s:%s", guid, instance)

	m := s.appInstanceMetrics[guidInstance]
	// if !present {
	// 	newM := appInstanceMetrics{
	// 		counterVecs:   make(map[string]prometheus.CounterVec),
	// 		gaugeVecs:     make(map[string]prometheus.GaugeVec),
	// 		histogramVecs: make(map[string]prometheus.HistogramVec),
	// 	}
	// 	s.appInstanceMetrics[guidInstance] = newM
	// 	return newM
	// }
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
	parts := strings.Split(guidInstance, ":")
	guid, instance := parts[0], parts[1]

	appInstanceMetrics := appInstanceMetrics{
		counterVecs: make(map[string]prometheus.CounterVec),
		//		gaugeVecs:     make(map[string]prometheus.GaugeVec),
		//		histogramVecs: make(map[string]prometheus.HistogramVec),
	}

	crashOptions := prometheus.CounterOpts{
		Name: "crash",
		Help: " ",
		ConstLabels: prometheus.Labels{
			"guid":     guid,
			"instance": instance,
		},
	}

	crashCounterVec := *prometheus.NewCounterVec(crashOptions, []string{"app", "organisation", "space"})
	prometheus.MustRegister(crashCounterVec)

	appInstanceMetrics.counterVecs["crash"] = crashCounterVec
	crashCounterVec.With(prometheus.Labels{"app": "foo", "space": "bar", "organisation": "baz"})

	options := prometheus.CounterOpts{
		Name: "requests",
		Help: " ",
		ConstLabels: prometheus.Labels{
			"guid":     guid,
			"instance": instance,
		},
	}

	requestsCounterVec := *prometheus.NewCounterVec(options, []string{"app", "organisation", "space", "status_range"})
	prometheus.MustRegister(requestsCounterVec)

	appInstanceMetrics.counterVecs["requests"] = requestsCounterVec

	s.appInstanceMetrics[guidInstance] = appInstanceMetrics
	return nil
}

func (s PrometheusSender) AppInstanceDeleted(guidInstance string) error {
	appInstanceMetrics := s.appInstanceMetrics[guidInstance]
	for _, v := range appInstanceMetrics.counterVecs {
		v.Reset()
	}
	for _, v := range appInstanceMetrics.gaugeVecs {
		v.Reset()
	}
	for _, v := range appInstanceMetrics.histogramVecs {
		v.Reset()
	}
	delete(s.appInstanceMetrics, guidInstance)
	return nil
}
