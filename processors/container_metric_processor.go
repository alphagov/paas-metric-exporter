package processors

import (
	"errors"
	"strconv"

	"github.com/alphagov/paas-cf-apps-statsd/events"
	"github.com/alphagov/paas-cf-apps-statsd/metrics"
)

type ContainerMetricProcessor struct {
	tmpl string
}

func NewContainerMetricProcessor(tmpl string) *ContainerMetricProcessor {
	return &ContainerMetricProcessor{tmpl: tmpl}
}

func (p *ContainerMetricProcessor) Process(appEvent *events.AppEvent) ([]metrics.Metric, error) {
	processedMetrics := make([]metrics.Metric, 3)
	var err error

	for i, metricType := range []string{"cpu", "mem", "dsk"} {
		processedMetrics[i], err = p.ProcessContainerMetric(metricType, appEvent)
		if err != nil {
			return nil, err
		}
	}
	return processedMetrics, nil
}

func (p *ContainerMetricProcessor) ProcessContainerMetric(metricType string, appEvent *events.AppEvent) (metrics.GaugeMetric, error) {
	containerMetricEvent := appEvent.Envelope.GetContainerMetric()
	instanceIndex := strconv.Itoa(int(containerMetricEvent.GetInstanceIndex()))

	var err error
	var metric metrics.GaugeMetric
	var newMetric string
	var value int64

	mv := metrics.Vars{}
	mv.Parse(appEvent)

	mv.Instance = instanceIndex

	switch metricType {
	case "cpu":
		mv.Metric = "cpu"
		newMetric, err = mv.Compose(p.tmpl)
		value = int64(containerMetricEvent.GetCpuPercentage())
	case "mem":
		mv.Metric = "memoryBytes"
		newMetric, err = mv.Compose(p.tmpl)
		value = int64(containerMetricEvent.GetMemoryBytes())
	case "dsk":
		mv.Metric = "diskBytes"
		newMetric, err = mv.Compose(p.tmpl)
		value = int64(containerMetricEvent.GetDiskBytes())
	default:
		err = errors.New("Unsupported metric type.")
	}

	if err != nil {
		return metrics.GaugeMetric{}, err
	}

	metric = *metrics.NewGaugeMetric(newMetric, value)

	return metric, nil
}
